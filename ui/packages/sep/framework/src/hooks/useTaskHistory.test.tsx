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

import { act, renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';

const { mockApiGet, mockApiPost } = vi.hoisted(() => ({
  mockApiGet: vi.fn(),
  mockApiPost: vi.fn(),
}));

vi.mock('@sep/api', async () => {
  const actual = await vi.importActual<typeof import('@sep/api')>('@sep/api');
  return {
    ...actual,
    apiClient: { get: mockApiGet, post: mockApiPost },
  };
});

import {
  useExecuteTask,
  useStopTaskHistory,
  useTaskHistory,
  useTaskHistoryByNames,
} from './useTaskHistory';

const EMPTY_PAGE = {
  items: [],
  total: 0,
  offset: 0,
  limit: 0,
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
  mockApiGet.mockReset();
  mockApiPost.mockReset();
  mockApiGet.mockResolvedValue({ data: EMPTY_PAGE });
  mockApiPost.mockResolvedValue({ data: {} });
});

describe('useTaskHistoryByNames', () => {
  it('does not call the API when taskNames is empty', () => {
    renderHook(() => useTaskHistoryByNames([]), { wrapper: wrapper() });
    expect(mockApiGet).not.toHaveBeenCalled();
  });

  it('does not call the API when enabled=false', () => {
    renderHook(() => useTaskHistoryByNames(['task-a'], { enabled: false }), {
      wrapper: wrapper(),
    });
    expect(mockApiGet).not.toHaveBeenCalled();
  });

  it('calls /api/sep/task-history/ with deduplicated sorted task_names', async () => {
    renderHook(() => useTaskHistoryByNames(['task-b', 'task-a', 'task-b']), {
      wrapper: wrapper(),
    });
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/sep/task-history/', {
        params: {
          task_names: ['task-a', 'task-b'],
        },
        paramsSerializer: { indexes: null },
      });
    });
  });

  it('forwards status, offset, and limit query params', async () => {
    renderHook(
      () =>
        useTaskHistoryByNames(['task-a'], {
          status: 'running',
          offset: 5,
          limit: 10,
        }),
      { wrapper: wrapper() }
    );
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/sep/task-history/', {
        params: {
          status: 'running',
          offset: 5,
          limit: 10,
          task_names: ['task-a'],
        },
        paramsSerializer: { indexes: null },
      });
    });
  });
});

describe('useTaskHistory', () => {
  it('lists through /api/sep/task-history/ and never /api/tasks/...', async () => {
    renderHook(() => useTaskHistory(), { wrapper: wrapper() });
    await waitFor(() => expect(mockApiGet).toHaveBeenCalled());
    const [url] = mockApiGet.mock.calls[0];
    expect(url).toBe('/sep/task-history/');
    expect(
      mockApiGet.mock.calls.every(([u]) => !String(u).startsWith('/tasks/'))
    ).toBe(true);
  });

  it('sends exclude_internal=true when excludeInternal option is set', async () => {
    renderHook(() => useTaskHistory({ excludeInternal: true }), {
      wrapper: wrapper(),
    });
    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/sep/task-history/', {
        params: { exclude_internal: true },
      });
    });
  });

  it('omits exclude_internal from params when excludeInternal is not set', async () => {
    renderHook(() => useTaskHistory(), { wrapper: wrapper() });
    await waitFor(() => expect(mockApiGet).toHaveBeenCalled());
    const params = mockApiGet.mock.calls[0][1]?.params ?? {};
    expect(params).not.toHaveProperty('exclude_internal');
  });
});

describe('useStopTaskHistory', () => {
  it('stops through /api/sep/task-history/{id}/stop/ and never /api/tasks/...', async () => {
    const { result } = renderHook(() => useStopTaskHistory(), {
      wrapper: wrapper(),
    });
    await act(async () => {
      await result.current.mutateAsync(42);
    });
    expect(mockApiPost).toHaveBeenCalledWith('/sep/task-history/42/stop/');
    expect(
      mockApiPost.mock.calls.every(([u]) => !String(u).startsWith('/tasks/'))
    ).toBe(true);
  });
});

describe('useExecuteTask', () => {
  it('encodes slash-containing plugin names per path segment', async () => {
    const { result } = renderHook(
      () => useExecuteTask('backup_mongo/restores'),
      {
        wrapper: wrapper(),
      }
    );

    await act(async () => {
      await result.current.mutateAsync({ taskName: 'my-restore-task' });
    });

    expect(mockApiPost).toHaveBeenCalledWith(
      '/apps/backup_mongo/restores/my-restore-task/execute',
      {}
    );
  });

  it('posts to a single-segment plugin path unchanged', async () => {
    const { result } = renderHook(() => useExecuteTask('backup_mongo'), {
      wrapper: wrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({ taskName: 'my-backup-task' });
    });

    expect(mockApiPost).toHaveBeenCalledWith(
      '/apps/backup_mongo/my-backup-task/execute',
      {}
    );
  });

  it('encodes special characters in task names', async () => {
    const taskName = 'weird/name with spaces&q=?';
    const { result } = renderHook(
      () => useExecuteTask('backup_mongo/restores'),
      {
        wrapper: wrapper(),
      }
    );

    await act(async () => {
      await result.current.mutateAsync({ taskName });
    });

    expect(mockApiPost).toHaveBeenCalledWith(
      `/apps/backup_mongo/restores/${encodeURIComponent(taskName)}/execute`,
      {}
    );
  });

  it('posts an optional execute body for chain wiring', async () => {
    const executeBody = {
      chain_task_names: ['my-alter'],
      chain_on_failure: true,
    };
    const { result } = renderHook(() => useExecuteTask('alters'), {
      wrapper: wrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        taskName: 'my-alter-pre-checks',
        executeBody,
      });
    });

    expect(mockApiPost).toHaveBeenCalledWith(
      '/apps/alters/my-alter-pre-checks/execute',
      executeBody
    );
  });
});
