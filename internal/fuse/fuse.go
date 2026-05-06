package fuse

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"pdfdb/internal/store"

	gofuse "github.com/hanwen/go-fuse/v2/fuse"
	"github.com/hanwen/go-fuse/v2/fs"
)

type FS struct {
	store *store.Store
}

func New(st *store.Store) *FS {
	return &FS{store: st}
}

func (f *FS) Mount(ctx context.Context, mountpoint string) error {
	if err := requireFUSE(); err != nil {
		return err
	}
	if err := os.MkdirAll(mountpoint, 0o755); err != nil {
		return err
	}
	root := &rootNode{store: f.store}
	server, err := fs.Mount(mountpoint, root, &fs.Options{
		MountOptions: gofuse.MountOptions{
			Name:   "pdfdb",
			FsName: "pdfdb",
		},
	})
	if err != nil {
		return err
	}
	done := make(chan struct{})
	go func() {
		server.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		_ = server.Unmount()
		<-done
		return ctx.Err()
	case <-done:
		return nil
	}
}

func requireFUSE() error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	for _, path := range []string{
		"/Library/Filesystems/macfuse.fs",
		"/Library/Filesystems/osxfuse.fs",
	} {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	}
	return errors.New("macFUSE is required for pdfdb mount; install it from https://macfuse.github.io/ and retry")
}

type rootNode struct {
	fs.Inode
	store *store.Store
}

func (r *rootNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	docs, err := r.store.ListDocuments(ctx)
	if err != nil {
		return nil, syscall.EIO
	}
	entries := make([]gofuse.DirEntry, 0, len(docs))
	for _, doc := range docs {
		entries = append(entries, gofuse.DirEntry{
			Name: fileName(doc),
			Mode: gofuse.S_IFREG,
		})
	}
	return fs.NewListDirStream(entries), 0
}

func (r *rootNode) Lookup(ctx context.Context, name string, out *gofuse.EntryOut) (*fs.Inode, syscall.Errno) {
	docs, err := r.store.ListDocuments(ctx)
	if err != nil {
		return nil, syscall.EIO
	}
	for _, doc := range docs {
		if fileName(doc) == name {
			stable := fs.StableAttr{Mode: gofuse.S_IFREG, Ino: inodeFor(doc.SHA256)}
			child := r.NewInode(ctx, &fileNode{store: r.store, doc: doc}, stable)
			out.Attr.Mode = gofuse.S_IFREG | 0o444
			out.Attr.Size = uint64(doc.SizeBytes)
			return child, 0
		}
	}
	return nil, syscall.ENOENT
}

type fileNode struct {
	fs.Inode
	store *store.Store
	doc   store.Document
}

func (f *fileNode) Getattr(ctx context.Context, fh fs.FileHandle, out *gofuse.AttrOut) syscall.Errno {
	out.Mode = gofuse.S_IFREG | 0o444
	out.Size = uint64(f.doc.SizeBytes)
	return 0
}

func (f *fileNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	if flags&syscall.O_ACCMODE != syscall.O_RDONLY {
		return nil, 0, syscall.EROFS
	}
	return &fileHandle{store: f.store, doc: f.doc}, gofuse.FOPEN_KEEP_CACHE, 0
}

type fileHandle struct {
	store *store.Store
	doc   store.Document
}

func (h *fileHandle) Read(ctx context.Context, dest []byte, off int64) (gofuse.ReadResult, syscall.Errno) {
	if off >= h.doc.SizeBytes {
		return gofuse.ReadResultData(nil), 0
	}
	end := off + int64(len(dest))
	if end > h.doc.SizeBytes {
		end = h.doc.SizeBytes
	}
	data, err := h.store.ReadRange(ctx, h.doc.ID, off, end)
	if err != nil {
		return nil, syscall.EIO
	}
	return gofuse.ReadResultData(data), 0
}

func fileName(doc store.Document) string {
	name := doc.Slug
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSuffix(doc.Filename, filepath.Ext(doc.Filename))
	}
	return fmt.Sprintf("%s.pdf", name)
}

func inodeFor(sum string) uint64 {
	var out uint64 = 1469598103934665603
	for i := 0; i < len(sum); i++ {
		out ^= uint64(sum[i])
		out *= 1099511628211
	}
	if out == 1 {
		return 2
	}
	return out
}
