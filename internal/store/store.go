package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/lmist/pdfdb/internal/pdfmeta"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ChunkSize is the per-chunk byte budget for PDF ingest. 2 MiB is small
// enough to fit a single bytea TOAST page comfortably and large enough that
// a 50 MB paper produces only ~25 chunks. Override with PDFDB_CHUNK_SIZE.
var ChunkSize = chunkSizeFromEnv(2 * 1024 * 1024)

func chunkSizeFromEnv(def int64) int64 {
	v := strings.TrimSpace(os.Getenv("PDFDB_CHUNK_SIZE"))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

type Store struct {
	pool *pgxpool.Pool
}

type Document struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Filename  string    `json:"filename"`
	Mime      string    `json:"mime"`
	SourceURL string    `json:"sourceUrl,omitempty"`
	SHA256    string    `json:"sha256"`
	SizeBytes int64     `json:"sizeBytes"`
	PageCount int       `json:"pageCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type chunkRef struct {
	SHA256    string
	SizeBytes int
	Start     int64
	End       int64
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) Init(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, SchemaSQL)
	return err
}

func (s *Store) Ingest(ctx context.Context, source string) (*Document, error) {
	reader, filename, canonicalSource, err := openSource(ctx, source)
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var jobID uuid.UUID
	if err := tx.QueryRow(ctx, `insert into ingest_jobs (source, status) values ($1, 'running') returning id`, source).Scan(&jobID); err != nil {
		return nil, err
	}

	var all bytes.Buffer
	fullHash := sha256.New()
	tee := io.TeeReader(reader, io.MultiWriter(fullHash, &all))

	var refs []chunkRef
	buf := make([]byte, int(ChunkSize))
	var offset int64
	for {
		n, readErr := io.ReadFull(tee, buf)
		if readErr == io.ErrUnexpectedEOF || readErr == io.EOF {
			if n > 0 {
				ref, err := insertChunk(ctx, tx, buf[:n], offset)
				if err != nil {
					markJobFailed(ctx, tx, jobID, err)
					return nil, err
				}
				refs = append(refs, ref)
				offset += int64(n)
			}
			break
		}
		if readErr != nil {
			markJobFailed(ctx, tx, jobID, readErr)
			return nil, readErr
		}
		ref, err := insertChunk(ctx, tx, buf[:n], offset)
		if err != nil {
			markJobFailed(ctx, tx, jobID, err)
			return nil, err
		}
		refs = append(refs, ref)
		offset += int64(n)
	}
	if offset == 0 {
		err := errors.New("source produced zero bytes")
		markJobFailed(ctx, tx, jobID, err)
		return nil, err
	}

	sum := hex.EncodeToString(fullHash.Sum(nil))
	meta := pdfmeta.Extract(all.Bytes(), filename, canonicalSource)
	if meta.Title == "" {
		meta.Title = filename
	}
	slug := uniqueSlug(ctx, tx, pdfmeta.Slug(meta.Title), sum)

	var doc Document
	err = tx.QueryRow(ctx, `
insert into documents (slug, title, filename, mime, source_url, sha256, size_bytes, page_count)
values ($1, $2, $3, 'application/pdf', $4, $5, $6, $7)
on conflict (sha256) do update set
  source_url = coalesce(documents.source_url, excluded.source_url),
  updated_at = now()
returning id, slug, title, filename, mime, coalesce(source_url, ''), sha256, size_bytes, page_count, created_at, updated_at
`, slug, meta.Title, filename, nullIfEmpty(canonicalSource), sum, offset, meta.PageCount).Scan(
		&doc.ID, &doc.Slug, &doc.Title, &doc.Filename, &doc.Mime, &doc.SourceURL, &doc.SHA256, &doc.SizeBytes, &doc.PageCount, &doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		markJobFailed(ctx, tx, jobID, err)
		return nil, err
	}

	if _, err := tx.Exec(ctx, `delete from document_chunks where document_id = $1`, doc.ID); err != nil {
		markJobFailed(ctx, tx, jobID, err)
		return nil, err
	}
	batch := &pgx.Batch{}
	for i, ref := range refs {
		batch.Queue(`
insert into document_chunks (document_id, ordinal, chunk_sha256, byte_start, byte_end)
values ($1, $2, $3, $4, $5)
`, doc.ID, i, ref.SHA256, ref.Start, ref.End)
	}
	br := tx.SendBatch(ctx, batch)
	for range refs {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			markJobFailed(ctx, tx, jobID, err)
			return nil, err
		}
	}
	if err := br.Close(); err != nil {
		markJobFailed(ctx, tx, jobID, err)
		return nil, err
	}

	if _, err := tx.Exec(ctx, `
update ingest_jobs
set status = 'completed', byte_count = $2, sha256 = $3, document_id = $4, updated_at = now()
where id = $1
`, jobID, offset, sum, doc.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &doc, nil
}

func insertChunk(ctx context.Context, tx pgx.Tx, data []byte, start int64) (chunkRef, error) {
	hash := sha256.Sum256(data)
	sum := hex.EncodeToString(hash[:])
	copyData := append([]byte(nil), data...)
	_, err := tx.Exec(ctx, `
insert into chunks (sha256, size_bytes, data)
values ($1, $2, $3)
on conflict (sha256) do nothing
`, sum, len(copyData), copyData)
	return chunkRef{SHA256: sum, SizeBytes: len(copyData), Start: start, End: start + int64(len(copyData))}, err
}

func markJobFailed(ctx context.Context, tx pgx.Tx, jobID uuid.UUID, err error) {
	if _, txErr := tx.Exec(ctx, `update ingest_jobs set status = 'failed', error = $2, updated_at = now() where id = $1`, jobID, err.Error()); txErr != nil {
		slog.Error("mark ingest job failed", "jobID", jobID, "cause", err, "txErr", txErr)
	}
}

func (s *Store) ListDocuments(ctx context.Context) ([]Document, error) {
	rows, err := s.pool.Query(ctx, `
select id, slug, title, filename, mime, coalesce(source_url, ''), sha256, size_bytes, page_count, created_at, updated_at
from documents
order by created_at asc
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.Slug, &doc.Title, &doc.Filename, &doc.Mime, &doc.SourceURL, &doc.SHA256, &doc.SizeBytes, &doc.PageCount, &doc.CreatedAt, &doc.UpdatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

func (s *Store) GetDocument(ctx context.Context, key string) (*Document, error) {
	var doc Document
	err := s.pool.QueryRow(ctx, `
select id, slug, title, filename, mime, coalesce(source_url, ''), sha256, size_bytes, page_count, created_at, updated_at
from documents
where id::text = $1 or slug = $1 or filename = $1
`, key).Scan(&doc.ID, &doc.Slug, &doc.Title, &doc.Filename, &doc.Mime, &doc.SourceURL, &doc.SHA256, &doc.SizeBytes, &doc.PageCount, &doc.CreatedAt, &doc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Store) ReadRange(ctx context.Context, docID uuid.UUID, start, end int64) ([]byte, error) {
	if start < 0 || end <= start {
		return nil, fmt.Errorf("invalid byte range %d-%d", start, end)
	}
	rows, err := s.pool.Query(ctx, `
select dc.byte_start, dc.byte_end, c.data
from document_chunks dc
join chunks c on c.sha256 = dc.chunk_sha256
where dc.document_id = $1 and dc.byte_end > $2 and dc.byte_start < $3
order by dc.ordinal
`, docID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out bytes.Buffer
	for rows.Next() {
		var chunkStart, chunkEnd int64
		var data []byte
		if err := rows.Scan(&chunkStart, &chunkEnd, &data); err != nil {
			return nil, err
		}
		from := max(start, chunkStart) - chunkStart
		to := min(end, chunkEnd) - chunkStart
		out.Write(data[from:to])
	}
	return out.Bytes(), rows.Err()
}

func (s *Store) Reconstruct(ctx context.Context, docID uuid.UUID) ([]byte, error) {
	doc, err := s.GetDocument(ctx, docID.String())
	if err != nil {
		return nil, err
	}
	return s.ReadRange(ctx, doc.ID, 0, doc.SizeBytes)
}

func (s *Store) Verify(ctx context.Context) error {
	docs, err := s.ListDocuments(ctx)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		if err := s.verifyOne(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) verifyOne(ctx context.Context, doc Document) error {
	rows, err := s.pool.Query(ctx, `
select ordinal, byte_start, byte_end
from document_chunks
where document_id = $1
order by ordinal
`, doc.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var expected int64
	var ord int
	for rows.Next() {
		var ordinal int
		var start, end int64
		if err := rows.Scan(&ordinal, &start, &end); err != nil {
			return err
		}
		if ordinal != ord || start != expected || end <= start {
			return fmt.Errorf("%s has invalid manifest at ordinal %d", doc.Slug, ordinal)
		}
		expected = end
		ord++
	}
	if expected != doc.SizeBytes {
		return fmt.Errorf("%s manifest ends at %d, expected %d", doc.Slug, expected, doc.SizeBytes)
	}
	data, err := s.Reconstruct(ctx, doc.ID)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != doc.SHA256 {
		return fmt.Errorf("%s checksum mismatch", doc.Slug)
	}
	return nil
}

func openSource(ctx context.Context, source string) (io.ReadCloser, string, string, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		url := canonicalPDFURL(source)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", "", err
		}
		res, err := httpClient.Do(req)
		if err != nil {
			return nil, "", "", err
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			_ = res.Body.Close()
			return nil, "", "", fmt.Errorf("download failed: %s", res.Status)
		}
		return res.Body, filenameFromURL(url), url, nil
	}
	f, err := os.Open(source)
	if err != nil {
		return nil, "", "", err
	}
	return f, filepath.Base(source), "", nil
}

func canonicalPDFURL(source string) string {
	re := regexp.MustCompile(`^https://arxiv\.org/abs/([^?#]+)`)
	if m := re.FindStringSubmatch(source); len(m) == 2 {
		return "https://arxiv.org/pdf/" + m[1]
	}
	return source
}

func filenameFromURL(url string) string {
	base := filepath.Base(strings.Split(url, "?")[0])
	if base == "" || base == "." || base == "/" {
		return "document.pdf"
	}
	if !strings.HasSuffix(strings.ToLower(base), ".pdf") {
		base += ".pdf"
	}
	return base
}

func uniqueSlug(ctx context.Context, tx pgx.Tx, base string, sum string) string {
	if base == "" {
		base = "document"
	}
	var exists bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from documents where slug = $1 and sha256 <> $2)`, base, sum).Scan(&exists); err != nil || !exists {
		return base
	}
	return fmt.Sprintf("%s-%s", base, sum[:8])
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

