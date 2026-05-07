package zathura

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lmist/pdfdb/internal/store"
)

func TestParseProcessesAndMatchOpenDocuments(t *testing.T) {
	t.Parallel()
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

func TestCloseReturnsAfterSignal(t *testing.T) {
	// t.Parallel omitted: manipulates OS processes
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	start := time.Now()
	err := closeOpenDocuments(context.Background(), []OpenDocument{{Slug: "paper", PID: cmd.Process.Pid, Path: "/tmp/paper.pdf"}})
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("Close took %s, expected immediate return", elapsed)
	}
}
