package doccache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmist/pdfdb/internal/store"
)

func TestPathUsesSlugAndChecksum(t *testing.T) {
	t.Parallel()
	mgr := New(t.TempDir())
	doc := store.Document{
		ID:        uuid.New(),
		Slug:      "dbos-provenance",
		Filename:  "paper.pdf",
		SHA256:    "abcdef1234567890",
		SizeBytes: 123,
		CreatedAt: time.Now(),
	}
	got := mgr.Path(doc)
	want := filepath.Join(mgr.Dir(), "dbos-provenance-abcdef123456.pdf")
	if got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
}

func TestHealthCountsValidCachedFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mgr := New(dir)
	docs := []store.Document{
		{ID: uuid.New(), Slug: "one", SHA256: "111111111111", SizeBytes: 3},
		{ID: uuid.New(), Slug: "two", SHA256: "222222222222", SizeBytes: 5},
	}
	if err := osWriteFile(mgr.Path(docs[0]), []byte("abc")); err != nil {
		t.Fatal(err)
	}
	h := mgr.Health(docs)
	if h.Ready || h.Cached != 1 || h.Total != 2 {
		t.Fatalf("health = %#v", h)
	}
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o444)
}
