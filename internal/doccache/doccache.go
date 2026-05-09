package doccache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lmist/pdfdb/internal/store"
)

type Manager struct {
	dir string

	verifyMu sync.Mutex
	verified map[string]verifyEntry

	ensureLocks sync.Map // path -> *sync.Mutex
}

type verifyEntry struct {
	mtime time.Time
	size  int64
	sha   string
}

type Health struct {
	Ready    bool   `json:"ready"`
	Cached   int    `json:"cached"`
	Total    int    `json:"total"`
	CacheDir string `json:"cacheDir"`
	Message  string `json:"message"`
}

func NewDefault() (*Manager, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return New(filepath.Join(dir, "pdfdb", "documents")), nil
}

func New(dir string) *Manager {
	return &Manager{dir: dir, verified: map[string]verifyEntry{}}
}

func (m *Manager) Dir() string {
	return m.dir
}

func (m *Manager) Path(doc store.Document) string {
	sum := doc.SHA256
	if len(sum) > 12 {
		sum = sum[:12]
	}
	name := doc.Slug
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(doc.Filename, filepath.Ext(doc.Filename))
	}
	name = strings.Trim(name, ". ")
	if name == "" {
		name = "document"
	}
	return filepath.Join(m.dir, fmt.Sprintf("%s-%s.pdf", name, sum))
}

func (m *Manager) Ensure(ctx context.Context, st *store.Store, doc store.Document) (string, error) {
	path := m.Path(doc)
	mu := m.ensureLock(path)
	mu.Lock()
	defer mu.Unlock()
	if m.valid(path, doc) {
		return path, nil
	}
	if err := os.MkdirAll(m.dir, 0o700); err != nil {
		return "", err
	}
	data, err := st.Reconstruct(ctx, doc.ID)
	if err != nil {
		return "", err
	}
	if int64(len(data)) != doc.SizeBytes {
		return "", fmt.Errorf("%s reconstructed to %d bytes, expected %d", doc.Slug, len(data), doc.SizeBytes)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != doc.SHA256 {
		return "", fmt.Errorf("%s cache checksum mismatch", doc.Slug)
	}
	tmp, err := os.CreateTemp(m.dir, ".pdfdb-*.pdf")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chmod(tmpPath, 0o444); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return path, nil
}

func (m *Manager) Warm(ctx context.Context, st *store.Store, docs []store.Document) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for _, doc := range docs {
		if m.valid(m.Path(doc), doc) {
			continue
		}
		doc := doc
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}
			_, _ = m.Ensure(ctx, st, doc)
		}()
	}
	wg.Wait()
}

func (m *Manager) Health(docs []store.Document) Health {
	h := Health{Ready: true, Total: len(docs), CacheDir: m.dir}
	for _, doc := range docs {
		if m.valid(m.Path(doc), doc) {
			h.Cached++
		}
	}
	switch {
	case h.Total == 0:
		h.Message = "No PDFs in this database"
	case h.Cached == h.Total:
		h.Message = "Local cache ready"
	default:
		h.Ready = false
		h.Message = fmt.Sprintf("Warming cache %d/%d", h.Cached, h.Total)
	}
	return h
}

func (m *Manager) Paths(docs []store.Document) map[string]string {
	paths := make(map[string]string, len(docs))
	for _, doc := range docs {
		paths[doc.Slug] = m.Path(doc)
	}
	return paths
}

func (m *Manager) ensureLock(path string) *sync.Mutex {
	if v, ok := m.ensureLocks.Load(path); ok {
		return v.(*sync.Mutex)
	}
	v, _ := m.ensureLocks.LoadOrStore(path, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (m *Manager) valid(path string, doc store.Document) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() != doc.SizeBytes {
		return false
	}
	m.verifyMu.Lock()
	cached, hit := m.verified[path]
	m.verifyMu.Unlock()
	if hit && cached.mtime.Equal(info.ModTime()) && cached.size == info.Size() {
		return cached.sha == doc.SHA256
	}
	sum, err := hashFile(path)
	if err != nil {
		return false
	}
	m.verifyMu.Lock()
	m.verified[path] = verifyEntry{mtime: info.ModTime(), size: info.Size(), sha: sum}
	m.verifyMu.Unlock()
	return sum == doc.SHA256
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
