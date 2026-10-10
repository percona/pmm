import { describe, it, expect } from 'vitest';
import {
  convertSecondsStringToHour,
  convertHoursStringToSeconds,
  convertCheckIntervalsToHours,
} from './Advisors.utils';

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
