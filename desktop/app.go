package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/lmist/pdfdb/internal/doccache"
	"github.com/lmist/pdfdb/internal/profiles"
	"github.com/lmist/pdfdb/internal/store"
	"github.com/lmist/pdfdb/internal/zathura"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	profile *profiles.Manager
	cache   *doccache.Manager
	mu      sync.Mutex
	docs    map[string]store.Document
}

type State struct {
	Profiles  []profiles.Profile     `json:"profiles"`
	Documents []DocumentState        `json:"documents"`
	Open      []zathura.OpenDocument `json:"open"`
	Health    doccache.Health        `json:"health"`
	NeedsDB   bool                   `json:"needsDb"`
	Error     string                 `json:"error,omitempty"`
}

type DocumentState struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Filename  string `json:"filename"`
	SourceURL string `json:"sourceUrl,omitempty"`
	SizeBytes int64  `json:"sizeBytes"`
	PageCount int    `json:"pageCount"`
	Open      bool   `json:"open"`
}

func NewApp() (*App, error) {
	mgr, err := profiles.NewDefault()
	if err != nil {
		return nil, fmt.Errorf("profile manager: %w", err)
	}
	cache, err := doccache.NewDefault()
	if err != nil {
		return nil, fmt.Errorf("document cache: %w", err)
	}
	return &App{profile: mgr, cache: cache, docs: map[string]store.Document{}}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if _, err := a.profile.ActiveURL(); err == nil {
		go a.warmCacheBackground()
	}
}

func (a *App) warmCacheBackground() {
	if err := a.WarmCache(); err != nil {
		slog.Warn("warm cache", "err", err)
	}
}

func (a *App) GetState() State {
	state := State{}
	state.Profiles, _ = a.profile.List()
	st, err := a.openStore()
	if err != nil {
		state.NeedsDB = len(state.Profiles) == 0
		state.Error = err.Error()
		state.Health = doccache.Health{CacheDir: a.cache.Dir(), Message: "No database selected"}
		return state
	}
	defer st.Close()
	docs, err := st.ListDocuments(a.ctx)
	if err != nil {
		state.Error = err.Error()
		return state
	}
	a.setDocs(docs)
	state.Health = a.cache.Health(docs)
	if !state.Health.Ready {
		go a.warmDocs(docs)
	}
	open, err := zathura.OpenDocuments(a.ctx, a.cache.Paths(docs))
	if err != nil {
		state.Error = err.Error()
	}
	state.Open = open
	openBySlug := map[string]bool{}
	for _, item := range open {
		openBySlug[item.Slug] = true
	}
	for _, doc := range docs {
		state.Documents = append(state.Documents, DocumentState{
			ID:        doc.ID.String(),
			Slug:      doc.Slug,
			Title:     doc.Title,
			Filename:  doc.Filename,
			SourceURL: doc.SourceURL,
			SizeBytes: doc.SizeBytes,
			PageCount: doc.PageCount,
			Open:      openBySlug[doc.Slug],
		})
	}
	return state
}

func (a *App) SaveProfile(name, databaseURL string) error {
	if strings.TrimSpace(name) == "" {
		name = profiles.DefaultProfileName
	}
	if err := a.profile.Save(name, strings.TrimSpace(databaseURL)); err != nil {
		return err
	}
	st, err := a.openStore()
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Init(a.ctx); err != nil {
		return err
	}
	go a.warmCacheBackground()
	return nil
}

func (a *App) SetActiveProfile(name string) error {
	if err := a.profile.SetActive(name); err != nil {
		return err
	}
	a.setDocs(nil)
	go a.warmCacheBackground()
	return nil
}

func (a *App) DeleteProfile(name string) error {
	a.setDocs(nil)
	return a.profile.Delete(name)
}

func (a *App) WarmCache() error {
	st, err := a.openStore()
	if err != nil {
		return err
	}
	defer st.Close()
	docs, err := st.ListDocuments(a.ctx)
	if err != nil {
		return err
	}
	a.cache.Warm(a.ctx, st, docs)
	return nil
}

func (a *App) OpenDocument(slug string) error {
	st, err := a.openStore()
	if err != nil {
		return err
	}
	defer st.Close()
	doc, ok := a.doc(slug)
	if !ok {
		loaded, err := st.GetDocument(a.ctx, slug)
		if err != nil {
			return err
		}
		doc = loaded
	}
	path, err := a.cache.Ensure(a.ctx, st, *doc)
	if err != nil {
		return err
	}
	return zathura.Open(a.ctx, path)
}

func (a *App) CloseDocument(slug string) error {
	doc, ok := a.doc(slug)
	if !ok {
		return fmt.Errorf("document %q is not loaded", slug)
	}
	path := a.cache.Path(*doc)
	return zathura.Close(a.ctx, doc.Slug, path)
}

func (a *App) IngestSource(source string) (*DocumentState, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, errors.New("URL or file path is required")
	}
	return a.ingest(source)
}

func (a *App) PickAndIngestFile() (*DocumentState, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Import PDF",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "PDF documents (*.pdf)", Pattern: "*.pdf"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	return a.ingest(path)
}

func (a *App) ingest(source string) (*DocumentState, error) {
	st, err := a.openStore()
	if err != nil {
		return nil, err
	}
	defer st.Close()
	doc, err := st.Ingest(a.ctx, source)
	if err != nil {
		return nil, fmt.Errorf("ingest failed: %w", err)
	}
	if _, err := a.cache.Ensure(a.ctx, st, *doc); err != nil {
		return nil, fmt.Errorf("cache imported PDF: %w", err)
	}
	out := &DocumentState{
		ID:        doc.ID.String(),
		Slug:      doc.Slug,
		Title:     doc.Title,
		Filename:  doc.Filename,
		SourceURL: doc.SourceURL,
		SizeBytes: doc.SizeBytes,
		PageCount: doc.PageCount,
	}
	return out, nil
}

func (a *App) openStore() (*store.Store, error) {
	databaseURL, err := a.profile.ActiveURL()
	if err != nil {
		return nil, err
	}
	return store.Open(a.ctx, databaseURL)
}

func (a *App) warmDocs(docs []store.Document) {
	st, err := a.openStore()
	if err != nil {
		return
	}
	defer st.Close()
	a.cache.Warm(a.ctx, st, docs)
}

func (a *App) setDocs(docs []store.Document) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.docs = map[string]store.Document{}
	for _, doc := range docs {
		a.docs[doc.Slug] = doc
		a.docs[doc.ID.String()] = doc
	}
}

func (a *App) doc(slug string) (*store.Document, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	doc, ok := a.docs[slug]
	return &doc, ok
}
