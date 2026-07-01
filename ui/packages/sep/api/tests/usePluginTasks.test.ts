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
  buildEntityItemPath,
  isBackendUnavailable,
  unwrapTasks,
} from '../src/hooks/usePluginTasks';
import { ApiError } from '../src/errors';

describe('unwrapTasks', () => {
  it('returns a legacy array response as-is', () => {
    const tasks = [{ name: 'a' }, { name: 'b' }];
    expect(unwrapTasks(tasks)).toEqual(tasks);
  });

  it('unwraps a PaginatedResponse envelope to its items', () => {
    const tasks = [{ name: 'a' }];
    expect(
      unwrapTasks({ items: tasks, total: 1, offset: 0, limit: 50 })
    ).toEqual(tasks);
  });

  it('returns an empty array for an empty envelope', () => {
    expect(unwrapTasks({ items: [], total: 0, offset: 0, limit: 50 })).toEqual(
      []
    );
  });

  it('returns [] when items is null', () => {
    expect(unwrapTasks({ items: null } as never)).toEqual([]);
  });

  it('returns [] when payload is an empty object', () => {
    expect(unwrapTasks({} as never)).toEqual([]);
  });

  it('returns [] when payload is null or undefined', () => {
    expect(unwrapTasks(null as never)).toEqual([]);
    expect(unwrapTasks(undefined as never)).toEqual([]);
  });
});

describe('buildEntityItemPath', () => {
  it('encodes the id segment to prevent path traversal', () => {
    // A backend bug or attacker-controlled id with "/" or ".." would otherwise
    // change which resource (or even which plugin) the request targets.
    expect(buildEntityItemPath('mysql_backups', 'backups', '../secret')).toBe(
      '/apps/mysql_backups/backups/..%2Fsecret'
    );
    expect(buildEntityItemPath('mysql_backups', 'backups', 'a/b')).toBe(
      '/apps/mysql_backups/backups/a%2Fb'
    );
  });

  it('encodes whitespace, unicode, and reserved characters', () => {
    expect(buildEntityItemPath('p', 'e', 'a b')).toBe('/apps/p/e/a%20b');
    expect(buildEntityItemPath('p', 'e', '#frag')).toBe('/apps/p/e/%23frag');
    expect(buildEntityItemPath('p', 'e', '?q=1')).toBe('/apps/p/e/%3Fq%3D1');
    expect(buildEntityItemPath('p', 'e', 'café')).toBe('/apps/p/e/caf%C3%A9');
  });

  it('passes bare alphanumeric / UUID ids through unchanged', () => {
    expect(buildEntityItemPath('p', 'e', '42')).toBe('/apps/p/e/42');
    expect(
      buildEntityItemPath('p', 'e', '550e8400-e29b-41d4-a716-446655440000')
    ).toBe('/apps/p/e/550e8400-e29b-41d4-a716-446655440000');
  });
});

describe('isBackendUnavailable (mock-fallback gate)', () => {
  // The mock-fallback layer in usePluginTasks only swallows errors that look
  // like the backend is unreachable. A 401/403/404 must always propagate so
  // mock data never masks real auth or routing problems even in dev builds.
  it.each([
    ['network kind', new ApiError({ kind: 'network', message: 'net' })],
    ['500', new ApiError({ kind: 'http', status: 500, message: '500' })],
    ['502', new ApiError({ kind: 'http', status: 502, message: '502' })],
    ['503', new ApiError({ kind: 'http', status: 503, message: '503' })],
    ['504', new ApiError({ kind: 'http', status: 504, message: '504' })],
  ])('treats %s as backend-unavailable', (_label, err) => {
    expect(isBackendUnavailable(err)).toBe(true);
  });

  it.each([
    ['400', new ApiError({ kind: 'http', status: 400, message: '400' })],
    ['401', new ApiError({ kind: 'http', status: 401, message: '401' })],
    ['403', new ApiError({ kind: 'http', status: 403, message: '403' })],
    ['404', new ApiError({ kind: 'http', status: 404, message: '404' })],
    ['422', new ApiError({ kind: 'http', status: 422, message: '422' })],
    ['499', new ApiError({ kind: 'http', status: 499, message: '499' })],
  ])('does NOT mask deterministic 4xx (%s) with mock data', (_label, err) => {
    expect(isBackendUnavailable(err)).toBe(false);
  });

  it('does not mask timeout, canceled, or unknown kinds as backend-down', () => {
    expect(
      isBackendUnavailable(new ApiError({ kind: 'timeout', message: 't' }))
    ).toBe(false);
    expect(
      isBackendUnavailable(new ApiError({ kind: 'canceled', message: 'c' }))
    ).toBe(false);
    expect(
      isBackendUnavailable(new ApiError({ kind: 'unknown', message: '?' }))
    ).toBe(false);
  });

  it('does not classify a plain Error as backend-unavailable', () => {
    // Plain Errors aren't an axios surface — they shouldn't reach the gate,
    // but if they do, the fallback must not engage.
    expect(isBackendUnavailable(new Error('boom'))).toBe(false);
    expect(isBackendUnavailable(undefined)).toBe(false);
    expect(isBackendUnavailable(null)).toBe(false);
    expect(isBackendUnavailable('string')).toBe(false);
  });

  it('does not treat http error with missing status as backend-down', () => {
    // `kind === 'http'` but status undefined would otherwise compare
    // `undefined >= 500 === false`, but pin the behaviour explicitly.
    expect(
      isBackendUnavailable(new ApiError({ kind: 'http', message: 'mystery' }))
    ).toBe(false);
  });
});
