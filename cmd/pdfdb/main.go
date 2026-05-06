package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pdffuse "github.com/lmist/pdfdb/internal/fuse"
	"github.com/lmist/pdfdb/internal/server"
	"github.com/lmist/pdfdb/internal/store"
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
	case "mount":
		err = cmdMount(ctx, st, args)
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

func openStore(ctx context.Context) (*store.Store, error) {
	return store.Open(ctx, os.Getenv("DATABASE_URL"))
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
	return server.New(st).ListenAndServe(addr)
}

func cmdMount(ctx context.Context, st *store.Store, args []string) error {
	if len(args) != 1 {
		return errors.New("mount needs a mountpoint")
	}
	fmt.Printf("mounting read-only pdfdb filesystem at %s\n", args[0])
	return pdffuse.New(st).Mount(ctx, expandHome(args[0]))
}

func cmdOpen(ctx context.Context, st *store.Store, args []string) error {
	if len(args) == 0 || len(args) > 2 {
		return errors.New("open needs a document id/slug and optional mountpoint")
	}
	if args[0] == "all" {
		return cmdOpenAll(ctx, st, args[1:])
	}
	mountpoint := "~/Mounts/pdfdb"
	if len(args) == 2 {
		mountpoint = args[1]
	}
	doc, err := st.GetDocument(ctx, args[0])
	if err != nil {
		return err
	}
	path := filepath.Join(expandHome(mountpoint), doc.Slug+".pdf")
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("mounted PDF %s is not available; run `pdfdb mount %s` in another shell first: %w", path, mountpoint, err)
	}
	zathura, err := exec.LookPath("zathura")
	if err != nil {
		return errors.New("zathura is not on PATH")
	}
	cmd := exec.CommandContext(ctx, zathura, path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func cmdOpenAll(ctx context.Context, st *store.Store, args []string) error {
	if len(args) > 1 {
		return errors.New("open-all accepts an optional mountpoint")
	}
	mountpoint := "~/Mounts/pdfdb"
	if len(args) == 1 {
		mountpoint = args[0]
	}
	docs, err := st.ListDocuments(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return errors.New("no documents found")
	}
	zathura, err := exec.LookPath("zathura")
	if err != nil {
		return errors.New("zathura is not on PATH")
	}
	expandedMount := expandHome(mountpoint)
	argv := make([]string, 0, len(docs))
	for _, doc := range docs {
		path := filepath.Join(expandedMount, doc.Slug+".pdf")
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("mounted PDF %s is not available; run `pdfdb mount %s` in another shell first: %w", path, mountpoint, err)
		}
		argv = append(argv, path)
	}
	cmd := exec.CommandContext(ctx, zathura, argv...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
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

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cacheDir = filepath.Join(cacheDir, "pdfdb", "zathura")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	paths := make([]string, 0, len(selected))
	for _, doc := range selected {
		data, err := st.Reconstruct(ctx, doc.ID)
		if err != nil {
			return err
		}
		path := filepath.Join(cacheDir, doc.Slug+"-"+doc.SHA256[:8]+".pdf")
		tmpPath := path + ".tmp"
		if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
			return err
		}
		if err := os.Rename(tmpPath, path); err != nil {
			return err
		}
		if err := os.Chmod(path, 0o444); err != nil {
			return err
		}
		paths = append(paths, path)
	}

	_ = exec.CommandContext(ctx, "open", "-a", "Zathura").Run()
	time.Sleep(600 * time.Millisecond)
	_ = exec.CommandContext(ctx, "open", "-a", "Zathura", paths[0]).Run()
	time.Sleep(600 * time.Millisecond)

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
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func usage() {
	fmt.Println(`pdfdb commands:
  init-db                      create or update Postgres schema
  ingest <url-or-path> [...]   import PDFs into chunked Postgres storage
  seed                         ingest the three DBOS seed PDFs
  list                         list documents
  verify                       verify manifests and reconstructed SHA-256 values
  serve [host:port]            run the range-capable API
  mount <mountpoint>           mount read-only macFUSE filesystem
  open <id-or-slug|all> [mount] open mounted PDF(s) in Zathura
  open-all [mount]              open every mounted PDF in Zathura
  zathura [id-or-slug|all]      start /Applications/Zathura.app from DB cache
  zathura-pick                  choose a database PDF and open it in Zathura`)
}

func expandHome(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "pdfdb: %v\n", err)
	time.Sleep(10 * time.Millisecond)
	os.Exit(1)
}
