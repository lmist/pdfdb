package doccache

import (
	"crypto/sha256"
	"encoding/hex"
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
	one := []byte("abc")
	two := []byte("hello")
	docs := []store.Document{
		{ID: uuid.New(), Slug: "one", SHA256: hashHex(one), SizeBytes: int64(len(one))},
		{ID: uuid.New(), Slug: "two", SHA256: hashHex(two), SizeBytes: int64(len(two))},
	}
	if err := osWriteFile(mgr.Path(docs[0]), one); err != nil {
		t.Fatal(err)
	}
	h := mgr.Health(docs)
	if h.Ready || h.Cached != 1 || h.Total != 2 {
		t.Fatalf("health = %#v", h)
	}
}

func TestValidRejectsCorruptFileWithMatchingSize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mgr := New(dir)
	good := []byte("good")
	doc := store.Document{ID: uuid.New(), Slug: "doc", SHA256: hashHex(good), SizeBytes: int64(len(good))}
	if err := osWriteFile(mgr.Path(doc), []byte("bad!")); err != nil {
		t.Fatal(err)
	}
	if mgr.valid(mgr.Path(doc), doc) {
		t.Fatal("valid() returned true for a same-size, wrong-content file")
	}
}

func hashHex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o444)
}
