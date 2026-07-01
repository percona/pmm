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
  normalizeChange,
  toDisplayValue,
  type ReferenceOption,
} from './freeSoloValue';

const OPTIONS: ReferenceOption[] = [
  { id: 10, name: 'app_prod' },
  { id: 11, name: 'analytics' },
];

const labelOf = (o: ReferenceOption) => o.name;

describe('toDisplayValue', () => {
  it('resolves a stored numeric id to its option', () => {
    expect(toDisplayValue(10, OPTIONS)).toEqual({ id: 10, name: 'app_prod' });
  });

  it('shows nothing for an id whose option has not loaded yet', () => {
    expect(toDisplayValue(99, OPTIONS)).toBeNull();
    expect(toDisplayValue(10, [])).toBeNull();
  });

  it('shows a stored free string verbatim', () => {
    expect(toDisplayValue('custom_db', OPTIONS)).toBe('custom_db');
  });

  it('resolves a stored option object (back-compat) to the matching option', () => {
    expect(toDisplayValue({ id: 11, name: 'analytics' }, OPTIONS)).toEqual({
      id: 11,
      name: 'analytics',
    });
  });

  it('treats empty / null / undefined as no value', () => {
    expect(toDisplayValue('', OPTIONS)).toBeNull();
    expect(toDisplayValue(null, OPTIONS)).toBeNull();
    expect(toDisplayValue(undefined, OPTIONS)).toBeNull();
  });

  it('keeps a numeric-looking custom string as a string (not resolved to an id)', () => {
    expect(toDisplayValue('123', OPTIONS)).toBe('123');
  });
});

describe('normalizeChange', () => {
  it('commits the id when an option object is picked', () => {
    expect(
      normalizeChange({ id: 10, name: 'app_prod' }, OPTIONS, labelOf)
    ).toBe(10);
  });

  it('commits a string for a typed value with no matching option', () => {
    expect(normalizeChange('brand_new', OPTIONS, labelOf)).toBe('brand_new');
  });

  it('resolves a typed value that matches an option label to the id', () => {
    expect(normalizeChange('analytics', OPTIONS, labelOf)).toBe(11);
  });

  it('commits null when cleared or blank', () => {
    expect(normalizeChange(null, OPTIONS, labelOf)).toBeNull();
    expect(normalizeChange('   ', OPTIONS, labelOf)).toBeNull();
    expect(normalizeChange('', OPTIONS, labelOf)).toBeNull();
  });
});
