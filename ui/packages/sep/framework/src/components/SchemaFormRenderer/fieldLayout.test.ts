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
import { collectParentNames, isFullRowField } from './fieldLayout';
import type { PluginField } from './types';

describe('collectParentNames', () => {
  it('gathers every name pointed at by a `parent`', () => {
    const fields: PluginField[] = [
      { type: 'bool', name: 'encrypt', label: 'Encrypt' },
      { type: 'bool', name: 'prepare', label: 'Prepare' },
      { type: 'string', name: 'tmpdir', label: 'Tmpdir', parent: 'encrypt' },
      { type: 'string', name: 'memory', label: 'Memory', parent: 'prepare' },
      { type: 'string', name: 'loose', label: 'Loose' },
    ];
    expect([...collectParentNames(fields)].sort()).toEqual([
      'encrypt',
      'prepare',
    ]);
  });

  it('is empty for a schema with no parented fields', () => {
    expect(
      collectParentNames([{ type: 'string', name: 'a', label: 'A' }]).size
    ).toBe(0);
  });
});

describe('isFullRowField', () => {
  const scalar: PluginField = { type: 'string', name: 'a', label: 'A' };

  it('puts an ordinary scalar field in one column', () => {
    expect(isFullRowField(scalar)).toBe(false);
  });

  it.each(['textarea', 'yaml', 'script_preview', 'multi_choice', 'file'])(
    'gives %s the whole row',
    (type) => {
      expect(
        isFullRowField({ ...scalar, type } as unknown as PluginField)
      ).toBe(true);
    }
  );

  it('gives a parented child the whole row so it starts a fresh line', () => {
    expect(isFullRowField({ ...scalar, parent: 'encrypt' })).toBe(true);
  });

  it('gives a toggle that has children the whole row', () => {
    const parents = new Set(['encrypt']);
    expect(
      isFullRowField({ type: 'bool', name: 'encrypt', label: 'E' }, parents)
    ).toBe(true);
    expect(
      isFullRowField({ type: 'bool', name: 'other', label: 'O' }, parents)
    ).toBe(false);
  });
});
