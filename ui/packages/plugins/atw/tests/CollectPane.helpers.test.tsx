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
import {
  buildBatchPayload,
  fieldDeclaresGate,
  filterSnippetOptions,
  mergeSnippetOptions,
  namespaceField,
  snippetMatchesTerm,
} from '../src/CollectPane';
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
      []
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
      snippets
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
      [snippets[0]]
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

describe('mergeSnippetOptions', () => {
  it('dedupes on filename, not title, keeping first-seen order', () => {
    const selectedRow: AtwSnippetSummary = {
      name: 'a.sh',
      title: 'A',
      description: '',
    };
    const categoryRow: AtwSnippetSummary = {
      name: 'a.sh',
      title: 'A (renamed)',
      description: '',
    };
    const searchRow: AtwSnippetSummary = {
      name: 'c.sh',
      title: 'A',
      description: '',
    };

    const merged = mergeSnippetOptions(
      [selectedRow],
      [categoryRow],
      [searchRow]
    );

    expect(merged.map((snippet) => snippet.name)).toEqual(['a.sh', 'c.sh']);
    // Same filename from a later source wins the row, so the picker shows the
    // freshest metadata while the option itself stays single.
    expect(merged[0].title).toBe('A (renamed)');
  });

  it('returns an empty list when every source is empty', () => {
    expect(mergeSnippetOptions([], [], [])).toEqual([]);
  });
});

describe('snippetMatchesTerm', () => {
  const snippet: AtwSnippetSummary = {
    name: 'ops/pt-summary.sh',
    title: 'PT Summary',
    description: 'Collects a percona-toolkit system summary.',
  };

  it('matches title, filename and description case-insensitively', () => {
    expect(snippetMatchesTerm(snippet, 'pt summary')).toBe(true);
    expect(snippetMatchesTerm(snippet, 'OPS/PT-')).toBe(true);
    expect(snippetMatchesTerm(snippet, 'percona-toolkit')).toBe(true);
  });

  it('keeps everything for an empty or blank term', () => {
    expect(snippetMatchesTerm(snippet, '')).toBe(true);
    expect(snippetMatchesTerm(snippet, '   ')).toBe(true);
  });

  it('rejects a term present in none of the three fields', () => {
    expect(snippetMatchesTerm(snippet, 'mongodb')).toBe(false);
  });
});

describe('filterSnippetOptions', () => {
  const category: AtwSnippetSummary = {
    name: 'diag/slow-query.sh',
    title: 'Slow Query Diagnostics',
    description: '',
  };
  const searched: AtwSnippetSummary = {
    name: 'ops/pt-summary.sh',
    title: 'PT Summary',
    description: 'Collects a percona-toolkit system summary.',
  };

  it('keeps a server match whose label does not contain the term', () => {
    const filtered = filterSnippetOptions(
      [category, searched],
      'percona-toolkit',
      new Set([searched.name])
    );
    expect(filtered).toEqual([searched]);
  });

  it('still filters category options the server never saw', () => {
    const filtered = filterSnippetOptions(
      [category, searched],
      'slow',
      new Set([searched.name])
    );
    // The searched row survives on server provenance, the category row on its title.
    expect(filtered.map((snippet) => snippet.name)).toEqual([
      category.name,
      searched.name,
    ]);
  });

  it('passes every option through when nothing is typed', () => {
    expect(filterSnippetOptions([category, searched], '  ', new Set())).toEqual(
      [category, searched]
    );
  });
});

describe('namespaceField', () => {
  it('prefixes a leaf field name', () => {
    const field: SectionField = {
      type: 'string',
      name: 'minutes',
      label: 'Minutes',
    };
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
      expect(result.branches[0].fields[0].name).toBe(
        'overrides.snip1.source.path'
      );
    }
  });
});

describe('fieldDeclaresGate', () => {
  it('is false for a plain field', () => {
    expect(
      fieldDeclaresGate({ type: 'string', name: 'minutes', label: 'Minutes' })
    ).toBe(false);
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
