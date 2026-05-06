package store

const SchemaSQL = `
create extension if not exists pgcrypto;

create table if not exists documents (
  id uuid primary key default gen_random_uuid(),
  slug text not null unique,
  title text not null,
  filename text not null,
  mime text not null default 'application/pdf',
  source_url text unique,
  sha256 text not null unique,
  size_bytes bigint not null check (size_bytes > 0),
  page_count int not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists chunks (
  sha256 text primary key,
  size_bytes int not null check (size_bytes > 0),
  data bytea not null,
  created_at timestamptz not null default now()
);

create table if not exists document_chunks (
  document_id uuid not null references documents(id) on delete cascade,
  ordinal int not null,
  chunk_sha256 text not null references chunks(sha256),
  byte_start bigint not null,
  byte_end bigint not null,
  primary key (document_id, ordinal),
  unique (document_id, chunk_sha256, ordinal),
  check (byte_start >= 0),
  check (byte_end > byte_start)
);

create index if not exists document_chunks_document_range_idx
  on document_chunks (document_id, byte_start, byte_end);

create table if not exists reading_state (
  document_id uuid primary key references documents(id) on delete cascade,
  page int not null default 1,
  zoom text not null default 'auto',
  mode text not null default 'fit-width',
  updated_at timestamptz not null default now()
);

create table if not exists annotations (
  id uuid primary key default gen_random_uuid(),
  document_id uuid not null references documents(id) on delete cascade,
  page int not null,
  kind text not null,
  anchor jsonb not null default '{}'::jsonb,
  text_snippet text,
  note text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists ingest_jobs (
  id uuid primary key default gen_random_uuid(),
  source text not null,
  status text not null,
  error text,
  byte_count bigint not null default 0,
  sha256 text,
  document_id uuid references documents(id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
`
