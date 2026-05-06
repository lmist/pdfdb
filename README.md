![pdfdb logo](assets/pdfdb-logo.svg)

# pdfdb

Database-backed PDF reader workspace. Postgres stores PDF bytes as content-addressed chunks. Zathura reads PDFs through a read-only macFUSE mount, and the web UI reads through the range-capable Go API.

## Prerequisites

- Go 1.26+
- Bun
- Neon CLI authenticated on `$PATH`
- Zathura
- macFUSE for `pdfdb mount`

Install macFUSE on macOS:

```sh
brew install --cask macfuse
```

## Install

```sh
go install github.com/lmist/pdfdb/cmd/pdfdb@latest
```

## Quick Start

```sh
pdfdb init-db
pdfdb seed
pdfdb list
pdfdb verify
pdfdb serve
```

In another shell:

```sh
cd web
bun install
bun run dev
```

For Zathura:

```sh
mkdir -p ~/Mounts/pdfdb
pdfdb mount ~/Mounts/pdfdb
pdfdb open-all
```

Zathura receives the mounted PDFs as normal files:

```sh
zathura \
  ~/Mounts/pdfdb/dbos-a-proposal-for-a-data-centric-operating-system.pdf \
  ~/Mounts/pdfdb/dbos-a-dbms-oriented-operating-system.pdf \
  ~/Mounts/pdfdb/dbos-provenance.pdf
```

Use Zathura's normal bindings to move through them: `J`/`K` for next/previous page, `gg`/`G` for first/last page, `Tab` for index mode, and `:open <path>` for another mounted PDF.

No custom Zathura plugin is needed for this path: the macFUSE mount exposes normal `application/pdf` files, so Zathura's PDF plugin handles rendering. A custom plugin becomes useful only if pdfdb should teach Zathura a new URI or MIME type such as `pdfdb://document/<id>` instead of mounted file paths. Zathura plugin development starts with `ZATHURA_PLUGIN_REGISTER` and a shared-object plugin that registers supported MIME types: https://pwmt.org/projects/zathura/plugins/development/

## Commands

```text
pdfdb init-db                      create or update the Neon/Postgres schema
pdfdb ingest <url-or-path> [...]   import PDFs into chunked database storage
pdfdb seed                         ingest the three DBOS seed PDFs
pdfdb list                         list documents with ids, slugs, sizes, and page counts
pdfdb verify                       reconstruct all PDFs from chunks and verify SHA-256
pdfdb serve [host:port]            run the HTTP API with PDF range support
pdfdb mount <mountpoint>           mount a read-only database-backed PDF filesystem
pdfdb open <id-or-slug|all> [mount] open mounted PDF(s) in Zathura
pdfdb open-all [mount]             open every mounted PDF in Zathura
```
