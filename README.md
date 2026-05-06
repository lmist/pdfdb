![pdfdb logo](assets/pdfdb-logo.svg)

# pdfdb

Database-backed PDF reader workspace. Postgres stores PDF bytes as content-addressed chunks. Zathura reads PDFs through a read-only macFUSE mount, and the web UI reads through the range-capable Go API.

## Prerequisites

- Go 1.26+
- Bun
- Neon CLI authenticated on `$PATH`
- Zathura
- macFUSE for `pdfdb mount`

## Quick Start

```sh
go run ./cmd/pdfdb init-db
go run ./cmd/pdfdb ingest https://arxiv.org/abs/2007.11112
go run ./cmd/pdfdb ingest https://vldb.org/pvldb/vol15/p21-skiadopoulos.pdf
go run ./cmd/pdfdb ingest https://petereliaskraft.net/res/dbos-provenance.pdf
go run ./cmd/pdfdb verify
go run ./cmd/pdfdb serve
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
go run ./cmd/pdfdb mount ~/Mounts/pdfdb
zathura ~/Mounts/pdfdb/dbos-a-proposal-for-a-data-centric-operating-system.pdf
```
