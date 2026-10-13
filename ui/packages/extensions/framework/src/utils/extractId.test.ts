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
import { extractId } from './extractId';

describe('extractId', () => {
  it('passes a finite number through', () => {
    expect(extractId(7)).toBe(7);
    expect(extractId(0)).toBe(0);
  });

  it('parses a decimal-integer string', () => {
    expect(extractId('42')).toBe(42);
    expect(extractId(' 42 ')).toBe(42);
    expect(extractId('-3')).toBe(-3);
  });

  it('extracts the id from an option object recursively', () => {
    expect(extractId({ id: '9', name: 'svc' })).toBe(9);
    expect(extractId({ id: { id: 9 } })).toBe(9);
  });

  it('rejects strings that only look numeric', () => {
    // `Number` would turn these into 0 / 1.5 / 16, each of which then reads as a
    // resolvable inventory id and fires a lookup for a service that cannot exist.
    expect(extractId('   ')).toBeNull();
    expect(extractId('')).toBeNull();
    expect(extractId('1.5')).toBeNull();
    expect(extractId('0x10')).toBeNull();
    expect(extractId('12abc')).toBeNull();
  });

  it('rejects a fractional or unsafe-integer number, like the string branch', () => {
    expect(extractId(1.5)).toBeNull();
    expect(extractId(Number.MAX_SAFE_INTEGER + 1)).toBeNull();
    expect(extractId({ id: 1.5 })).toBeNull();
  });

  it('rejects non-finite numbers and unrelated values', () => {
    expect(extractId(Number.NaN)).toBeNull();
    expect(extractId(Number.POSITIVE_INFINITY)).toBeNull();
    expect(extractId(null)).toBeNull();
    expect(extractId(undefined)).toBeNull();
    expect(extractId({ name: 'no id' })).toBeNull();
  });
});
