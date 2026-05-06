package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"pdfdb/internal/store"
)

type Server struct {
	store *store.Store
	mux   *http.ServeMux
}

func New(st *store.Store) *Server {
	s := &Server{store: st, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	slog.Info("serving pdfdb api", "addr", addr)
	return http.ListenAndServe(addr, s)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Range, Content-Type")
	w.Header().Set("Access-Control-Expose-Headers", "Accept-Ranges, Content-Length, Content-Range")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	s.mux.HandleFunc("GET /api/documents", s.handleDocuments)
	s.mux.HandleFunc("GET /api/documents/{key}", s.handleDocument)
	s.mux.HandleFunc("GET /api/documents/{key}/file", s.handleFile)
}

func (s *Server) handleDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := s.store.ListDocuments(r.Context())
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, docs)
}

func (s *Server) handleDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocument(r.Context(), r.PathValue("key"))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, doc)
}

func (s *Server) handleFile(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocument(r.Context(), r.PathValue("key"))
	if err != nil {
		writeError(w, err, http.StatusNotFound)
		return
	}

	start, end, partial, err := parseRange(r.Header.Get("Range"), doc.SizeBytes)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", doc.SizeBytes))
		writeError(w, err, http.StatusRequestedRangeNotSatisfiable)
		return
	}

	data, err := s.store.ReadRange(r.Context(), doc.ID, start, end)
	if err != nil {
		writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, sanitizeHeader(doc.Filename)))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if partial {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end-1, doc.SizeBytes))
		w.WriteHeader(http.StatusPartialContent)
	}
	_, _ = w.Write(data)
}

func parseRange(header string, size int64) (int64, int64, bool, error) {
	if header == "" {
		return 0, size, false, nil
	}
	re := regexp.MustCompile(`^bytes=(\d*)-(\d*)$`)
	m := re.FindStringSubmatch(header)
	if len(m) != 3 {
		return 0, 0, false, errors.New("invalid range")
	}
	if m[1] == "" && m[2] == "" {
		return 0, 0, false, errors.New("invalid range")
	}
	if m[1] == "" {
		suffix, err := strconv.ParseInt(m[2], 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, false, errors.New("invalid suffix range")
		}
		if suffix > size {
			suffix = size
		}
		return size - suffix, size, true, nil
	}
	start, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, false, errors.New("range start outside document")
	}
	end := size - 1
	if m[2] != "" {
		end, err = strconv.ParseInt(m[2], 10, 64)
		if err != nil || end < start {
			return 0, 0, false, errors.New("invalid range end")
		}
		if end >= size {
			end = size - 1
		}
	}
	return start, end + 1, true, nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func sanitizeHeader(value string) string {
	return strings.ReplaceAll(value, `"`, `'`)
}

func ShutdownContext() context.Context {
	return context.Background()
}
