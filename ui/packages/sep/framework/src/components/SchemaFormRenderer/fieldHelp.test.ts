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
  descriptionEchoesLabel,
  fieldHelp,
  INLINE_HELP_MAX_LENGTH,
} from './fieldHelp';
import type { PluginField } from './types';

function stringField(
  overrides: Partial<Extract<PluginField, { type: 'string' }>> = {}
): Extract<PluginField, { type: 'string' }> {
  return {
    type: 'string',
    name: 'samples',
    label: 'Save samples',
    ...overrides,
  };
}

describe('descriptionEchoesLabel', () => {
  it('matches when description restates the label, ignoring case and trim', () => {
    expect(descriptionEchoesLabel('Save samples', 'Save samples')).toBe(true);
    expect(descriptionEchoesLabel('Save samples', '  save samples  ')).toBe(
      true
    );
  });

  it('does not match when the description adds real detail', () => {
    expect(
      descriptionEchoesLabel('Save samples', 'Write samples to disk')
    ).toBe(false);
  });
});

describe('fieldHelp', () => {
  it('returns nothing when description is missing', () => {
    expect(fieldHelp(stringField())).toEqual({});
  });

  it('returns nothing when description only echoes the label', () => {
    expect(
      fieldHelp(stringField({ description: 'Save samples' }))
    ).toEqual({});
    expect(
      fieldHelp(
        stringField({
          description: 'save samples',
          help_placement: 'tooltip',
        })
      )
    ).toEqual({});
  });

  it('keeps a short distinct description inline', () => {
    expect(
      fieldHelp(stringField({ description: 'Write samples to disk' }))
    ).toEqual({ inline: 'Write samples to disk' });
  });

  it('puts long distinct prose behind the help icon', () => {
    const longProse = `prose ${'x'.repeat(INLINE_HELP_MAX_LENGTH)}`;
    expect(fieldHelp(stringField({ description: longProse }))).toEqual({
      tooltip: longProse,
    });
  });
});
