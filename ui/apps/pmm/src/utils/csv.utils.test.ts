import { describe, expect, it } from 'vitest';
import { sanitizeCsvCell, toCsvValue } from './csv.utils';

describe('csv utils', () => {
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
});
