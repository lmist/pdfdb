import './styles.css';

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
      message: 'loading'
    },
    needsDb: true
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
    state = await call<AppState>('GetState');
    if (state.needsDb) showSettings = true;
    if (!silent) status = '';
  } catch (error) {
    state.error = errorText(error);
    if (!silent) status = '';
  }
  render();
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

function render() {
  const docs = filteredDocuments();
  const active = state.profiles.find((profile) => profile.active);
  root.innerHTML = `
    <section class="shell">
      <header class="titlebar">
        <div>
          <strong>PDF DB</strong>
          <span>${escapeHTML(active?.name ?? 'No database')}</span>
        </div>
        <button class="icon" data-action="settings" title="Database settings">...</button>
      </header>

      <section class="health ${state.health.ready ? 'ok' : 'warn'}">
        <span></span>
        <p>${escapeHTML(state.health.message || 'Checking')}</p>
        <button data-action="warm" ${disabled()}>Warm</button>
      </section>

      <section class="search">
        <input id="search" type="search" placeholder="Search PDFs" value="${escapeAttr(query)}" />
      </section>

      <section class="list" aria-label="PDFs">
        ${docs.map(renderDocument).join('') || '<p class="empty">No PDFs in this database.</p>'}
      </section>

      <section class="ingest">
        <input id="ingest" type="text" placeholder="Paste URL or path" value="${escapeAttr(ingestSource)}" />
        <button data-action="ingest-url" ${disabled()}>Import</button>
        <button data-action="ingest-file" ${disabled()}>File</button>
      </section>

      <footer>
        <span>${escapeHTML(status || `${state.documents.length} PDFs`)}</span>
      </footer>
    </section>
    ${showSettings ? renderSettings() : ''}
  `;
  bind();
}

function renderDocument(doc: DocumentState) {
  return `
    <article class="row ${doc.open ? 'open' : ''}" data-slug="${escapeAttr(doc.slug)}" title="Double-click to open">
      <div class="mark"></div>
      <div class="meta">
        <h2>${escapeHTML(doc.title || doc.filename || doc.slug)}</h2>
        <p>${escapeHTML(doc.pageCount ? `${doc.pageCount} pages` : doc.filename)} · ${formatBytes(doc.sizeBytes)}</p>
      </div>
      <button data-action="close" data-slug="${escapeAttr(doc.slug)}" ${doc.open ? '' : 'disabled'} title="Close in Zathura">Close</button>
    </article>
  `;
}

function renderSettings() {
  return `
    <aside class="modal">
      <section class="panel">
        <header>
          <strong>Database</strong>
          <button class="icon" data-action="settings" title="Close">x</button>
        </header>
        <label>
          <span>Profile</span>
          <input id="profile-name" value="${escapeAttr(profileName)}" />
        </label>
        <label>
          <span>Postgres URL</span>
          <input id="database-url" type="password" value="${escapeAttr(databaseURL)}" placeholder="postgresql://..." />
        </label>
        <button class="primary" data-action="save-profile" ${disabled()}>Save to Keychain</button>
        <div class="profiles">
          ${state.profiles.map((profile) => `
            <button data-action="switch-profile" data-name="${escapeAttr(profile.name)}" class="${profile.active ? 'active' : ''}">
              ${escapeHTML(profile.name)}
            </button>
          `).join('')}
        </div>
      </section>
    </aside>
  `;
}

function bind() {
  document.querySelector<HTMLInputElement>('#search')?.addEventListener('input', (event) => {
    query = (event.target as HTMLInputElement).value;
    render();
  });
  document.querySelector<HTMLInputElement>('#ingest')?.addEventListener('input', (event) => {
    ingestSource = (event.target as HTMLInputElement).value;
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
    row.addEventListener('dblclick', () => {
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
    const source = ingestSource;
    void run('Importing', async () => {
      await call('IngestSource', source);
      ingestSource = '';
    });
    return;
  }
  if (action === 'ingest-file') {
    void run('Importing file', () => call('PickAndIngestFile'));
  }
}

function disabled() {
  return busy ? 'disabled' : '';
}

function errorText(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function formatBytes(bytes: number) {
  if (!bytes) return '0 B';
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function escapeHTML(value: string) {
  return value.replace(/[&<>"']/g, (char) => entity(char));
}

function escapeAttr(value: string) {
  return escapeHTML(value);
}

function entity(char: string) {
  const entities: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;'
  };
  return entities[char] ?? char;
}

void refresh();
setInterval(() => void refresh(true), 1000);
