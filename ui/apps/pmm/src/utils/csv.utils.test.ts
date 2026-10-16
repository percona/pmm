import { describe, expect, it, vi } from 'vitest';
import {
  buildCsvFilename,
  collectCsvColumns,
  downloadCsv,
  sanitizeCsvCell,
  toCsvValue,
} from './csv.utils';

const { download, generateCsv, mkConfig } = vi.hoisted(() => ({
  download: vi.fn(() => vi.fn()),
  generateCsv: vi.fn(() => vi.fn(() => 'csv-content')),
  mkConfig: vi.fn((config) => config),
}));

vi.mock('export-to-csv', () => ({
  download,
  generateCsv,
  mkConfig,
}));

describe('csv utils', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('serializes values the csv generator cannot handle on its own', () => {
    expect(toCsvValue(10)).toBe(10);
    expect(toCsvValue(null)).toBe('');
    expect(toCsvValue(undefined)).toBe('');
    expect(toCsvValue(true)).toBe(true);
    expect(toCsvValue(['a', 'b'])).toBe('["a","b"]');
  });

  it('sanitizes values that could be interpreted as spreadsheet formulas', () => {
    expect(sanitizeCsvCell('=1+1')).toBe("'=1+1");
    expect(sanitizeCsvCell('+cmd')).toBe("'+cmd");
    expect(sanitizeCsvCell('-10')).toBe("'-10");
    expect(sanitizeCsvCell('@SUM(A1)')).toBe("'@SUM(A1)");
    expect(sanitizeCsvCell('{ find: "x" }')).toBe('{ find: "x" }');
  });

  it('collects columns across all rows, listed ones first', () => {
    expect(
      collectCsvColumns(
        [
          { c: 1, a: 1 },
          { d: 1, b: 1 },
        ],
        ['b', 'a']
      )
    ).toEqual(['b', 'a', 'c', 'd']);
  });

  it('builds the filename from a prefix and a timestamp', () => {
    expect(
      buildCsvFilename('export', new Date('2026-06-25T14:30:22.000Z'))
    ).toMatch(/^export_\d{8}_\d{6}$/);
  });

  it('downloads the rows as csv', () => {
    downloadCsv([{ b: 1, a: 2 }], 'export', ['a']);

    expect(mkConfig).toHaveBeenCalledWith({
      columnHeaders: ['a', 'b'],
      filename: expect.stringMatching(/^export_\d{8}_\d{6}$/),
    });
    expect(generateCsv).toHaveBeenCalled();
    expect(download).toHaveBeenCalled();
  });

  it('does not download when there are no rows', () => {
    downloadCsv([], 'export');

    expect(mkConfig).not.toHaveBeenCalled();
    expect(download).not.toHaveBeenCalled();
  });
});
