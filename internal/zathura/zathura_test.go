package zathura

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmist/pdfdb/internal/store"
)

func TestParseProcessesAndMatchOpenDocuments(t *testing.T) {
	processes := ParseProcesses([]byte(`  101 /Applications/Zathura.app/Contents/MacOS/zathura-bin -T --fork /Users/lou/Library/Caches/pdfdb/documents/dbos-provenance-abc123.pdf
  102 /bin/zsh
  103 /Applications/Zathura.app/Contents/MacOS/zathura-bin /tmp/other.pdf
`))
	docs := []store.Document{
		{ID: uuid.New(), Slug: "dbos-provenance", Title: "DBOS Provenance", CreatedAt: time.Now()},
		{ID: uuid.New(), Slug: "other", Title: "Other", CreatedAt: time.Now()},
	}
	paths := map[string]string{}
	for _, doc := range docs {
		paths[doc.Slug] = "/Users/lou/Library/Caches/pdfdb/documents/" + doc.Slug + "-abc123.pdf"
	}
	open := MatchOpenDocuments(processes, paths)
	if len(open) != 1 {
		t.Fatalf("open = %#v", open)
	}
	if open[0].PID != 101 || open[0].Slug != "dbos-provenance" {
		t.Fatalf("open[0] = %#v", open[0])
	}
}
