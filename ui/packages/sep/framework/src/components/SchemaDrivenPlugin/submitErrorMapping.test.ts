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
import { ApiError } from '@sep/api';
import type { FormSection } from '../SchemaFormRenderer/types';
import { EMPTY_SUBMIT_ERROR, mapSubmitError } from './submitErrorMapping';

const SECTIONS: FormSection[] = [
  {
    title: 'Task',
    fields: [
      { type: 'string', name: 'task_name', label: 'Task Name', required: true },
      { type: 'integer', name: 'limit', label: 'Row Limit' },
    ],
  },
];

function http(status: number, detail?: unknown): ApiError {
  return new ApiError({
    kind: 'http',
    status,
    message: `HTTP ${status}`,
    data: { detail },
  });
}

describe('mapSubmitError', () => {
  it('returns the empty state for non-422 errors (toast-only path unchanged)', () => {
    expect(mapSubmitError(http(500), SECTIONS, 'Failed')).toBe(
      EMPTY_SUBMIT_ERROR
    );
    expect(
      mapSubmitError(
        new ApiError({ kind: 'network', message: 'x' }),
        SECTIONS,
        'Failed'
      )
    ).toBe(EMPTY_SUBMIT_ERROR);
    expect(mapSubmitError(new Error('boom'), SECTIONS, 'Failed')).toBe(
      EMPTY_SUBMIT_ERROR
    );
  });

  it('maps a 422 to per-field errors and a banner labelled by field', () => {
    const result = mapSubmitError(
      http(422, [
        { loc: ['body', 'limit'], msg: 'ensure this value is greater than 0' },
        { loc: ['body', 'task_name'], msg: 'field required' },
      ]),
      SECTIONS,
      'Failed to create'
    );

    expect(result.fieldErrors).toEqual([
      { path: 'limit', message: 'ensure this value is greater than 0' },
      { path: 'task_name', message: 'field required' },
    ]);
    expect(result.submitError).toBe(
      [
        'Failed to create',
        '• Row Limit: ensure this value is greater than 0',
        '• Task Name: field required',
      ].join('\n')
    );
  });

  it('surfaces an unmatched/unknown field error in the banner using its raw path', () => {
    const result = mapSubmitError(
      http(422, [{ loc: ['body', 'mystery'], msg: 'not allowed here' }]),
      SECTIONS,
      'Failed'
    );
    expect(result.fieldErrors).toEqual([
      { path: 'mystery', message: 'not allowed here' },
    ]);
    expect(result.submitError).toBe(
      ['Failed', '• mystery: not allowed here'].join('\n')
    );
  });

  it('shows a generic banner for a 422 without a parseable detail array', () => {
    const result = mapSubmitError(
      http(422, 'Unprocessable Entity'),
      SECTIONS,
      'Server rejected it'
    );
    expect(result.fieldErrors).toEqual([]);
    expect(result.submitError).toBe('Server rejected it');
  });
});
