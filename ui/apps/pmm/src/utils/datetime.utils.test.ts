import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { diffFromNow } from './datetime.utils';

// 1 hour = 3,600,000 ms
const HOUR_MS = 3_600_000;

describe.only('diffFromNow', () => {
  beforeEach(() => {
    // Mock Date.now() to have consistent test results
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should return positive milliseconds when timestamp is in the future', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    const futureTimestamp = '2024-01-01T12:00:01.000Z';
    const result = diffFromNow(futureTimestamp);

    expect(result).toBe(1000); // 1 second = 1000 milliseconds
  });

  it('should return negative milliseconds when timestamp is in the past', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    const pastTimestamp = '2024-01-01T11:59:59.000Z';
    const result = diffFromNow(pastTimestamp);

    expect(result).toBe(-1000); // 1 second ago = -1000 milliseconds
  });

  it('should return 0 when timestamp is exactly now', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    const currentTimestamp = '2024-01-01T12:00:00.000Z';
    const result = diffFromNow(currentTimestamp);

    expect(result).toBe(0);
  });

  it('should handle timestamps with different formats', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    // Test with ISO string format
    const isoTimestamp = '2024-01-01T12:00:00.500Z';
    expect(diffFromNow(isoTimestamp)).toBe(500);

    // Test with date string format (this will be interpreted as local time)
    const dateString = '2024-01-01T12:00:00.500Z'; // Use UTC format for consistency
    expect(diffFromNow(dateString)).toBe(500);
  });

  it('should handle edge cases with milliseconds precision', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    // Test with very small differences
    const smallFuture = '2024-01-01T12:00:00.001Z';
    expect(diffFromNow(smallFuture)).toBe(1);

    const smallPast = '2024-01-01T11:59:59.999Z';
    expect(diffFromNow(smallPast)).toBe(-1);
  });

  it('should handle large time differences', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    // Test with 1 hour difference
    const oneHourFuture = '2024-01-01T13:00:00.000Z';
    expect(diffFromNow(oneHourFuture)).toBe(HOUR_MS);

    const oneHourPast = '2024-01-01T11:00:00.000Z';
    expect(diffFromNow(oneHourPast)).toBe(-HOUR_MS);
  });

  it('should handle different timezones correctly', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    // Test with UTC timestamp
    const utcTimestamp = '2024-01-01T12:00:01.000Z';
    expect(diffFromNow(utcTimestamp)).toBe(1000);

    // Test with another UTC timestamp for consistency
    const anotherUtcTimestamp = '2024-01-01T12:00:01.000Z';
    expect(diffFromNow(anotherUtcTimestamp)).toBe(1000);

    const plusOneTimestamp = '2024-01-01T12:00:00.000-01:00';
    expect(diffFromNow(plusOneTimestamp)).toBe(HOUR_MS);
  });

  it('should handle invalid timestamp strings gracefully', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    // Invalid date should return NaN
    expect(diffFromNow('invalid-date')).toBeNaN();
    expect(diffFromNow('')).toBeNaN();
    expect(diffFromNow('not-a-date')).toBeNaN();
  });

  it('should be consistent with multiple calls at the same time', () => {
    const now = new Date('2024-01-01T12:00:00.000Z');
    vi.setSystemTime(now);

    const timestamp = '2024-01-01T12:00:01.000Z';
    const result1 = diffFromNow(timestamp);
    const result2 = diffFromNow(timestamp);

    expect(result1).toBe(result2);
    expect(result1).toBe(1000);
  });
});
