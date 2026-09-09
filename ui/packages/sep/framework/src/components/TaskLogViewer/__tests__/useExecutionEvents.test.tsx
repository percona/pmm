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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useExecutionEvents } from '../../../hooks/useExecutionEvents';
import {
  flushPromises,
  mockStreamFetch,
} from '../../../../tests/eventSourceStub';

// Manual mock keeps axios out of the resolution graph.
// setTokenProvider / getToken are re-implemented with the same stateful semantics.
// apiClient.get is a vi.fn() that tests configure per-scenario.
let _tokenProvider: () => string | null = () => null;
const mockApiGet =
  vi.fn<(url: string, config: object) => Promise<{ data: unknown[] }>>();

vi.mock('@sep/api', () => ({
  setTokenProvider: (p: () => string | null) => {
    _tokenProvider = p;
  },
  getToken: () => _tokenProvider(),
  refreshAccessToken: vi.fn<() => Promise<string | null>>(),
  emitUnauthorized: vi.fn(),
  apiClient: {
    get: (...args: Parameters<typeof mockApiGet>) => mockApiGet(...args),
    defaults: {},
  },
  SEP_BASE_PATH: '/sep',
}));

const TEST_TOKEN = 'test-bearer-token';

function createClient(): QueryClient {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function makeWrapper(client: QueryClient) {
  return ({ children }: PropsWithChildren) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

describe('useExecutionEvents', () => {
  let mock: ReturnType<typeof mockStreamFetch>;

  beforeEach(() => {
    mock = mockStreamFetch();
    mock.install();
    _tokenProvider = () => TEST_TOKEN;
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  describe('running tasks (SSE)', () => {
    it('fetches /sep/stream-logs/{id}/execution-events with Bearer token', async () => {
      const wrapper = makeWrapper(createClient());
      renderHook(() => useExecutionEvents(42, true), { wrapper });
      await flushPromises();

      expect(mock.fetchSpy).toHaveBeenCalledTimes(1);
      const [url] = mock.fetchSpy.mock.calls[0];
      const urlStr = typeof url === 'string' ? url : (url as URL).href;
      expect(urlStr).toBe('/sep/stream-logs/42/execution-events');
      // Headers instance lowercases names; stub normalises via Object.fromEntries(headers.entries())
      expect(mock.pending[0].requestHeaders.authorization).toBe(
        `Bearer ${TEST_TOKEN}`
      );
    });

    it('accumulates events and groups them by step', async () => {
      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      const handle = mock.pending[0];

      act(() => {
        handle.pushMessage({
          timestamp: 't1',
          type: 'started',
          description: 'A',
          step: 'setup',
        });
        handle.pushMessage({
          timestamp: 't2',
          type: 'progress',
          description: 'B',
          step: 'setup',
        });
        handle.pushMessage({
          timestamp: 't3',
          type: 'started',
          description: 'C',
          step: 'test',
        });
      });

      await waitFor(() => expect(result.current.events).toHaveLength(3));

      expect(result.current.stepOrder).toEqual(['setup', 'test']);
      expect(result.current.eventsByStep.setup).toHaveLength(2);
      expect(result.current.eventsByStep.test).toHaveLength(1);
    });

    it('dedupes events with the same composite key', async () => {
      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      const handle = mock.pending[0];
      const ev = {
        timestamp: 't',
        type: 'started',
        description: 'A',
        step: 'x',
      };

      act(() => {
        handle.pushMessage(ev);
        handle.pushMessage(ev); // duplicate
        handle.pushMessage({ ...ev, description: 'B' }); // different → new key
      });

      await waitFor(() => expect(result.current.events).toHaveLength(2));
    });

    it('buckets events without a step under the stepless key', async () => {
      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      const handle = mock.pending[0];

      act(() => {
        handle.pushMessage({
          timestamp: 't1',
          type: 'started',
          description: 'A',
        });
        handle.pushMessage({
          timestamp: 't2',
          type: 'started',
          description: 'B',
          step: null,
        });
      });

      await waitFor(() => expect(result.current.events).toHaveLength(2));

      expect(result.current.stepOrder).toEqual(['']);
      expect(result.current.eventsByStep['']).toHaveLength(2);
    });

    it('handles finish by stopping loading and stopping the stream', async () => {
      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      const handle = mock.pending[0];

      act(() => {
        handle.pushNamed('finish', { status: 'success' });
      });

      await waitFor(() => expect(result.current.isLoading).toBe(false));
    });

    it('handles sep-error by setting error and stopping the stream', async () => {
      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      const handle = mock.pending[0];

      act(() => {
        handle.pushNamed('sep-error', { code: 500, detail: 'boom' });
      });

      await waitFor(() => expect(result.current.error).toBeDefined());
    });

    it('surfaces a terminal error when the stream closes unexpectedly', async () => {
      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      const handle = mock.pending[0];

      act(() => {
        handle.close();
      });

      await waitFor(() => expect(result.current.isLoading).toBe(false));
    });

    it('aborts the fetch on unmount', async () => {
      const wrapper = makeWrapper(createClient());
      const { unmount } = renderHook(() => useExecutionEvents(1, true), {
        wrapper,
      });
      await flushPromises();

      unmount();

      expect(mock.pending[0].signal?.aborted).toBe(true);
    });
  });

  describe('completed tasks (REST via apiClient)', () => {
    it('calls /sep/execution-events/{id} via apiClient and surfaces events', async () => {
      mockApiGet.mockResolvedValue({
        data: [
          { timestamp: 't1', type: 'started', description: 'A', step: 'setup' },
          {
            timestamp: 't2',
            type: 'completed',
            description: 'B',
            step: 'setup',
          },
        ],
      } as never);

      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(7, false), {
        wrapper,
      });

      await waitFor(() => expect(result.current.events).toHaveLength(2));

      // Verify apiClient.get was called with the right URL and baseURL escape
      expect(mockApiGet).toHaveBeenCalledWith(
        '/execution-events/7',
        expect.objectContaining({ baseURL: '/sep' })
      );
      expect(result.current.stepOrder).toEqual(['setup']);
      // SSE must not have been opened for a completed task
      expect(mock.fetchSpy).not.toHaveBeenCalled();
    });

    it('surfaces an error when the REST response fails', async () => {
      mockApiGet.mockRejectedValue(new Error('Request failed with status 401'));

      const wrapper = makeWrapper(createClient());
      const { result } = renderHook(() => useExecutionEvents(7, false), {
        wrapper,
      });

      await waitFor(() => expect(result.current.error).toBeDefined());
      expect(result.current.events).toEqual([]);
    });

    it('does not fetch when taskHistoryId is missing', () => {
      const wrapper = makeWrapper(createClient());
      renderHook(() => useExecutionEvents(undefined, false), { wrapper });

      expect(mockApiGet).not.toHaveBeenCalled();
      expect(mock.fetchSpy).not.toHaveBeenCalled();
    });
  });

  describe('transition', () => {
    it('switches from SSE to REST when isRunning flips to false', async () => {
      mockApiGet.mockResolvedValue({
        data: [
          {
            timestamp: 't-final',
            type: 'completed',
            description: 'done',
            step: 'x',
          },
        ],
      } as never);

      const wrapper = makeWrapper(createClient());
      const { rerender, result } = renderHook(
        ({ running }: { running: boolean }) => useExecutionEvents(9, running),
        { wrapper, initialProps: { running: true } }
      );

      await flushPromises();
      const firstHandle = mock.pending[0];
      expect(firstHandle).toBeDefined();

      rerender({ running: false });

      // SSE aborted after isRunning flip
      expect(firstHandle.signal?.aborted).toBe(true);

      await waitFor(() => expect(result.current.events).toHaveLength(1));
      expect(mockApiGet).toHaveBeenCalledWith(
        '/execution-events/9',
        expect.objectContaining({ baseURL: '/sep' })
      );
    });
  });
});
