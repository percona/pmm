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
  capitalizeItemLabel,
  resolveItemDisplayName,
  resolveItemDisplayNamePlural,
} from './itemLabels';

describe('capitalizeItemLabel', () => {
  it('capitalises the first character of a mid-sentence noun', () => {
    expect(capitalizeItemLabel('backup')).toBe('Backup');
    expect(capitalizeItemLabel('restore')).toBe('Restore');
  });

  it('leaves the rest of the string unchanged', () => {
    expect(capitalizeItemLabel('mySQL backup')).toBe('MySQL backup');
    expect(capitalizeItemLabel('Backup')).toBe('Backup');
  });

  it('handles a single character and the empty string', () => {
    expect(capitalizeItemLabel('b')).toBe('B');
    expect(capitalizeItemLabel('')).toBe('');
  });

  it('does not throw on nullish or non-string input', () => {
    expect(capitalizeItemLabel(undefined)).toBe('');
    expect(capitalizeItemLabel(null)).toBe('');
  });
});

describe('resolveItemDisplayName / resolveItemDisplayNamePlural', () => {
  it('reads the schema item fields when present', () => {
    const schema = {
      display_name: 'MySQL Backups',
      item_display_name: 'backup',
      item_display_name_plural: 'backups',
    };
    expect(resolveItemDisplayName(schema)).toBe('backup');
    expect(resolveItemDisplayNamePlural(schema)).toBe('backups');
  });

  it('falls back to display_name when item fields are missing or empty', () => {
    expect(
      resolveItemDisplayName({ display_name: 'Checksum' })
    ).toBe('Checksum');
    expect(
      resolveItemDisplayName({
        display_name: 'Checksum',
        item_display_name: '',
      })
    ).toBe('Checksum');
    expect(
      resolveItemDisplayNamePlural({
        display_name: 'Checksum',
        item_display_name_plural: null,
      })
    ).toBe('Checksum');
  });

  it('never returns blank when both item and display names are missing', () => {
    expect(resolveItemDisplayName({})).toBe('item');
    expect(resolveItemDisplayName({ display_name: '' })).toBe('item');
    expect(resolveItemDisplayNamePlural({})).toBe('items');
    expect(resolveItemDisplayNamePlural({ display_name: null })).toBe('items');
  });
});
