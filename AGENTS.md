# pdfdb Agent Guide

## What This Is

`pdfdb` is a database-backed PDF library and reader controller. Neon/Postgres is the source of truth for PDF bytes, stored as content-addressed chunks. Zathura reads normal local PDF files reconstructed into a disposable cache; do not reintroduce macFUSE or a mount daemon.

The primary user-facing app is `PDF DB.app`, a tiny fixed-portrait Wails desktop app for macOS. It lists/searches database PDFs, imports PDFs from URLs or local files, warms the cache, opens PDFs in the real Zathura app, highlights currently open PDFs, and closes matching Zathura processes.

## Quick Start (One Command)

**Desktop app** (requires Wails + Go + Bun + active database profile):
```sh
cd desktop && wails dev
```

**CLI server** (requires Go + DATABASE_URL in .env):
```sh
go run ./cmd/pdfdb serve
```

CI runs on every PR: `go test ./...`, `golangci-lint`, and `bun run build` for the desktop frontend.
See `.github/workflows/ci.yml`.

## Layout

- `cmd/pdfdb`: CLI entrypoint for schema setup, ingest, list, verify, HTTP serving, cached Zathura opens, and Keychain-backed profile management.
- `desktop`: Wails v2 desktop app. Go backend lives in `desktop/app.go`; frontend lives in `desktop/frontend`.
- `internal/store`: Postgres schema access, chunked PDF ingest, range reads, reconstruction, and verification.
- `internal/doccache`: immutable local cache files under `~/Library/Caches/pdfdb/documents`.
- `internal/profiles`: database profile config plus macOS Keychain storage for database URLs.
- `internal/zathura`: Zathura process detection, cached-file open, and immediate close signaling.
- `internal/server`: range-capable HTTP API used by external clients.

## Build And Test

Use these checks before committing:

```sh
go test -count=1 ./...
golangci-lint run ./...
cd desktop/frontend && bun run test
cd desktop && wails build
```

Frontend-only desktop build and tests:

```sh
cd desktop/frontend
bun install
bun run build
bun run test
```

Tests are run in CI on every PR (see `.github/workflows/ci.yml`).

### Test Conventions

- **Go**: test files use `*_test.go` suffix. Top-level test functions use `t.Parallel()` where safe (no shared OS resources). Test packages match the package under test.
- **TypeScript (desktop)**: test files use `*.test.ts` suffix, co-located with source. Vitest with `describe`/`it` blocks. Utility functions are extracted to `src/utils.ts` so they can be tested independently of the DOM/API bridge.
- **Coverage**: Go CI runs `go test -coverprofile=coverage.out -covermode=atomic`. Desktop frontend CI runs `bun run test` (vitest coverage configurable via `vitest.config.ts`).
```

Lint and format desktop frontend:

```sh
cd desktop/frontend
bun run lint
bun run format
```

Pre-commit hooks (runs all lint/format/checks automatically):

```sh
pre-commit install
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
- Frontend dependencies: `desktop/frontend/node_modules/` is ignored.
- `.env` is local only and ignored; never commit database URLs or Neon credentials.

## Naming Conventions

- **Go**: exported identifiers are PascalCase (e.g., `LoadDotEnv`), unexported are camelCase (e.g., `openSource`). Package names are lowercase, single-word where possible. Error variables follow the `errXxx` or `XxxError` pattern.
- **TypeScript**: variables and functions use camelCase. React components use PascalCase (e.g., `App`, `PageCanvas`). Types and interfaces use PascalCase. Private class members may use a leading underscore. Constants may use UPPER_CASE.
- **Files**: Go test files use `*_test.go`. TypeScript source files are lowercase with hyphens or camelCase.

## Important Decisions

- No macFUSE. The app and CLI must use cache-backed normal files for Zathura.
- The cache is disposable. Postgres remains durable source of truth.
- Desktop close should be instant: send the signal and return; let UI polling refresh open state.
- Zathura plugin work is not needed for this architecture unless adding a custom URI/MIME integration later.
- Keep the app small and portrait-oriented; `desktop/main.go` fixes the Wails window at `360 x 620`.

## Git Hygiene

- Do not use `git add .`; stage explicit files.
- Keep generated Wails bindings and `desktop/frontend/dist` in sync after `wails build`.
- Before committing, run `git diff --check` to catch generated trailing whitespace.
