import { describe, it, expect } from 'vitest';
import { convertSecondsToDays } from './Advanced.utils';

describe('convertSecondsToDays', () => {
  it('converts seconds to days', () => {
    expect(convertSecondsToDays('86400s')).toBe(1);
    expect(convertSecondsToDays('172800s')).toBe(2);
  });

  it('converts minutes to days', () => {
    expect(convertSecondsToDays('1440m')).toBe(1);
    expect(convertSecondsToDays('2880m')).toBe(2);
  });

  it('converts hours to days', () => {
    expect(convertSecondsToDays('24h')).toBe(1);
    expect(convertSecondsToDays('48h')).toBe(2);
  });

  it('returns value unchanged for days unit', () => {
    expect(convertSecondsToDays('7d')).toBe(7);
  });

  it('returns empty string for unknown units', () => {
    expect(convertSecondsToDays('100x')).toBe('');
  });

  it('returns empty string for empty input', () => {
    expect(convertSecondsToDays('')).toBe('');
  });
});
