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

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';

const { mockSepGet } = vi.hoisted(() => ({ mockSepGet: vi.fn() }));

vi.mock('@sep/api', async () => {
  const actual = await vi.importActual<typeof import('@sep/api')>('@sep/api');
  return {
    ...actual,
    sepApi: { GET: mockSepGet },
  };
});

import { useTaskStats } from './useTaskStats';

const SUCCESS_RESPONSE = {
  data: {
    engine: 'nomad',
    total: 1,
    status: { pass: 1, fail: 0 },
    duration: { average_seconds: 1, last_seconds: 1, total_seconds: 1 },
    last_finished_at: new Date().toISOString(),
  },
  error: undefined,
  response: { ok: true, status: 200, statusText: 'OK' } as Response,
};

function wrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  mockSepGet.mockReset();
});

describe('useTaskStats', () => {
  it('does not call the API when taskName is undefined', () => {
    mockSepGet.mockResolvedValue(SUCCESS_RESPONSE);
    renderHook(() => useTaskStats(undefined), { wrapper: wrapper() });
    expect(mockSepGet).not.toHaveBeenCalled();
  });

  it('does not call the API for whitespace-only taskName', () => {
    mockSepGet.mockResolvedValue(SUCCESS_RESPONSE);
    renderHook(() => useTaskStats('   '), { wrapper: wrapper() });
    expect(mockSepGet).not.toHaveBeenCalled();
  });

  it('does not call the API when enabled=false', () => {
    mockSepGet.mockResolvedValue(SUCCESS_RESPONSE);
    renderHook(() => useTaskStats('foo', false), { wrapper: wrapper() });
    expect(mockSepGet).not.toHaveBeenCalled();
  });

  it('calls /api/sep/task-stats/{task_name} with the trimmed name in path params', async () => {
    mockSepGet.mockResolvedValue(SUCCESS_RESPONSE);
    renderHook(() => useTaskStats('  my-task  '), { wrapper: wrapper() });
    await waitFor(() => {
      expect(mockSepGet).toHaveBeenCalledWith(
        '/api/sep/task-stats/{task_name}',
        {
          params: { path: { task_name: 'my-task' } },
        }
      );
    });
  });

  it('passes URL-special characters through to the openapi-fetch client', async () => {
    mockSepGet.mockResolvedValue(SUCCESS_RESPONSE);
    renderHook(() => useTaskStats('weird/name with spaces&q=?'), {
      wrapper: wrapper(),
    });
    await waitFor(() => {
      expect(mockSepGet).toHaveBeenCalledWith(
        '/api/sep/task-stats/{task_name}',
        {
          params: { path: { task_name: 'weird/name with spaces&q=?' } },
        }
      );
    });
  });

  it('throws an ApiError with the response status (no retry on 401)', async () => {
    mockSepGet.mockResolvedValue({
      data: undefined,
      error: { detail: 'nope' },
      response: {
        ok: false,
        status: 401,
        statusText: 'Unauthorized',
        url: 'http://localhost/api/sep/task-stats/foo',
      } as Response,
    });
    const { result } = renderHook(() => useTaskStats('foo'), {
      wrapper: wrapper(),
    });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    const { ApiError } = await import('@sep/api');
    expect(result.current.error).toBeInstanceOf(ApiError);
    const err = result.current.error as InstanceType<typeof ApiError>;
    expect(err.status).toBe(401);
    expect(err.message).toBe('nope');
    expect(mockSepGet).toHaveBeenCalledTimes(1);
  });

  it('throws an ApiError with status 502 and does not retry on upstream failure', async () => {
    mockSepGet.mockResolvedValue({
      data: undefined,
      error: { detail: 'tasks unreachable' },
      response: {
        ok: false,
        status: 502,
        statusText: 'Bad Gateway',
        url: 'http://localhost/api/sep/task-stats/foo',
      } as Response,
    });
    const { result } = renderHook(() => useTaskStats('foo'), {
      wrapper: wrapper(),
    });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    const { ApiError } = await import('@sep/api');
    expect(result.current.error).toBeInstanceOf(ApiError);
    const err = result.current.error as InstanceType<typeof ApiError>;
    expect(err.status).toBe(502);
    expect(err.message).toBe('tasks unreachable');
    expect(mockSepGet).toHaveBeenCalledTimes(1);
  });
});
