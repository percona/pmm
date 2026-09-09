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
import type { ChoiceOption } from '@sep/api';
import { normalizeChange, toDisplayValue } from './choiceFreeSoloValue';

const OPTIONS: ChoiceOption[] = [
  { value: 'backup-1', label: 'Backup 1' },
  {
    value: 'backup-2',
    label: 'Backup 2',
    disabled: true,
    disabled_reason: 'In progress',
  },
];

describe('toDisplayValue', () => {
  it('resolves a stored value string to its option', () => {
    expect(toDisplayValue('backup-1', OPTIONS)).toEqual({
      value: 'backup-1',
      label: 'Backup 1',
    });
  });

  it('shows a stored custom string verbatim when no option matches', () => {
    expect(toDisplayValue('typed-custom', OPTIONS)).toBe('typed-custom');
  });

  it('resolves a stored option object (back-compat) to the matching option', () => {
    expect(
      toDisplayValue({ value: 'backup-2', label: 'Backup 2' }, OPTIONS)
    ).toEqual(OPTIONS[1]);
  });

  it('treats empty / null / undefined as no value', () => {
    expect(toDisplayValue('', OPTIONS)).toBeNull();
    expect(toDisplayValue(null, OPTIONS)).toBeNull();
    expect(toDisplayValue(undefined, OPTIONS)).toBeNull();
  });

  it('coerces a stored number to its string-value option', () => {
    const numeric: ChoiceOption[] = [{ value: '7', label: 'Seven' }];
    expect(toDisplayValue(7, numeric)).toEqual({ value: '7', label: 'Seven' });
  });
});

describe('normalizeChange', () => {
  it('commits the value string when an option is picked', () => {
    expect(normalizeChange(OPTIONS[0], OPTIONS)).toBe('backup-1');
  });

  it('commits a typed custom value verbatim', () => {
    expect(normalizeChange('brand-new', OPTIONS)).toBe('brand-new');
  });

  it('resolves a typed value matching an option label to that option value', () => {
    expect(normalizeChange('Backup 1', OPTIONS)).toBe('backup-1');
  });

  it('keeps a typed value matching a disabled option label as a custom string', () => {
    // Backup 2 is disabled; typing its label must not commit the disabled value.
    expect(normalizeChange('Backup 2', OPTIONS)).toBe('Backup 2');
  });

  it('commits null when cleared or blank', () => {
    expect(normalizeChange(null, OPTIONS)).toBeNull();
    expect(normalizeChange('   ', OPTIONS)).toBeNull();
    expect(normalizeChange('', OPTIONS)).toBeNull();
  });

  it('never commits an id — the committed value is always a string or null', () => {
    const result = normalizeChange(OPTIONS[1], OPTIONS);
    expect(typeof result).toBe('string');
    expect(result).toBe('backup-2');
  });
});
