package doccache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lmist/pdfdb/internal/store"
)

type Manager struct {
	dir string
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
	return &Manager{dir: filepath.Join(dir, "pdfdb", "documents")}, nil
}

func New(dir string) *Manager {
	return &Manager{dir: dir}
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

func (m *Manager) valid(path string, doc store.Document) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() == doc.SizeBytes
}
