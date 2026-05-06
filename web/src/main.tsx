import React, { useEffect, useMemo, useRef, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BookOpen, ChevronLeft, ChevronRight, RefreshCw } from 'lucide-react';
import * as pdfjs from 'pdfjs-dist';
import type { PDFDocumentProxy } from 'pdfjs-dist';
import './styles.css';

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  'pdfjs-dist/build/pdf.worker.mjs',
  import.meta.url,
).toString();

type DocumentRow = {
  id: string;
  slug: string;
  title: string;
  filename: string;
  sourceUrl?: string;
  sha256: string;
  sizeBytes: number;
  pageCount: number;
};

const apiBase = import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8787';

function App() {
  const [documents, setDocuments] = useState<DocumentRow[]>([]);
  const [selected, setSelected] = useState<DocumentRow | null>(null);
  const [pdf, setPdf] = useState<PDFDocumentProxy | null>(null);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState('Loading library');

  async function loadDocuments() {
    setStatus('Loading library');
    const response = await fetch(`${apiBase}/api/documents`);
    if (!response.ok) throw new Error(await response.text());
    const rows = (await response.json()) as DocumentRow[];
    setDocuments(rows);
    setSelected((current) => current ?? rows[0] ?? null);
    setStatus(rows.length ? 'Ready' : 'No PDFs ingested');
  }

  useEffect(() => {
    loadDocuments().catch((error) => setStatus(error.message));
  }, []);

  useEffect(() => {
    if (!selected) return;
    setStatus(`Opening ${selected.title}`);
    setPage(1);
    const task = pdfjs.getDocument({
      url: `${apiBase}/api/documents/${selected.id}/file`,
      rangeChunkSize: 65536,
      disableAutoFetch: false,
    });
    task.promise
      .then((loaded) => {
        setPdf(loaded);
        setStatus('Ready');
      })
      .catch((error) => setStatus(error.message));
    return () => {
      task.destroy();
    };
  }, [selected]);

  const currentUrl = useMemo(() => {
    return selected ? `${apiBase}/api/documents/${selected.id}/file` : '';
  }, [selected]);

  return (
    <main className="app">
      <aside className="library" aria-label="PDF library">
        <div className="toolbar">
          <BookOpen size={18} />
          <span>pdfdb</span>
          <button title="Refresh library" onClick={() => loadDocuments()}>
            <RefreshCw size={16} />
          </button>
        </div>
        <nav>
          {documents.map((doc) => (
            <button
              className={selected?.id === doc.id ? 'active' : ''}
              key={doc.id}
              onClick={() => setSelected(doc)}
            >
              <strong>{doc.title}</strong>
              <span>
                {formatBytes(doc.sizeBytes)} · {doc.pageCount || '?'} pages
              </span>
            </button>
          ))}
        </nav>
      </aside>

      <section className="reader">
        <header>
          <div>
            <h1>{selected?.title ?? 'No document selected'}</h1>
            <p>{status}</p>
          </div>
          <div className="controls">
            <button title="Previous page" disabled={!pdf || page <= 1} onClick={() => setPage(page - 1)}>
              <ChevronLeft size={18} />
            </button>
            <span>{pdf ? `${page} / ${pdf.numPages}` : '-'}</span>
            <button title="Next page" disabled={!pdf || page >= pdf.numPages} onClick={() => setPage(page + 1)}>
              <ChevronRight size={18} />
            </button>
          </div>
        </header>
        {pdf && <PageCanvas pdf={pdf} page={page} />}
        {!pdf && <div className="empty">Start `pdfdb serve`, ingest PDFs, then refresh.</div>}
        {currentUrl && (
          <a className="raw" href={currentUrl} target="_blank" rel="noreferrer">
            Open raw PDF endpoint
          </a>
        )}
      </section>
    </main>
  );
}

function PageCanvas({ pdf, page }: { pdf: PDFDocumentProxy; page: number }) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    pdf.getPage(page).then((pdfPage) => {
      if (cancelled || !canvasRef.current) return;
      const containerWidth = Math.min(window.innerWidth - 360, 980);
      const viewport = pdfPage.getViewport({ scale: 1 });
      const scale = Math.max(0.7, containerWidth / viewport.width);
      const scaled = pdfPage.getViewport({ scale });
      const canvas = canvasRef.current;
      const context = canvas.getContext('2d');
      if (!context) return;
      canvas.width = scaled.width;
      canvas.height = scaled.height;
      pdfPage.render({ canvasContext: context, viewport: scaled });
    });
    return () => {
      cancelled = true;
    };
  }, [pdf, page]);

  return <canvas ref={canvasRef} className="page" />;
}

function formatBytes(bytes: number) {
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

createRoot(document.getElementById('root')!).render(<App />);
