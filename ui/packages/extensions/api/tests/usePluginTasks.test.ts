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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  buildEntityItemPath,
  DEFAULT_PLUGIN_LIST_LIMIT,
  DEFAULT_PLUGIN_LIST_OFFSET,
  fetchPluginTasksList,
  fetchPluginEntityList,
  isBackendUnavailable,
  isRunningStatus,
  MAX_FETCH_ALL_PAGES,
  normalizePluginListResponse,
  RUNNING_STATUSES,
  taskListRefetchInterval,
} from '../src/hooks/usePluginTasks';
import { ApiError } from '../src/errors';

vi.mock('../src/client', () => ({
  apiClient: {
    get: vi.fn(),
  },
}));

import { apiClient } from '../src/client';

const getMock = vi.mocked(apiClient.get);

beforeEach(() => {
  getMock.mockReset();
});

describe('normalizePluginListResponse', () => {
  it('returns items with pagination null for a bare array', () => {
    const items = [{ name: 'a' }, { name: 'b' }];
    expect(normalizePluginListResponse(items)).toEqual({
      items,
      pagination: null,
    });
  });

  it('preserves pagination metadata for a full envelope', () => {
    const items = [{ name: 'a' }];
    expect(
      normalizePluginListResponse({ items, total: 120, offset: 50, limit: 50 })
    ).toEqual({
      items,
      pagination: { total: 120, offset: 50, limit: 50 },
    });
  });

  it('unwraps a partial items-only envelope without pagination metadata', () => {
    const items = [{ name: 'a' }];
    expect(normalizePluginListResponse({ items })).toEqual({
      items,
      pagination: null,
    });
  });

  it('returns empty items for null, undefined, or empty object', () => {
    expect(normalizePluginListResponse(null)).toEqual({
      items: [],
      pagination: null,
    });
    expect(normalizePluginListResponse(undefined)).toEqual({
      items: [],
      pagination: null,
    });
    expect(normalizePluginListResponse({} as never)).toEqual({
      items: [],
      pagination: null,
    });
  });

  it('returns empty items when items is null in a partial envelope', () => {
    expect(normalizePluginListResponse({ items: null })).toEqual({
      items: [],
      pagination: null,
    });
  });
});

describe('fetchPluginTasksList', () => {
  it('requests offset/limit query params for a single page', async () => {
    getMock.mockResolvedValue({
      data: { items: [{ name: 'a' }], total: 1, offset: 50, limit: 50 },
    });

    const result = await fetchPluginTasksList('mysql_backups', {
      offset: 50,
      limit: 50,
    });

    expect(getMock).toHaveBeenCalledWith('/apps/mysql_backups/', {
      params: { offset: 50, limit: 50 },
    });
    expect(result).toEqual({
      items: [{ name: 'a' }],
      pagination: { total: 1, offset: 50, limit: 50 },
    });
  });

  it('defaults offset/limit to backend defaults', async () => {
    getMock.mockResolvedValue({ data: [] });

    await fetchPluginTasksList('mysql_backups');

    expect(getMock).toHaveBeenCalledWith('/apps/mysql_backups/', {
      params: {
        offset: DEFAULT_PLUGIN_LIST_OFFSET,
        limit: DEFAULT_PLUGIN_LIST_LIMIT,
      },
    });
  });

  it('fetchAllPages loops until total is reached', async () => {
    getMock
      .mockResolvedValueOnce({
        data: {
          items: [{ name: 'a' }],
          total: 2,
          offset: 0,
          limit: DEFAULT_PLUGIN_LIST_LIMIT,
        },
      })
      .mockResolvedValueOnce({
        data: {
          items: [{ name: 'b' }],
          total: 2,
          offset: 1,
          limit: DEFAULT_PLUGIN_LIST_LIMIT,
        },
      });

    const result = await fetchPluginTasksList('mysql_backups', {
      fetchAllPages: true,
    });

    expect(getMock).toHaveBeenCalledWith('/apps/mysql_backups/', {
      params: { offset: 0, limit: DEFAULT_PLUGIN_LIST_LIMIT },
    });
    expect(getMock).toHaveBeenCalledTimes(2);
    expect(result).toEqual({
      items: [{ name: 'a' }, { name: 'b' }],
      pagination: null,
    });
  });
  it('fetchAllPages returns the bare array on NO_PAGINATION routes', async () => {
    const items = [{ name: 'a' }, { name: 'b' }];
    getMock.mockResolvedValue({ data: items });

    const result = await fetchPluginTasksList('legacy_app', {
      fetchAllPages: true,
    });

    expect(getMock).toHaveBeenCalledTimes(1);
    expect(result).toEqual({ items, pagination: null });
  });

  it('fetchAllPages sets truncated and warns when the page cap is hit', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    getMock.mockImplementation(async (path, config) => {
      const offset = Number(
        (config as { params: { offset: number } }).params.offset
      );
      return {
        data: {
          items: Array.from({ length: DEFAULT_PLUGIN_LIST_LIMIT }, (_, i) => ({
            name: `t${offset + i}`,
          })),
          total: 3000,
          offset,
          limit: DEFAULT_PLUGIN_LIST_LIMIT,
        },
      };
    });

    const result = await fetchPluginTasksList('mysql_backups', {
      fetchAllPages: true,
    });

    expect(getMock).toHaveBeenCalledTimes(MAX_FETCH_ALL_PAGES);
    expect(result.items).toHaveLength(
      MAX_FETCH_ALL_PAGES * DEFAULT_PLUGIN_LIST_LIMIT
    );
    expect(result.truncated).toBe(true);
    expect(warn).toHaveBeenCalledWith(
      expect.stringContaining('fetchAllPages truncated')
    );
    warn.mockRestore();
  });
});

describe('fetchPluginEntityList', () => {
  it('requests the entity list path with offset/limit', async () => {
    getMock.mockResolvedValue({
      data: { items: [{ id: 1 }], total: 1, offset: 0, limit: 50 },
    });

    const result = await fetchPluginEntityList('inventory', 'nodes', {
      offset: 0,
      limit: 50,
    });

    expect(getMock).toHaveBeenCalledWith('/apps/inventory/nodes/', {
      params: { offset: 0, limit: 50 },
    });
    expect(result).toEqual({
      items: [{ id: 1 }],
      pagination: { total: 1, offset: 0, limit: 50 },
    });
  });

  it('returns a bare array for NO_PAGINATION entity routes', async () => {
    const items = [{ id: 1 }, { id: 2 }];
    getMock.mockResolvedValue({ data: items });

    const result = await fetchPluginEntityList('legacy_app', 'items');

    expect(result).toEqual({ items, pagination: null });
  });

  it('fetchAllPages loops entity pages until total is reached', async () => {
    getMock
      .mockResolvedValueOnce({
        data: {
          items: [{ id: 1 }],
          total: 2,
          offset: 0,
          limit: DEFAULT_PLUGIN_LIST_LIMIT,
        },
      })
      .mockResolvedValueOnce({
        data: {
          items: [{ id: 2 }],
          total: 2,
          offset: 1,
          limit: DEFAULT_PLUGIN_LIST_LIMIT,
        },
      });

    const result = await fetchPluginEntityList('inventory', 'nodes', {
      fetchAllPages: true,
    });

    expect(getMock).toHaveBeenCalledWith('/apps/inventory/nodes/', {
      params: { offset: 0, limit: DEFAULT_PLUGIN_LIST_LIMIT },
    });
    expect(getMock).toHaveBeenCalledTimes(2);
    expect(result).toEqual({
      items: [{ id: 1 }, { id: 2 }],
      pagination: null,
    });
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

describe('isRunningStatus / RUNNING_STATUSES', () => {
  it('treats running and pending as still-executing', () => {
    expect(isRunningStatus('running')).toBe(true);
    expect(isRunningStatus('pending')).toBe(true);
  });

  it('treats terminal statuses as not running', () => {
    for (const status of ['success', 'failed', 'stopped'] as const) {
      expect(isRunningStatus(status)).toBe(false);
    }
  });

  it('RUNNING_STATUSES holds exactly running and pending', () => {
    expect([...RUNNING_STATUSES].sort()).toEqual(['pending', 'running']);
  });
});

describe('taskListRefetchInterval (poll-while-running)', () => {
  const polling = 5000;

  it('polls while any row is running', () => {
    const rows = [{ status: 'success' }, { status: 'running' }];
    expect(taskListRefetchInterval(rows, polling, false)).toBe(polling);
  });

  it('polls while any row is pending', () => {
    expect(
      taskListRefetchInterval([{ status: 'pending' }], polling, false)
    ).toBe(polling);
  });

  it('stays idle when no row is running (no repeat requests)', () => {
    const rows = [
      { status: 'success' },
      { status: 'failed' },
      { status: 'stopped' },
    ];
    expect(taskListRefetchInterval(rows, polling, false)).toBe(false);
  });

  it('stays idle for an empty list', () => {
    expect(taskListRefetchInterval([], polling, false)).toBe(false);
  });

  it('stays idle before the first fetch resolves', () => {
    expect(taskListRefetchInterval(undefined, polling, false)).toBe(false);
  });

  it('never polls when disabled, even with a running row', () => {
    expect(
      taskListRefetchInterval([{ status: 'running' }], polling, true)
    ).toBe(false);
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
