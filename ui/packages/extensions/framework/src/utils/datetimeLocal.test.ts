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
import {
  fromDatetimeLocalValue,
  localInputToUtcIso,
  toDatetimeLocalValue,
  utcIsoToLocalInput,
} from './datetimeLocal';

describe('utcIsoToLocalInput / localInputToUtcIso', () => {
  it('round-trips a whole-minute UTC instant through the browser zone', () => {
    const iso = '2024-06-15T12:00:00.000Z';
    const local = utcIsoToLocalInput(iso);
    expect(local).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
    expect(localInputToUtcIso(local)).toBe(iso);
  });

  it('returns empty for invalid ISO', () => {
    expect(utcIsoToLocalInput('not-a-date')).toBe('');
  });

  it('returns undefined for empty or invalid local input', () => {
    expect(localInputToUtcIso('')).toBeUndefined();
    expect(localInputToUtcIso('   ')).toBeUndefined();
    expect(localInputToUtcIso('not-a-date')).toBeUndefined();
  });
});

describe('toDatetimeLocalValue / fromDatetimeLocalValue', () => {
  it('converts ISO defaults to datetime-local', () => {
    const iso = '2024-06-15T12:00:00.000Z';
    expect(toDatetimeLocalValue(iso)).toBe(utcIsoToLocalInput(iso));
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

  it('round-trips through fromDatetimeLocalValue', () => {
    const iso = '2024-01-01T00:00:00.000Z';
    expect(fromDatetimeLocalValue(toDatetimeLocalValue(iso))).toBe(iso);
  });
});
