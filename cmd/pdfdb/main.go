package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lmist/pdfdb/internal/doccache"
	"github.com/lmist/pdfdb/internal/profiles"
	"github.com/lmist/pdfdb/internal/server"
	"github.com/lmist/pdfdb/internal/store"
	"github.com/lmist/pdfdb/internal/zathura"
)

var seedURLs = []string{
	"https://arxiv.org/abs/2007.11112",
	"https://vldb.org/pvldb/vol15/p21-skiadopoulos.pdf",
	"https://petereliaskraft.net/res/dbos-provenance.pdf",
}

func main() {
	log.SetFlags(0)
	store.LoadDotEnv(".env")

	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	ctx := context.Background()
	cmd := os.Args[1]
	args := os.Args[2:]

	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		usage()
		return
	}
	if cmd == "profile" {
		if err := cmdProfile(args); err != nil {
			fatal(err)
		}
		return
	}

	st, err := openStore(ctx)
	if err != nil {
		fatal(err)
	}
	defer st.Close()

	switch cmd {
	case "init-db":
		err = st.Init(ctx)
	case "ingest":
		err = cmdIngest(ctx, st, args)
	case "seed":
		err = cmdIngest(ctx, st, seedURLs)
	case "list":
		err = cmdList(ctx, st)
	case "verify":
		err = cmdVerify(ctx, st)
	case "serve":
		err = cmdServe(st, args)
	case "open":
		err = cmdOpen(ctx, st, args)
	case "open-all":
		err = cmdOpenAll(ctx, st, args)
	case "zathura":
		err = cmdZathura(ctx, st, args)
	case "zathura-pick":
		err = cmdZathuraPick(ctx, st)
	default:
		err = fmt.Errorf("unknown command %q", cmd)
	}
	if err != nil {
		fatal(err)
	}
}

func cmdProfile(args []string) error {
	if len(args) == 0 {
		return errors.New("profile needs list, save, use, or delete")
	}
	mgr, err := profiles.NewDefault()
	if err != nil {
		return err
	}
	switch args[0] {
	case "list":
		items, err := mgr.List()
		if err != nil {
			return err
		}
		for _, item := range items {
			prefix := " "
			if item.Active {
				prefix = "*"
			}
			fmt.Printf("%s %s\n", prefix, item.Name)
		}
		return nil
	case "save":
		name := profiles.DefaultProfileName
		if len(args) > 1 {
			name = args[1]
		}
		databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
		if databaseURL == "" {
			return errors.New("DATABASE_URL is not set; refusing to save an empty profile")
		}
		if err := mgr.Save(name, databaseURL); err != nil {
			return err
		}
		fmt.Printf("saved active profile %q in macOS Keychain\n", name)
		return nil
	case "use":
		if len(args) != 2 {
			return errors.New("profile use needs a profile name")
		}
		return mgr.SetActive(args[1])
	case "delete":
		if len(args) != 2 {
			return errors.New("profile delete needs a profile name")
		}
		return mgr.Delete(args[1])
	default:
		return fmt.Errorf("unknown profile command %q", args[0])
	}
}

func openStore(ctx context.Context) (*store.Store, error) {
	databaseURL, err := profiles.ResolveDatabaseURL()
	if err != nil {
		return nil, err
	}
	return store.Open(ctx, databaseURL)
}

func cmdIngest(ctx context.Context, st *store.Store, args []string) error {
	if len(args) == 0 {
		return errors.New("ingest needs at least one URL or path")
	}
	for _, source := range args {
		doc, err := st.Ingest(ctx, source)
		if err != nil {
			return fmt.Errorf("ingest %s: %w", source, err)
		}
		fmt.Printf("ingested %s  %s  %d bytes  %d pages\n", doc.ID, doc.Title, doc.SizeBytes, doc.PageCount)
	}
	return nil
}

func cmdList(ctx context.Context, st *store.Store) error {
	docs, err := st.ListDocuments(ctx)
	if err != nil {
		return err
	}
	for _, doc := range docs {
		fmt.Printf("%s\t%s\t%d bytes\t%d pages\t%s\n", doc.ID, doc.Slug, doc.SizeBytes, doc.PageCount, doc.Title)
	}
	return nil
}

func cmdVerify(ctx context.Context, st *store.Store) error {
	if err := st.Verify(ctx); err != nil {
		return err
	}
	fmt.Println("ok: all documents reconstruct from chunk manifests and match SHA-256")
	return nil
}

func cmdServe(st *store.Store, args []string) error {
	addr := os.Getenv("PDFDB_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8787"
	}
	if len(args) > 0 {
		addr = args[0]
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		return fmt.Errorf("serve address must be host:port: %w", err)
	}

	srv := server.New(st)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe(addr) }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func cmdOpen(ctx context.Context, st *store.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("open needs a document id/slug or all")
	}
	if args[0] == "all" {
		return cmdOpenAll(ctx, st, nil)
	}
	doc, err := st.GetDocument(ctx, args[0])
	if err != nil {
		return err
	}
	cache, err := doccache.NewDefault()
	if err != nil {
		return err
	}
	path, err := cache.Ensure(ctx, st, *doc)
	if err != nil {
		return err
	}
	return zathura.Open(ctx, path)
}

func cmdOpenAll(ctx context.Context, st *store.Store, args []string) error {
	docs, err := st.ListDocuments(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return errors.New("no documents found")
	}
	cache, err := doccache.NewDefault()
	if err != nil {
		return err
	}
	for _, doc := range docs {
		path, err := cache.Ensure(ctx, st, doc)
		if err != nil {
			return err
		}
		if err := zathura.Open(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func cmdZathura(ctx context.Context, st *store.Store, args []string) error {
	docs, err := st.ListDocuments(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return errors.New("no documents found")
	}

	selected := docs
	if len(args) > 0 && args[0] != "all" {
		doc, err := st.GetDocument(ctx, args[0])
		if err != nil {
			return err
		}
		selected = []store.Document{*doc}
	}

	cache, err := doccache.NewDefault()
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(selected))
	for _, doc := range selected {
		path, err := cache.Ensure(ctx, st, doc)
		if err != nil {
			return err
		}
		paths = append(paths, path)
	}

	_ = exec.CommandContext(ctx, "open", "-a", "Zathura").Run()
	waitForZathura(ctx, 3*time.Second)
	_ = exec.CommandContext(ctx, "open", "-a", "Zathura", paths[0]).Run()
	waitForZathura(ctx, 3*time.Second)

	for _, path := range paths {
		cmd := exec.CommandContext(ctx, "open", "-a", "Zathura", path)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func cmdZathuraPick(ctx context.Context, st *store.Store) error {
	docs, err := st.ListDocuments(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return errors.New("no documents found")
	}

	items := make([]string, 0, len(docs))
	byLabel := make(map[string]string, len(docs))
	for _, doc := range docs {
		label := fmt.Sprintf("%s  (%d pages)", doc.Title, doc.PageCount)
		items = append(items, label)
		byLabel[label] = doc.Slug
	}

	script := `set picked to choose from list {` + strings.Join(appleScriptList(items), ", ") + `} with title "pdfdb" with prompt "Open a database PDF in Zathura" OK button name "Open" cancel button name "Cancel"` + "\n" +
		`if picked is false then return ""` + "\n" +
		`return item 1 of picked`
	out, err := exec.CommandContext(ctx, "osascript", "-e", script).Output()
	if err != nil {
		return err
	}
	label := strings.TrimSpace(string(out))
	if label == "" {
		return nil
	}
	slug, ok := byLabel[label]
	if !ok {
		return fmt.Errorf("unknown selection %q", label)
	}
	return cmdZathura(ctx, st, []string{slug})
}

func appleScriptList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, appleScriptString(value))
	}
	return out
}

func appleScriptString(value string) string {
	value = strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t':
			return ' '
		}
		return r
	}, value)
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

// waitForZathura polls `pgrep -x zathura` every 100 ms until it succeeds or
// the deadline elapses. macOS `open -a Zathura` returns immediately even when
// the app is still bootstrapping; subsequent open invocations against a
// not-yet-ready Zathura silently drop the file argument, so we must block
// until at least one zathura process exists.
func waitForZathura(ctx context.Context, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := exec.CommandContext(ctx, "pgrep", "-x", "zathura").Run(); err == nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func usage() {
	fmt.Println(`pdfdb commands:
  init-db                      create or update Postgres schema
  ingest <url-or-path> [...]   import PDFs into chunked Postgres storage
  seed                         ingest the three DBOS seed PDFs
  list                         list documents
  verify                       verify manifests and reconstructed SHA-256 values
  serve [host:port]            run the range-capable API
  open <id-or-slug|all>        open cached PDF(s) in Zathura
  open-all                     open every cached PDF in Zathura
  zathura [id-or-slug|all]     start /Applications/Zathura.app from DB cache
  zathura-pick                 choose a database PDF and open it in Zathura
  profile list|save|use|delete manage Keychain-backed database profiles`)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "pdfdb: %v\n", err)
	os.Exit(1)
}
