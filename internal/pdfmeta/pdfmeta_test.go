package pdfmeta

import "testing"

func TestSlug(t *testing.T) {
	t.Parallel()
	got := Slug("DBOS: A DBMS-oriented Operating System")
	want := "dbos-a-dbms-oriented-operating-system"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKnownSourceTitle(t *testing.T) {
	t.Parallel()
	meta := Extract([]byte(`/Type /Page /Type /Page`), "2007.11112.pdf", "https://arxiv.org/pdf/2007.11112")
	if meta.Title != "DBOS: A Proposal for a Data-Centric Operating System" {
		t.Fatalf("unexpected title %q", meta.Title)
	}
	if meta.PageCount != 2 {
		t.Fatalf("got %d pages, want 2", meta.PageCount)
	}
}
