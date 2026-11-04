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
import { toDatetimeLocalValue, utcIsoToUtcInput } from './datetimeLocal';

describe('utcIsoToUtcInput', () => {
  it('formats a zoned instant as its UTC wall-clock', () => {
    expect(utcIsoToUtcInput('2024-06-15T12:00:00.000Z')).toBe(
      '2024-06-15T12:00'
    );
    expect(utcIsoToUtcInput('2024-06-15T09:00:00-03:00')).toBe(
      '2024-06-15T12:00'
    );
  });

  it('returns empty for invalid ISO', () => {
    expect(utcIsoToUtcInput('not-a-date')).toBe('');
  });
});

describe('toDatetimeLocalValue', () => {
  it('converts zoned ISO defaults to their UTC wall-clock', () => {
    expect(toDatetimeLocalValue('2024-06-15T12:00:00.000Z')).toBe(
      '2024-06-15T12:00'
    );
  });

  it('keeps bare local wall-clock strings at minute precision', () => {
    expect(toDatetimeLocalValue('2024-06-15T10:30:45')).toBe(
      '2024-06-15T10:30'
    );
    expect(toDatetimeLocalValue('2024-06-15T10:30')).toBe('2024-06-15T10:30');
  });

  it('treats blank and non-string as empty', () => {
    expect(toDatetimeLocalValue('')).toBe('');
    expect(toDatetimeLocalValue(null)).toBe('');
    expect(toDatetimeLocalValue(42)).toBe('');
  });
});
