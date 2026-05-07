import { describe, expect, it } from 'vitest';
import { disabled, errorText, escapeAttr, escapeHTML, formatBytes } from './utils';

describe('formatBytes', () => {
  it('returns 0 B for falsy input', () => {
    expect(formatBytes(0)).toBe('0 B');
  });

  it('formats kilobytes', () => {
    expect(formatBytes(512)).toBe('1 KB');
    expect(formatBytes(1024)).toBe('1 KB');
    expect(formatBytes(1536)).toBe('2 KB');
  });

  it('formats megabytes', () => {
    expect(formatBytes(1_048_576)).toBe('1.0 MB');
    expect(formatBytes(5_242_880)).toBe('5.0 MB');
    expect(formatBytes(1_572_864)).toBe('1.5 MB');
  });
});

describe('escapeHTML', () => {
  it('escapes ampersands and angle brackets', () => {
    expect(escapeHTML('<script>alert("xss")</script>')).toBe(
      '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;',
    );
  });

  it('escapes single quotes', () => {
    expect(escapeHTML("it's")).toBe('it&#39;s');
  });

  it('returns plain text unchanged', () => {
    expect(escapeHTML('hello world.pdf')).toBe('hello world.pdf');
  });

  it('handles empty string', () => {
    expect(escapeHTML('')).toBe('');
  });
});

describe('escapeAttr', () => {
  it('escapes HTML in attribute values', () => {
    expect(escapeAttr('a"b')).toBe('a&quot;b');
  });
});

describe('errorText', () => {
  it('extracts message from Error objects', () => {
    expect(errorText(new Error('something failed'))).toBe('something failed');
  });

  it('stringifies non-Error values', () => {
    expect(errorText(42)).toBe('42');
    expect(errorText('oops')).toBe('oops');
  });
});

describe('disabled', () => {
  it('returns disabled when busy', () => {
    expect(disabled('Importing')).toBe('disabled');
  });

  it('returns empty string when idle', () => {
    expect(disabled('')).toBe('');
  });
});
