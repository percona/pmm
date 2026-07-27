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
import type { SectionField } from '@sep/api';
import { buildBatchPayload, fieldDeclaresGate, namespaceField } from '../src/CollectPane';
import type { AtwSnippetSummary } from '../src/types';

const snippets: AtwSnippetSummary[] = [
  { name: 'a.sh', title: 'A', description: '' },
  { name: 'b.sh', title: 'B', description: '' },
];

describe('buildBatchPayload', () => {
  it('hoists executor_host and sudo and keeps other top-level values as shared args', () => {
    const payload = buildBatchPayload(
      {
        executor_host: 'db-1',
        sudo: true,
        'defaults-file': '/etc/my.cnf',
        overrides: {},
      },
      [],
    );

    expect(payload.executor_host).toBe('db-1');
    expect(payload.sudo).toBe(true);
    expect(payload.shared_args).toEqual({ 'defaults-file': '/etc/my.cnf' });
    expect(payload.items).toEqual([]);
  });

  it('maps each snippet to its namespaced override args in selection order', () => {
    const payload = buildBatchPayload(
      {
        executor_host: 'db-1',
        overrides: {
          snip0: { minutes: 5 },
          snip1: { minutes: 10 },
        },
      },
      snippets,
    );

    expect(payload.items).toEqual([
      { snippet_filename: 'a.sh', args: { minutes: 5 } },
      { snippet_filename: 'b.sh', args: { minutes: 10 } },
    ]);
  });

  it('drops empty-string and reserved fields from both shared and per-snippet args', () => {
    const payload = buildBatchPayload(
      {
        executor_host: 'db-1',
        sudo: false,
        script_preview: 'ignored',
        empty: '',
        overrides: {
          snip0: { minutes: 5, note: '', executor_host: 'ignored' },
        },
      },
      [snippets[0]],
    );

    expect(payload.shared_args).toEqual({});
    expect(payload.items[0].args).toEqual({ minutes: 5 });
  });

  it('defaults executor_host to an empty string when absent', () => {
    const payload = buildBatchPayload({ overrides: {} }, []);
    expect(payload.executor_host).toBe('');
    expect(payload.sudo).toBe(false);
  });
});

describe('namespaceField', () => {
  it('prefixes a leaf field name', () => {
    const field: SectionField = { type: 'string', name: 'minutes', label: 'Minutes' };
    expect(namespaceField(field, 'overrides.snip0.')).toMatchObject({
      name: 'overrides.snip0.minutes',
      label: 'Minutes',
    });
  });

  it('prefixes a one-of group discriminator and branch leaf names', () => {
    const group: SectionField = {
      type: 'one_of',
      name: 'source',
      label: 'Source',
      discriminator: 'source.mode',
      branches: [
        {
          value: 'file',
          label: 'File',
          fields: [{ type: 'string', name: 'source.path', label: 'Path' }],
        },
      ],
    };

    const result = namespaceField(group, 'overrides.snip1.');
    expect(result).toMatchObject({
      name: 'overrides.snip1.source',
      discriminator: 'overrides.snip1.source.mode',
    });
    if (result.type === 'one_of') {
      expect(result.branches[0].fields[0].name).toBe('overrides.snip1.source.path');
    }
  });
});

describe('fieldDeclaresGate', () => {
  it('is false for a plain field', () => {
    expect(fieldDeclaresGate({ type: 'string', name: 'minutes', label: 'Minutes' })).toBe(false);
  });

  it('is true when a field declares a requires gate', () => {
    const field: SectionField = {
      type: 'string',
      name: 'path',
      label: 'Path',
      requires: [{ when: { truthy: 'enabled' } }],
    };
    expect(fieldDeclaresGate(field)).toBe(true);
  });

  it('is true when a one-of branch leaf declares a forbidden gate', () => {
    const group: SectionField = {
      type: 'one_of',
      name: 'source',
      label: 'Source',
      discriminator: 'source.mode',
      branches: [
        {
          value: 'file',
          label: 'File',
          fields: [
            {
              type: 'string',
              name: 'source.path',
              label: 'Path',
              forbidden: [{ when: { truthy: 'inline' } }],
            },
          ],
        },
      ],
    };
    expect(fieldDeclaresGate(group)).toBe(true);
  });
});
