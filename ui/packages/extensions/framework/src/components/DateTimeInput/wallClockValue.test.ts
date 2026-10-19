/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { describe, expect, it } from 'vitest';
import { dateToWallClock, wallClockToDate } from './wallClockValue';

describe('wallClockToDate', () => {
  it('reads the digits as local time so the picker shows them unchanged', () => {
    const date = wallClockToDate('2026-03-01T14:30');

    expect(date?.getFullYear()).toBe(2026);
    expect(date?.getMonth()).toBe(2);
    expect(date?.getDate()).toBe(1);
    expect(date?.getHours()).toBe(14);
    expect(date?.getMinutes()).toBe(30);
  });

  it('accepts the seconds an existing value may carry', () => {
    expect(wallClockToDate('2026-03-01T14:30:45')?.getSeconds()).toBe(45);
  });

  it.each([
    ['empty', ''],
    ['date only', '2026-03-01'],
    ['an ISO instant', '2026-03-01T14:30:00Z'],
    ['nonsense', 'tomorrow'],
  ])('returns null for %s', (_label, value) => {
    expect(wallClockToDate(value)).toBeNull();
  });

  it('returns null for a non-string', () => {
    expect(wallClockToDate(new Date())).toBeNull();
    expect(wallClockToDate(undefined)).toBeNull();
  });

  // The pattern checks digit widths, not ranges, so these all reach `Date`,
  // which rolls them into some other date rather than refusing. Rendering a
  // value nobody wrote is worse than rendering nothing.
  it.each([
    ['a day the month does not have', '2026-02-30T00:00'],
    ['a thirteenth month', '2026-13-01T00:00'],
    ['a twenty-fifth hour', '2026-03-01T25:00'],
  ])('returns null for %s', (_label, value) => {
    expect(wallClockToDate(value)).toBeNull();
  });

  // `Date` maps a year below 100 into the 1900s. Restated, so a year means
  // what it says.
  it('keeps a year below 100 out of the 1900s', () => {
    expect(wallClockToDate('0050-01-02T03:04')?.getFullYear()).toBe(50);
    expect(dateToWallClock(wallClockToDate('0050-01-02T03:04'))).toBe(
      '0050-01-02T03:04'
    );
  });

  // Readable but not writable: nothing in these forms lets anyone choose a
  // second, so a stored value carrying one renders and then saves at minute
  // precision, exactly as the `datetime-local` field it replaced did.
  it('reads seconds but does not write them back', () => {
    expect(dateToWallClock(wallClockToDate('2026-03-01T14:30:45'))).toBe(
      '2026-03-01T14:30'
    );
  });
});

describe('dateToWallClock', () => {
  it('writes local parts at minute precision', () => {
    expect(dateToWallClock(new Date(2027, 3, 5, 9, 15, 45))).toBe(
      '2027-04-05T09:15'
    );
  });

  it('pads every part to two digits', () => {
    expect(dateToWallClock(new Date(2026, 0, 2, 3, 4))).toBe(
      '2026-01-02T03:04'
    );
  });

  it('returns an empty string for a cleared or half-typed value', () => {
    expect(dateToWallClock(null)).toBe('');
    expect(dateToWallClock(new Date(NaN))).toBe('');
  });

  it('round-trips a wall clock through the picker unchanged', () => {
    const value = '2026-12-31T23:59';

    expect(dateToWallClock(wallClockToDate(value))).toBe(value);
  });
});
