import { formatDurationSeconds } from './duration.utils';

describe('formatDurationSeconds', () => {
  it('returns an empty string for missing values', () => {
    expect(formatDurationSeconds(undefined)).toBe('');
    expect(formatDurationSeconds(NaN)).toBe('');
  });

  it('formats zero as 0s', () => {
    expect(formatDurationSeconds(0)).toBe('0s');
  });

  it('formats sub-second values as milliseconds', () => {
    expect(formatDurationSeconds(0.015)).toBe('15ms');
  });

  it('formats seconds, minutes and hours compactly', () => {
    expect(formatDurationSeconds(30)).toBe('30s');
    expect(formatDurationSeconds(300)).toBe('5m');
    expect(formatDurationSeconds(3600)).toBe('1h');
    expect(formatDurationSeconds(5400)).toBe('1h 30m');
    expect(formatDurationSeconds(3661)).toBe('1h 1m 1s');
  });
});
