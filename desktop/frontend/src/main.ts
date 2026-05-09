import './styles.css';
import { disabled, errorText, escapeAttr, escapeHTML, formatBytes } from './utils';

type Profile = {
  name: string;
  active: boolean;
};

type Health = {
  ready: boolean;
  cached: number;
  total: number;
  cacheDir: string;
  message: string;
};

type DocumentState = {
  id: string;
  slug: string;
  title: string;
  filename: string;
  sourceUrl?: string;
  sizeBytes: number;
  pageCount: number;
  open: boolean;
};

type AppState = {
  profiles: Profile[];
  documents: DocumentState[];
  health: Health;
  needsDb: boolean;
  error?: string;
};

declare global {
  interface Window {
    go?: {
      main?: {
        App?: Record<string, (...args: unknown[]) => Promise<unknown>>;
      };
    };
  }
}

const api = window.go?.main?.App;
const app = document.querySelector<HTMLElement>('#app');

let state: AppState = emptyState();
let query = '';
let busy = '';
let status = '';
let showSettings = false;
let profileName = 'default';
let databaseURL = '';
let ingestSource = '';

if (!app) {
  throw new Error('missing app root');
}
const root = app;

function emptyState(): AppState {
  return {
    profiles: [],
    documents: [],
    health: {
      ready: false,
      cached: 0,
      total: 0,
      cacheDir: '',
      message: 'loading',
    },
    needsDb: true,
  };
}

function call<T>(name: string, ...args: unknown[]): Promise<T> {
  const fn = api?.[name];
  if (!fn) {
    return Promise.reject(new Error(`desktop bridge is not ready: ${name}`));
  }
  return fn(...args) as Promise<T>;
}

async function refresh(silent = false) {
  try {
    if (!silent) status = 'Refreshing';
    const next = await call<AppState>('GetState');
    const changed = JSON.stringify(next) !== JSON.stringify(state);
    state = next;
    if (state.needsDb) showSettings = true;
    if (!silent) status = '';
    if (changed || !silent) render();
  } catch (error) {
    state.error = errorText(error);
    if (!silent) status = '';
    render();
  }
}

async function run(label: string, work: () => Promise<unknown>) {
  busy = label;
  status = label;
  render();
  try {
    await work();
    status = 'Done';
    await refresh(true);
  } catch (error) {
    status = errorText(error);
  } finally {
    busy = '';
    render();
  }
}

function filteredDocuments() {
  const q = query.trim().toLowerCase();
  if (!q) return state.documents;
  return state.documents.filter((doc) => {
    return [doc.title, doc.slug, doc.filename, doc.sourceUrl ?? '']
      .join(' ')
      .toLowerCase()
      .includes(q);
  });
}

const ICON_SETTINGS = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>`;

const ICON_CLOSE = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`;

const ICON_SEARCH = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="7"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>`;

function render() {
  const docs = filteredDocuments();
  const active = state.profiles.find((profile) => profile.active);
  const footerLabel =
    status ||
    `${state.documents.length} ${state.documents.length === 1 ? 'document' : 'documents'}`;
  root.innerHTML = `
    <section class="shell">
      <header class="titlebar">
        <div class="brand">
          <strong>PDF DB</strong>
          <span class="profile-name">${escapeHTML(active?.name ?? 'No database')}</span>
        </div>
        <button class="icon" data-action="settings" title="Database settings">${ICON_SETTINGS}</button>
      </header>

      <section class="health ${state.health.ready ? 'ok' : 'warn'}">
        <span class="dot"></span>
        <p>${escapeHTML(state.health.message || 'Checking')}</p>
        <button data-action="warm" ${disabled(busy)}>Warm cache</button>
      </section>

      <section class="search">
        ${ICON_SEARCH}
        <input id="search" type="search" placeholder="Search documents" value="${escapeAttr(query)}" autocomplete="off" spellcheck="false" aria-label="Search documents" />
      </section>

      <section class="list" aria-label="PDFs">
        ${docs.map(renderDocument).join('') || '<p class="empty">No documents yet.<br/>Paste a URL below to import.</p>'}
      </section>

      <section class="ingest">
        <input id="ingest" type="text" placeholder="Paste URL or drop a path" value="${escapeAttr(ingestSource)}" autocomplete="off" spellcheck="false" aria-label="Import URL or path" />
        <button data-action="ingest-url" ${disabled(busy)}>Import</button>
        <button data-action="ingest-file" ${disabled(busy)}>Browse</button>
      </section>

      <footer class="${busy ? 'busy' : ''}">
        <span class="pulse"></span>
        <span>${escapeHTML(footerLabel)}</span>
      </footer>
    </section>
    ${showSettings ? renderSettings() : ''}
  `;
  bind();
}

function renderDocument(doc: DocumentState) {
  const subtitle = doc.pageCount ? `${doc.pageCount} pages` : doc.filename;
  return `
    <article class="row ${doc.open ? 'open' : ''}" data-slug="${escapeAttr(doc.slug)}" title="Double-click to open">
      <div class="glyph">PDF</div>
      <div class="meta">
        <h2>${escapeHTML(doc.title || doc.filename || doc.slug)}</h2>
        <p>${escapeHTML(subtitle)} &middot; ${formatBytes(doc.sizeBytes)}</p>
      </div>
      <div class="actions">
        ${doc.open ? `<button class="icon close-btn" data-action="close" data-slug="${escapeAttr(doc.slug)}" title="Close in Zathura">${ICON_CLOSE}</button>` : ''}
      </div>
    </article>
  `;
}

function renderSettings() {
  return `
    <aside class="modal">
      <section class="panel">
        <header>
          <strong>Database</strong>
          <button class="icon" data-action="settings" title="Close">${ICON_CLOSE}</button>
        </header>
        <label>
          <span>Profile</span>
          <input id="profile-name" value="${escapeAttr(profileName)}" autocomplete="off" spellcheck="false" />
        </label>
        <label>
          <span>Postgres URL</span>
          <input id="database-url" type="password" value="${escapeAttr(databaseURL)}" placeholder="postgresql://..." autocomplete="off" spellcheck="false" />
        </label>
        <button class="primary" data-action="save-profile" ${disabled(busy)}>Save to Keychain</button>
        ${
          state.profiles.length
            ? `
          <div class="profiles">
            ${state.profiles
              .map(
                (profile) => `
              <button data-action="switch-profile" data-name="${escapeAttr(profile.name)}" class="${profile.active ? 'active' : ''}">
                ${escapeHTML(profile.name)}
              </button>
            `,
              )
              .join('')}
          </div>
        `
            : ''
        }
      </section>
    </aside>
  `;
}

let searchTimer: number | undefined;

function triggerIngestUrl() {
  const source = ingestSource;
  void run('Importing', async () => {
    await call('IngestSource', source);
    ingestSource = '';
  });
}

function bind() {
  document.querySelector<HTMLInputElement>('#search')?.addEventListener('input', (event) => {
    query = (event.target as HTMLInputElement).value;
    if (searchTimer !== undefined) clearTimeout(searchTimer);
    searchTimer = window.setTimeout(() => {
      searchTimer = undefined;
      render();
    }, 150);
  });
  const ingestEl = document.querySelector<HTMLInputElement>('#ingest');
  ingestEl?.addEventListener('input', (event) => {
    ingestSource = (event.target as HTMLInputElement).value;
  });
  ingestEl?.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' && !busy) {
      event.preventDefault();
      triggerIngestUrl();
    }
  });
  document.querySelector<HTMLInputElement>('#profile-name')?.addEventListener('input', (event) => {
    profileName = (event.target as HTMLInputElement).value;
  });
  document.querySelector<HTMLInputElement>('#database-url')?.addEventListener('input', (event) => {
    databaseURL = (event.target as HTMLInputElement).value;
  });
  document.querySelectorAll<HTMLElement>('[data-action]').forEach((element) => {
    element.addEventListener('click', onAction);
  });
  document.querySelectorAll<HTMLElement>('.row').forEach((row) => {
    row.addEventListener('click', (e) => {
      if ((e.target as HTMLElement).closest('[data-action]')) return;
      const slug = row.dataset.slug;
      if (slug) void run('Opening', () => call('OpenDocument', slug));
    });
  });
}

function onAction(event: Event) {
  const element = event.currentTarget as HTMLElement;
  const action = element.dataset.action;
  if (action === 'settings') {
    showSettings = !showSettings;
    render();
    return;
  }
  if (action === 'warm') {
    void run('Warming cache', () => call('WarmCache'));
    return;
  }
  if (action === 'save-profile') {
    void run('Saving profile', async () => {
      await call('SaveProfile', profileName, databaseURL);
      databaseURL = '';
      showSettings = false;
    });
    return;
  }
  if (action === 'switch-profile') {
    const name = element.dataset.name;
    if (name) void run('Switching database', () => call('SetActiveProfile', name));
    return;
  }
  if (action === 'close') {
    const slug = element.dataset.slug;
    if (slug) void run('Closing', () => call('CloseDocument', slug));
    return;
  }
  if (action === 'ingest-url') {
    triggerIngestUrl();
    return;
  }
  if (action === 'ingest-file') {
    void run('Importing file', () => call('PickAndIngestFile'));
  }
}

void refresh();
const pollHandle = window.setInterval(() => void refresh(true), 1000);

window.addEventListener('beforeunload', () => {
  window.clearInterval(pollHandle);
  if (searchTimer !== undefined) clearTimeout(searchTimer);
});

window.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && showSettings) {
    showSettings = false;
    render();
  }
});
