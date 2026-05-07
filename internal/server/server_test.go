package server

import "testing"

func TestParseRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		header  string
		size    int64
		start   int64
		end     int64
		partial bool
		wantErr bool
	}{
		{name: "full", size: 100, start: 0, end: 100},
		{name: "explicit", header: "bytes=10-19", size: 100, start: 10, end: 20, partial: true},
		{name: "open ended", header: "bytes=90-", size: 100, start: 90, end: 100, partial: true},
		{name: "suffix", header: "bytes=-10", size: 100, start: 90, end: 100, partial: true},
		{name: "clamped", header: "bytes=90-999", size: 100, start: 90, end: 100, partial: true},
		{name: "outside", header: "bytes=100-101", size: 100, wantErr: true},
		{name: "invalid", header: "items=1-2", size: 100, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, partial, err := parseRange(tt.header, tt.size)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if start != tt.start || end != tt.end || partial != tt.partial {
				t.Fatalf("got %d %d %v, want %d %d %v", start, end, partial, tt.start, tt.end, tt.partial)
			}
		})
	}
}
