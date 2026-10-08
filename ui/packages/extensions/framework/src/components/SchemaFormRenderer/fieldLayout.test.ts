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
import { fieldIndex } from './fieldLayout';
import type { PluginField } from './types';

describe('fieldIndex', () => {
  it('gathers every name pointed at by a `parent`', () => {
    const fields: PluginField[] = [
      { type: 'bool', name: 'encrypt', label: 'Encrypt' },
      { type: 'bool', name: 'prepare', label: 'Prepare' },
      { type: 'string', name: 'tmpdir', label: 'Tmpdir', parent: 'encrypt' },
      { type: 'string', name: 'memory', label: 'Memory', parent: 'prepare' },
      { type: 'string', name: 'loose', label: 'Loose' },
    ];
    expect([...fieldIndex(fields).parentNames].sort()).toEqual([
      'encrypt',
      'prepare',
    ]);
  });

  it('indexes every field by name so a parent resolves to its label', () => {
    const fields: PluginField[] = [
      { type: 'bool', name: 'encrypt', label: 'Encrypt backup' },
      { type: 'string', name: 'tmpdir', label: 'Tmpdir', parent: 'encrypt' },
    ];
    expect(fieldIndex(fields).byName.get('encrypt')?.label).toBe(
      'Encrypt backup'
    );
  });

  it('derives the index once per fields array', () => {
    // Every slot in a form is handed the same array from context; recomputing
    // per slot is the O(N^2) this cache exists to avoid.
    const fields: PluginField[] = [
      { type: 'bool', name: 'encrypt', label: 'Encrypt' },
    ];
    expect(fieldIndex(fields)).toBe(fieldIndex(fields));
    expect(fieldIndex([...fields])).not.toBe(fieldIndex(fields));
  });

  it('is empty for a schema with no parented fields', () => {
    expect(
      fieldIndex([{ type: 'string', name: 'a', label: 'A' }]).parentNames.size
    ).toBe(0);
  });
});
