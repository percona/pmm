import { describe, it, expect } from 'vitest';
import {
  convertSecondsToDays,
  convertSecondsStringToHour,
  convertHoursStringToSeconds,
  convertCheckIntervalsToHours,
} from './Advanced.utils';

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

describe('convertSecondsStringToHour', () => {
  it('converts a seconds string to hours', () => {
    expect(convertSecondsStringToHour('3600s')).toBe(1);
    expect(convertSecondsStringToHour('7200s')).toBe(2);
    expect(convertSecondsStringToHour('1800s')).toBe(0.5);
  });

  it('works without the s suffix', () => {
    expect(convertSecondsStringToHour('3600')).toBe(1);
  });

  it('returns 0 for zero', () => {
    expect(convertSecondsStringToHour('0s')).toBe(0);
  });
});

describe('convertHoursStringToSeconds', () => {
  it('converts hours to seconds', () => {
    expect(convertHoursStringToSeconds('1')).toBe(3600);
    expect(convertHoursStringToSeconds(2)).toBe(7200);
  });

  it('rounds fractional hours correctly', () => {
    expect(convertHoursStringToSeconds('0.5')).toBe(1800);
    expect(convertHoursStringToSeconds('0.1')).toBe(360);
  });
});

describe('convertCheckIntervalsToHours', () => {
  it('converts interval seconds strings to hour strings', () => {
    const result = convertCheckIntervalsToHours({
      rareInterval: '86400s',
      standardInterval: '3600s',
      frequentInterval: '1800s',
    });
    expect(result).toEqual({
      rareInterval: '24',
      standardInterval: '1',
      frequentInterval: '0.5',
    });
  });

  it('returns default 24-hour values when intervals are undefined', () => {
    const result = convertCheckIntervalsToHours(undefined);
    expect(result).toEqual({
      rareInterval: '24',
      standardInterval: '24',
      frequentInterval: '24',
    });
  });
});
