# pdfdb Agent Guide

## What This Is

`pdfdb` is a database-backed PDF library and reader controller. Neon/Postgres is the source of truth for PDF bytes, stored as content-addressed chunks. Zathura reads normal local PDF files reconstructed into a disposable cache; do not reintroduce macFUSE or a mount daemon.

The primary user-facing app is `PDF DB.app`, a tiny fixed-portrait Wails desktop app for macOS. It lists/searches database PDFs, imports PDFs from URLs or local files, warms the cache, opens PDFs in the real Zathura app, highlights currently open PDFs, and closes matching Zathura processes.

## Layout

- `cmd/pdfdb`: CLI entrypoint for schema setup, ingest, list, verify, HTTP serving, cached Zathura opens, and Keychain-backed profile management.
- `desktop`: Wails v2 desktop app. Go backend lives in `desktop/app.go`; frontend lives in `desktop/frontend`.
- `internal/store`: Postgres schema access, chunked PDF ingest, range reads, reconstruction, and verification.
- `internal/doccache`: immutable local cache files under `~/Library/Caches/pdfdb/documents`.
- `internal/profiles`: database profile config plus macOS Keychain storage for database URLs.
- `internal/zathura`: Zathura process detection, cached-file open, and immediate close signaling.
- `internal/server`: range-capable HTTP API used by the web reader.
- `web`: Bun/Vite web reader.

## Build And Test

Use these checks before committing:

```sh
go test ./...
cd desktop && wails build
```

Frontend-only desktop build:

```sh
cd desktop/frontend
bun install
bun run build
```

Web reader:

```sh
cd web
bun install
bun run dev
```

Wails must be installed for desktop work:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails doctor
```

## Running

CLI install:

```sh
go install github.com/lmist/pdfdb/cmd/pdfdb@latest
```

Typical local setup:

```sh
pdfdb init-db
pdfdb seed
pdfdb verify
pdfdb profile save default
```

Build and open the desktop app:

```sh
cd desktop
wails build
open "build/bin/PDF DB.app"
```

Open from CLI without FUSE:

```sh
pdfdb open <slug-or-id>
pdfdb open-all
```

## Local Storage

- Database URL secrets: macOS Keychain, service `pdfdb`, profile accounts managed by `internal/profiles`.
- Non-secret profile metadata: user config dir, `pdfdb/profiles.json`.
- PDF cache: `~/Library/Caches/pdfdb/documents`.
- Desktop build output: `desktop/build/bin/` is ignored.
- Frontend dependencies: `desktop/frontend/node_modules/` and `web/node_modules/` are ignored.
- `.env` is local only and ignored; never commit database URLs or Neon credentials.

## Important Decisions

- No macFUSE. The app and CLI must use cache-backed normal files for Zathura.
- The cache is disposable. Postgres remains durable source of truth.
- Desktop close should be instant: send the signal and return; let UI polling refresh open state.
- Zathura plugin work is not needed for this architecture unless adding a custom URI/MIME integration later.
- Keep the app small and portrait-oriented; `desktop/main.go` fixes the Wails window at `320 x 520`.

## Git Hygiene

- Do not use `git add .`; stage explicit files.
- Keep generated Wails bindings and `desktop/frontend/dist` in sync after `wails build`.
- Before committing, run `git diff --check` to catch generated trailing whitespace.
