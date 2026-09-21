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
import type { PluginField } from '@sep/api';
import {
  EMPTY_SECTION_SUMMARY,
  summariseFieldValue,
  summariseSectionValues,
} from './sectionValueSummary';

const text: PluginField = { type: 'string', name: 'host', label: 'Host' };

describe('summariseFieldValue', () => {
  it('names the field and its value', () => {
    expect(summariseFieldValue(text, 'db-1')).toBe('Host: db-1');
  });

  it.each([undefined, null, ''])('skips the empty value %p', (value) => {
    expect(summariseFieldValue(text, value)).toBeNull();
  });

  it('resolves a choice value to its option label', () => {
    const field: PluginField = {
      type: 'choice',
      name: 'mode',
      label: 'Mode',
      choices: [
        { label: 'Fast', value: 'fast' },
        { label: 'Thorough', value: 'thorough' },
      ],
    };
    expect(summariseFieldValue(field, 'thorough')).toBe('Mode: Thorough');
  });

  it('falls back to the raw value when no option matches', () => {
    const field: PluginField = {
      type: 'choice',
      name: 'mode',
      label: 'Mode',
      choices: [{ label: 'Fast', value: 'fast' }],
    };
    expect(summariseFieldValue(field, 'custom')).toBe('Mode: custom');
  });

  it('joins a multi-value field', () => {
    const field: PluginField = {
      type: 'multi_choice',
      name: 'tags',
      label: 'Tags',
      choices: [
        { label: 'One', value: '1' },
        { label: 'Two', value: '2' },
      ],
    };
    expect(summariseFieldValue(field, ['1', '2'])).toBe('Tags: One, Two');
    expect(summariseFieldValue(field, [])).toBeNull();
  });

  it('names a switch that is on', () => {
    const field: PluginField = { type: 'bool', name: 'sudo', label: 'Sudo' };
    expect(summariseFieldValue(field, true)).toBe('Sudo: on');
  });

  it('drops a switch that is off, unless the schema pre-filled it on', () => {
    const off: PluginField = { type: 'bool', name: 'sudo', label: 'Sudo' };
    expect(summariseFieldValue(off, false)).toBeNull();

    const defaultOn: PluginField = {
      type: 'bool',
      name: 'sudo',
      label: 'Sudo',
      default: true,
    };
    expect(summariseFieldValue(defaultOn, false)).toBe('Sudo: off');
  });

  it.each(['file', 'script_preview'] as const)(
    'skips the unsummarisable %s field',
    (type) => {
      const field = {
        type,
        name: 'payload',
        label: 'Payload',
        endpoint_url: '/preview',
        depends_on: [],
      } as unknown as PluginField;
      expect(summariseFieldValue(field, 'anything')).toBeNull();
    }
  );
});

describe('summariseSectionValues', () => {
  const fields: PluginField[] = [
    { type: 'string', name: 'a', label: 'A' },
    { type: 'string', name: 'b', label: 'B' },
  ];

  it('reads values positionally against the fields', () => {
    expect(summariseSectionValues(fields, ['1', '2'])).toBe('A: 1 · B: 2');
  });

  it('skips the fields holding nothing', () => {
    expect(summariseSectionValues(fields, ['', '2'])).toBe('B: 2');
  });

  it('says so when the section holds nothing at all', () => {
    expect(summariseSectionValues(fields, [undefined, undefined])).toBe(
      EMPTY_SECTION_SUMMARY
    );
    expect(summariseSectionValues([], [])).toBe(EMPTY_SECTION_SUMMARY);
  });

  it('counts the entries past the fourth instead of running off the line', () => {
    const many: PluginField[] = ['a', 'b', 'c', 'd', 'e', 'f'].map((name) => ({
      type: 'string',
      name,
      label: name.toUpperCase(),
    }));
    expect(summariseSectionValues(many, ['1', '2', '3', '4', '5', '6'])).toBe(
      'A: 1 · B: 2 · C: 3 · D: 4 · +2 more'
    );
  });
});
