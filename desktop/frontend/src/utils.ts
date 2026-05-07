export function disabled(busy: string): string {
  return busy ? 'disabled' : '';
}

export function errorText(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function formatBytes(bytes: number): string {
  if (!bytes) return '0 B';
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

export function escapeHTML(value: string): string {
  return value.replace(/[&<>"']/g, (char) => entity(char));
}

export function escapeAttr(value: string): string {
  return escapeHTML(value);
}

function entity(char: string): string {
  const entities: Record<string, string> = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  };
  return entities[char] ?? char;
}
