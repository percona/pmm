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
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useTaskLogs } from '../../../hooks/useTaskLogs';
import {
  flushPromises,
  mockStreamFetch,
} from '../../../../tests/eventSourceStub';

// Manual mock keeps axios out of the resolution graph.
// setTokenProvider / getToken are re-implemented with the same stateful semantics.
let _tokenProvider: () => string | null = () => null;
vi.mock('@sep/api', () => ({
  setTokenProvider: (p: () => string | null) => {
    _tokenProvider = p;
  },
  getToken: () => _tokenProvider(),
  refreshAccessToken: vi.fn<() => Promise<string | null>>(),
  emitUnauthorized: vi.fn(),
  SEP_BASE_PATH: '/sep',
}));

const TEST_TOKEN = 'test-bearer-token';

describe('useTaskLogs', () => {
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

  it('fetches /sep/stream-logs/{id} with Bearer token', async () => {
    renderHook(() => useTaskLogs(42));
    await flushPromises();

    expect(mock.fetchSpy).toHaveBeenCalledTimes(1);
    const [url] = mock.fetchSpy.mock.calls[0];
    const urlStr = typeof url === 'string' ? url : (url as URL).href;
    expect(urlStr).toBe('/sep/stream-logs/42');
    // Headers instance lowercases names; stub normalises via Object.fromEntries(headers.entries())
    expect(mock.pending[0].requestHeaders.authorization).toBe(
      `Bearer ${TEST_TOKEN}`
    );
  });

  it('includes ?tail= when tail is set', async () => {
    renderHook(() => useTaskLogs(42, 1000));
    await flushPromises();

    const [url] = mock.fetchSpy.mock.calls[0];
    const urlStr = typeof url === 'string' ? url : (url as URL).href;
    expect(urlStr).toBe('/sep/stream-logs/42?tail=1000');
  });

  it('omits tail query param when tail is undefined (All)', async () => {
    renderHook(() => useTaskLogs(42, undefined));
    await flushPromises();

    const [url] = mock.fetchSpy.mock.calls[0];
    const urlStr = typeof url === 'string' ? url : (url as URL).href;
    expect(urlStr).toBe('/sep/stream-logs/42');
  });

  it('opens a fresh fetch when tail changes', async () => {
    const { rerender } = renderHook(
      ({ tail }: { tail?: number }) => useTaskLogs(1, tail),
      {
        initialProps: { tail: 1000 as number | undefined },
      }
    );
    await flushPromises();

    expect(mock.fetchSpy).toHaveBeenCalledTimes(1);
    let urlStr =
      typeof mock.fetchSpy.mock.calls[0][0] === 'string'
        ? mock.fetchSpy.mock.calls[0][0]
        : (mock.fetchSpy.mock.calls[0][0] as URL).href;
    expect(urlStr).toBe('/sep/stream-logs/1?tail=1000');

    rerender({ tail: 5000 });
    await flushPromises();

    expect(mock.fetchSpy).toHaveBeenCalledTimes(2);
    urlStr =
      typeof mock.fetchSpy.mock.calls[1][0] === 'string'
        ? mock.fetchSpy.mock.calls[1][0]
        : (mock.fetchSpy.mock.calls[1][0] as URL).href;
    expect(urlStr).toBe('/sep/stream-logs/1?tail=5000');
  });

  it('omits Authorization header when no token is available', async () => {
    _tokenProvider = () => null;
    renderHook(() => useTaskLogs(42));
    await flushPromises();

    expect(mock.pending[0].requestHeaders.authorization).toBeUndefined();
  });

  it('accumulates log text grouped by step and type', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];

    act(() => {
      handle.pushMessage({
        msg: 'hello ',
        step: 'step1',
        type: 'stdout',
        offset: 1,
      });
      handle.pushMessage({
        msg: 'world',
        step: 'step1',
        type: 'stdout',
        offset: 2,
      });
      handle.pushMessage({
        msg: 'boom',
        step: 'step1',
        type: 'stderr',
        offset: 1,
      });
      handle.pushMessage({
        msg: 'more',
        step: 'step2',
        type: 'stdout',
        offset: 1,
      });
    });

    await waitFor(() =>
      expect(result.current.textByStep.step2?.stdout).toBe('more')
    );

    expect(result.current.textByStep.step1.stdout).toBe('hello world');
    expect(result.current.textByStep.step1.stderr).toBe('boom');
    expect(result.current.stepOrder).toEqual(['step1', 'step2']);
  });

  it('accepts the initial message when its offset is 0', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.pushMessage({
        msg: 'first ',
        step: 's',
        type: 'stdout',
        offset: 0,
      });
      handle.pushMessage({
        msg: 'second',
        step: 's',
        type: 'stdout',
        offset: 1,
      });
    });

    await waitFor(() =>
      expect(result.current.textByStep.s?.stdout).toBe('first second')
    );
  });

  it('preserves empty-string log chunks and advances the offset', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.pushMessage({ msg: 'a', step: 's', type: 'stdout', offset: 0 });
      handle.pushMessage({ msg: '', step: 's', type: 'stdout', offset: 1 });
      handle.pushMessage({ msg: 'b', step: 's', type: 'stdout', offset: 2 });
    });

    await waitFor(() => expect(result.current.textByStep.s?.stdout).toBe('ab'));

    // Offset 1 was consumed — duplicate at offset 1 must be ignored
    act(() => {
      handle.pushMessage({ msg: 'dup', step: 's', type: 'stdout', offset: 1 });
    });
    await flushPromises();

    expect(result.current.textByStep.s.stdout).toBe('ab');
  });

  it('dedupes messages whose offset is not greater than the last seen', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.pushMessage({ msg: 'a', step: 's', type: 'stdout', offset: 5 });
      handle.pushMessage({ msg: 'b', step: 's', type: 'stdout', offset: 5 }); // dup
      handle.pushMessage({ msg: 'c', step: 's', type: 'stdout', offset: 4 }); // stale
      handle.pushMessage({ msg: 'd', step: 's', type: 'stdout', offset: 6 });
    });

    await waitFor(() => expect(result.current.textByStep.s?.stdout).toBe('ad'));
  });

  it('ignores malformed payloads', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.pushMessage({ step: 's', type: 'stdout', offset: 2 }); // missing msg
      handle.pushMessage({ msg: 'x', step: 's', type: 'stdout' }); // missing offset
      handle.pushMessage({ msg: 'x', step: '', type: 'stdout', offset: 1 }); // empty step
      handle.pushMessage({ msg: 'x', step: 's', type: '', offset: 1 }); // empty type
    });
    await flushPromises();

    expect(result.current.textByStep).toEqual({});
  });

  it('handles finish event by setting status and stopping the stream', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.pushNamed('finish', { status: 'success' });
    });

    await waitFor(() => expect(result.current.streamStatus).toBe('finished'));
    expect(result.current.finishStatus).toBe('success');
  });

  it('handles sep-error event with a 410 payload', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.pushNamed('sep-error', {
        code: 410,
        detail: { resource_type: 'job', job_id: 'J', message: 'gone' },
      });
    });

    await waitFor(() => expect(result.current.streamStatus).toBe('error'));
    expect(result.current.error?.code).toBe(410);
  });

  it('surfaces a terminal error when the server closes the stream without a finish event', async () => {
    const { result } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    const handle = mock.pending[0];
    act(() => {
      handle.close();
    });

    await waitFor(() => expect(result.current.streamStatus).toBe('error'));
    expect(result.current.error).toBeDefined();
  });

  it('aborts the fetch on unmount', async () => {
    const { unmount } = renderHook(() => useTaskLogs(1));
    await flushPromises();

    unmount();

    // fetchEventSource propagates inputSignal.abort() to its internal
    // curRequestController synchronously via the abort event listener.
    expect(mock.pending[0].signal?.aborted).toBe(true);
  });

  it('opens a fresh fetch when the task id changes', async () => {
    const { rerender } = renderHook(({ id }) => useTaskLogs(id), {
      initialProps: { id: 1 as number | string },
    });
    await flushPromises();

    expect(mock.fetchSpy).toHaveBeenCalledTimes(1);

    rerender({ id: 2 });
    await flushPromises();

    expect(mock.fetchSpy).toHaveBeenCalledTimes(2);
    const secondUrl = mock.fetchSpy.mock.calls[1][0];
    const urlStr =
      typeof secondUrl === 'string' ? secondUrl : (secondUrl as URL).href;
    expect(urlStr).toBe('/sep/stream-logs/2');
  });

  it('refreshes token on 401 and reconnects with the new token', async () => {
    const { refreshAccessToken } = await import('@sep/api');
    vi.mocked(refreshAccessToken).mockResolvedValue('new-token');

    // First fetch returns 401; onopen calls refreshAccessToken() → new token →
    // throws StreamRetriableAfterRefresh → onerror returns 0 (immediate retry).
    mock.queueResponse({ status: 401 });

    renderHook(() => useTaskLogs(1));

    // onerror returns 0 so the library retries immediately (setTimeout 0 ms).
    // waitFor polls until the second fetch appears.
    await waitFor(() =>
      expect(mock.fetchSpy.mock.calls.length).toBeGreaterThanOrEqual(2)
    );

    // pending[0] is the successful retry (non-200 responses skip pending); headers are lowercased
    expect(mock.pending[0].requestHeaders.authorization).toBe(
      'Bearer new-token'
    );
  });

  it('surfaces an error and stops retrying when token refresh returns null', async () => {
    const { refreshAccessToken } = await import('@sep/api');
    vi.mocked(refreshAccessToken).mockResolvedValue(null);

    mock.queueResponse({ status: 401 });

    const { result } = renderHook(() => useTaskLogs(1));

    await waitFor(() => expect(result.current.streamStatus).toBe('error'));
    // Exactly one fetch: refresh failed → StreamFatalError → no retry.
    expect(mock.fetchSpy).toHaveBeenCalledTimes(1);
  });

  it('surfaces an error and stops retrying on a non-401 open failure', async () => {
    mock.queueResponse({ status: 503 });

    const { result } = renderHook(() => useTaskLogs(1));

    await waitFor(() => expect(result.current.streamStatus).toBe('error'));
    expect(mock.fetchSpy).toHaveBeenCalledTimes(1);
  });

  it('does not write state from a stale aborted stream after id changes', async () => {
    const { rerender, result } = renderHook(({ id }) => useTaskLogs(id), {
      initialProps: { id: 1 as number | string },
    });
    await flushPromises();

    const firstHandle = mock.pending[0];

    // Change id before the first stream emits anything
    rerender({ id: 2 });
    await flushPromises();

    // Push a message on the old (aborted) stream
    act(() => {
      firstHandle.pushMessage({
        msg: 'stale',
        step: 's',
        type: 'stdout',
        offset: 1,
      });
    });
    await flushPromises();

    // New stream has no data; stale message must not appear
    expect(result.current.textByStep).toEqual({});
  });

  it('does not write state from a stale aborted stream after tail changes', async () => {
    const { rerender, result } = renderHook(
      ({ tail }) => useTaskLogs(1, tail),
      {
        initialProps: { tail: 1000 as number | undefined },
      }
    );
    await flushPromises();

    const firstHandle = mock.pending[0];

    rerender({ tail: 100 });
    await flushPromises();

    act(() => {
      firstHandle.pushMessage({
        msg: 'stale',
        step: 's',
        type: 'stdout',
        offset: 1,
      });
    });
    await flushPromises();

    expect(result.current.textByStep).toEqual({});
  });
});
