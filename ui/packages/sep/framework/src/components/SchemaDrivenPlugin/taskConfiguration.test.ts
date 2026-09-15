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
import type { FormSection } from '@sep/api';
import { isDefaultValue, selectConfiguredSettings } from './taskConfiguration';

/** Mirrors the shape mysql_backups serves: a Task section plus gated mode sections. */
const SECTIONS: FormSection[] = [
  {
    title: 'Task',
    fields: [
      { name: 'task_name', label: 'Task Name', type: 'string' },
      {
        name: 'backup_type',
        label: 'Backup Type',
        type: 'choice',
        choices: [
          { value: 'M', label: 'Mydumper' },
          { value: 'X', label: 'XtraBackup' },
        ],
      },
      { name: 'backup_dir', label: 'Backup Directory', type: 'string' },
      {
        name: 'compress',
        label: 'Compress',
        type: 'bool',
        default: false,
      },
    ],
  },
  {
    title: 'Mydumper',
    forbidden: [{ when: { not_equals: { backup_type: 'M' } } }],
    fields: [
      {
        name: 'myloader_threads',
        label: 'Threads',
        type: 'integer',
        default: 4,
      },
    ],
  },
  {
    title: 'XtraBackup',
    forbidden: [{ when: { not_equals: { backup_type: 'X' } } }],
    fields: [
      { name: 'xb_parallel', label: 'Parallel', type: 'integer', default: 4 },
    ],
  },
];

describe('isDefaultValue', () => {
  it('treats absent, null and empty string as equal to an absent default', () => {
    expect(isDefaultValue(undefined, undefined)).toBe(true);
    expect(isDefaultValue(null, undefined)).toBe(true);
    expect(isDefaultValue('', null)).toBe(true);
  });

  it('treats a set value against an absent default as configured', () => {
    expect(isDefaultValue('/backups', undefined)).toBe(false);
  });

  it('treats a cleared value as default — there is nothing to show', () => {
    expect(isDefaultValue('', '/var/backups')).toBe(true);
  });

  it('treats a field absent from the stored body as default', () => {
    expect(isDefaultValue(undefined, 4)).toBe(true);
  });

  it('compares numbers across their string form', () => {
    expect(isDefaultValue(4, '4')).toBe(true);
    expect(isDefaultValue(8, 4)).toBe(false);
  });

  it('keeps false distinct from a true default', () => {
    expect(isDefaultValue(false, false)).toBe(true);
    expect(isDefaultValue(false, true)).toBe(false);
  });

  it('compares arrays as sets, not sequences', () => {
    expect(isDefaultValue(['a', 'b'], ['a', 'b'])).toBe(true);
    expect(isDefaultValue(['a'], ['a', 'b'])).toBe(false);
    // A multi-choice value is a set: reselecting an option reorders the stored
    // list without changing what is selected.
    expect(isDefaultValue(['b', 'a'], ['a', 'b'])).toBe(true);
    // Different members of the same count are still a change.
    expect(isDefaultValue(['a', 'c'], ['a', 'b'])).toBe(false);
    // Duplicates have to match in count, not just in kind.
    expect(isDefaultValue(['a', 'a'], ['a', 'b'])).toBe(false);
  });
});

describe('selectConfiguredSettings', () => {
  it('lists only what differs from the schema default', () => {
    const result = selectConfiguredSettings(SECTIONS, {
      task_name: 'nightly',
      backup_type: 'M',
      backup_dir: '/backups',
      compress: false,
      myloader_threads: 4,
    });

    expect(result).toHaveLength(1);
    expect(result[0].title).toBe('Task');
    // `compress` matches its default and `myloader_threads` matches its own, so
    // neither is listed; the Mydumper section drops out entirely as a result.
    expect(result[0].settings.map((s) => s.name)).toEqual([
      'task_name',
      'backup_type',
      'backup_dir',
    ]);
  });

  it('drops the sections gated out by the chosen backup type', () => {
    const result = selectConfiguredSettings(SECTIONS, {
      backup_type: 'M',
      myloader_threads: 16,
      xb_parallel: 32,
    });

    expect(result.map((s) => s.title)).toEqual(['Task', 'Mydumper']);
    expect(
      result.flatMap((s) => s.settings.map((setting) => setting.name))
    ).not.toContain('xb_parallel');
  });

  it('carries the form label and the choice labels for a configured enum', () => {
    const [task] = selectConfiguredSettings(SECTIONS, { backup_type: 'X' });
    const setting = task.settings.find((s) => s.name === 'backup_type');

    expect(setting?.label).toBe('Backup Type');
    expect(setting?.valueLabels).toEqual({ M: 'Mydumper', X: 'XtraBackup' });
  });

  it('honours excludeNames so the page does not repeat itself', () => {
    const result = selectConfiguredSettings(
      SECTIONS,
      { task_name: 'nightly', backup_type: 'M', backup_dir: '/backups' },
      new Set(['task_name', 'backup_type'])
    );

    expect(result[0].settings.map((s) => s.name)).toEqual(['backup_dir']);
  });

  it('returns nothing when every field is left at its default', () => {
    expect(selectConfiguredSettings(SECTIONS, { compress: false })).toEqual([]);
  });

  it('treats an undeclared boolean default as off', () => {
    const sections: FormSection[] = [
      {
        title: 'Task',
        fields: [{ name: 'verbose', label: 'Verbose', type: 'bool' }],
      },
    ];

    // Otherwise every task that ever submitted the form would list `Verbose: No`.
    expect(selectConfiguredSettings(sections, { verbose: false })).toEqual([]);

    const on = selectConfiguredSettings(sections, { verbose: true });
    expect(on[0].settings.map((s) => s.name)).toEqual(['verbose']);
  });

  it('treats an undeclared multi-choice default as an empty selection', () => {
    const sections: FormSection[] = [
      {
        title: 'Task',
        fields: [
          {
            name: 'databases',
            label: 'Databases',
            type: 'multi_choice',
            choices: [
              { value: 'a', label: 'A' },
              { value: 'b', label: 'B' },
            ],
          },
        ],
      },
    ];

    // The form seeds an omitted multi-choice default as `[]`, so an untouched
    // selection comes back as `[]` and must not read as configured — it would
    // list the label above an empty value.
    expect(selectConfiguredSettings(sections, { databases: [] })).toEqual([]);

    const picked = selectConfiguredSettings(sections, { databases: ['a'] });
    expect(picked[0].settings[0].value).toEqual(['a']);
  });

  it('keeps a falsy-but-declared default distinct from a set value', () => {
    const sections: FormSection[] = [
      {
        title: 'Task',
        fields: [
          { name: 'retries', label: 'Retries', type: 'integer', default: 0 },
        ],
      },
    ];

    expect(selectConfiguredSettings(sections, { retries: 0 })).toEqual([]);
    const set = selectConfiguredSettings(sections, { retries: 3 });
    expect(set[0].settings[0].value).toBe(3);
    // A zero explicitly set against no declared default is still a choice.
    const noDefault: FormSection[] = [
      {
        title: 'Task',
        fields: [{ name: 'retries', label: 'Retries', type: 'integer' }],
      },
    ];
    expect(
      selectConfiguredSettings(noDefault, { retries: 0 })[0].settings[0].value
    ).toBe(0);
  });

  it('does not confuse a numeric default with a boolean value', () => {
    expect(isDefaultValue(false, 0)).toBe(false);
    expect(isDefaultValue(0, false)).toBe(false);
  });

  it('drops a field its own forbidden gate hides', () => {
    const sections: FormSection[] = [
      {
        title: 'Task',
        fields: [
          { name: 'upload', label: 'Upload', type: 'bool', default: false },
          {
            name: 's3_bucket',
            label: 'S3 Bucket',
            type: 'string',
            forbidden: [{ when: { falsy: 'upload' } }],
          },
        ],
      },
    ];

    // A leftover bucket from a since-disabled upload is not this task's config.
    expect(
      selectConfiguredSettings(sections, {
        upload: false,
        s3_bucket: 'leftover',
      })
    ).toEqual([]);

    const enabled = selectConfiguredSettings(sections, {
      upload: true,
      s3_bucket: 'backups',
    });
    expect(enabled[0].settings.map((s) => s.name)).toEqual([
      'upload',
      's3_bucket',
    ]);
  });

  it('resolves a dotted discriminator and dotted branch leaves', () => {
    // The shape the schema types call out for a one-of nested on the write
    // model: the discriminator and the branch leaves are dotted paths, and the
    // stored body carries them nested.
    const sections: FormSection[] = [
      {
        title: 'Source',
        fields: [
          {
            type: 'one_of',
            name: 'source',
            label: 'Source',
            discriminator: 'source.mode',
            branches: [
              {
                value: 'local',
                label: 'Local',
                fields: [
                  { name: 'source.path', label: 'Path', type: 'string' },
                ],
              },
              {
                value: 's3',
                label: 'S3',
                fields: [
                  { name: 'source.bucket', label: 'Bucket', type: 'string' },
                ],
              },
            ],
          },
        ],
      },
    ];

    const result = selectConfiguredSettings(sections, {
      source: { mode: 's3', bucket: 'nightly' },
    });

    expect(result[0].settings).toEqual([
      {
        name: 'source.bucket',
        label: 'Bucket',
        value: 'nightly',
        valueLabels: undefined,
      },
    ]);
  });

  it('takes only the chosen branch of a one_of group', () => {
    const sections: FormSection[] = [
      {
        title: 'Source',
        fields: [
          {
            type: 'one_of',
            name: 'source',
            label: 'Source',
            discriminator: 'mode',
            branches: [
              {
                value: 'local',
                label: 'Local',
                fields: [{ name: 'path', label: 'Path', type: 'string' }],
              },
              {
                value: 's3',
                label: 'S3',
                fields: [{ name: 'bucket', label: 'Bucket', type: 'string' }],
              },
            ],
          },
        ],
      },
    ];

    const result = selectConfiguredSettings(sections, {
      mode: 's3',
      bucket: 'nightly',
      path: '/stale',
    });

    expect(result[0].settings.map((s) => s.name)).toEqual(['bucket']);
  });
});
