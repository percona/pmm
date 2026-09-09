/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import type { ReactNode } from 'react';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
}));

import { ApiError, apiClient } from '@sep/api';
import { useServices } from './useServices';

const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function wrapper() {
  // `retry: false` here is a no-op for this hook: useServices sets
  // `retry: sepRetry` at the query level, which overrides the client default
  // in React Query v5. We keep it for documentation. `retryDelay: 0` removes
  // the exponential backoff so tests that exercise the retry path finish in
  // milliseconds.
  const client = new QueryClient({
    defaultOptions: {
      queries: { gcTime: 0, staleTime: 0, retry: false, retryDelay: 0 },
    },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

function page(
  items: Array<{ id: number; name: string; type: string }>,
  total: number,
  offset = 0
) {
  return { data: { items, total, offset, limit: 200 } };
}

beforeEach(() => {
  mocked.get.mockReset();
});

describe('useServices', () => {
  it('omits service_type from request params when none is supplied', async () => {
    mocked.get.mockResolvedValueOnce(page([], 0));

    const { result } = renderHook(() => useServices(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mocked.get).toHaveBeenCalledTimes(1);
    const [, config] = mocked.get.mock.calls[0];
    expect(config.params).not.toHaveProperty('service_type');
    expect(config.params).toMatchObject({ offset: 0, limit: 200 });
  });

  it('paginates until offset >= total for a single service type', async () => {
    mocked.get
      .mockResolvedValueOnce(
        page(
          Array.from({ length: 200 }, (_, i) => ({
            id: i + 1,
            name: `s${i + 1}`,
            type: 'mysql',
          })),
          250
        )
      )
      .mockResolvedValueOnce(
        page(
          Array.from({ length: 50 }, (_, i) => ({
            id: 201 + i,
            name: `s${201 + i}`,
            type: 'mysql',
          })),
          250
        )
      );

    const { result } = renderHook(
      () => useServices({ serviceTypes: ['mysql'] as never[] }),
      {
        wrapper: wrapper(),
      }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mocked.get).toHaveBeenCalledTimes(2);
    expect(result.current.data).toHaveLength(250);
  });

  it('breaks when items.length === 0 even before offset reaches total', async () => {
    // Defensive break — server might claim total=999 but return zero items on
    // the second page (e.g. a row got deleted between requests).
    mocked.get
      .mockResolvedValueOnce(page([{ id: 1, name: 's1', type: 'mysql' }], 999))
      .mockResolvedValueOnce(page([], 999));

    const { result } = renderHook(
      () => useServices({ serviceTypes: ['mysql'] as never[] }),
      {
        wrapper: wrapper(),
      }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mocked.get).toHaveBeenCalledTimes(2);
    expect(result.current.data).toEqual([{ id: 1, name: 's1', type: 'mysql' }]);
  });

  it('dedupes by id across parallel service-type fan-out', async () => {
    mocked.get.mockImplementation(
      (_url: string, config?: { params: Record<string, unknown> }) => {
        if (config?.params.service_type === 'mysql') {
          return Promise.resolve(
            page(
              [
                { id: 1, name: 'svc1', type: 'mysql' },
                { id: 2, name: 'svc2', type: 'mysql' },
              ],
              2
            )
          );
        }
        if (config?.params.service_type === 'postgresql') {
          return Promise.resolve(
            page(
              [
                { id: 2, name: 'svc2', type: 'mysql' },
                { id: 3, name: 'svc3', type: 'postgresql' },
              ],
              2
            )
          );
        }
        return Promise.resolve(page([], 0));
      }
    );

    const { result } = renderHook(
      () => useServices({ serviceTypes: ['mysql', 'postgresql'] as never[] }),
      { wrapper: wrapper() }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(result.current.data?.map((s) => s.id).sort()).toEqual([1, 2, 3]);
  });

  it('surfaces error when any service-type fan-out fetch rejects', async () => {
    // Promise.all rejects on first failure — current trade-off. Document it
    // here so any future partial-fallback change is intentional.
    mocked.get.mockImplementation(
      (_url: string, config?: { params: Record<string, unknown> }) => {
        if (config?.params.service_type === 'mysql') {
          return Promise.resolve(
            page([{ id: 1, name: 'svc1', type: 'mysql' }], 1)
          );
        }
        return Promise.reject(
          new ApiError({
            kind: 'http',
            status: 500,
            message: 'upstream blew up',
          })
        );
      }
    );

    const { result } = renderHook(
      () => useServices({ serviceTypes: ['mysql', 'postgresql'] as never[] }),
      { wrapper: wrapper() }
    );
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect((result.current.error as InstanceType<typeof ApiError>).status).toBe(
      500
    );
  });

  it('produces a stable query key whether serviceTypes is undefined or empty', async () => {
    mocked.get.mockResolvedValue(page([], 0));

    const { result: ra } = renderHook(() => useServices(), {
      wrapper: wrapper(),
    });
    const { result: rb } = renderHook(
      () => useServices({ serviceTypes: [] as never[] }),
      {
        wrapper: wrapper(),
      }
    );

    await waitFor(() => {
      expect(ra.current.isSuccess).toBe(true);
      expect(rb.current.isSuccess).toBe(true);
    });

    // Both call the same endpoint with no service_type filter; combined with
    // STABLE_EMPTY this means React Query shares the cache entry.
    const allCalls = mocked.get.mock.calls;
    for (const [, config] of allCalls) {
      expect(config.params).not.toHaveProperty('service_type');
    }
  });

  it('aborts pagination after MAX_PAGES iterations to prevent runaway loops', async () => {
    // Backend bug: `total` is reported as larger than what `items` can ever
    // catch up to. Without the cap this would run forever and hang the tab.
    mocked.get.mockImplementation(() =>
      Promise.resolve(
        page([{ id: 1, name: 's', type: 'mysql' }], Number.MAX_SAFE_INTEGER)
      )
    );

    const { result } = renderHook(
      () => useServices({ serviceTypes: ['mysql'] as never[] }),
      {
        wrapper: wrapper(),
      }
    );
    await waitFor(() => expect(result.current.isError).toBe(true));
    // Each attempt fetches exactly MAX_PAGES (50) pages before throwing the
    // pagination-exceeded Error. sepRetry retries plain Errors up to 2 times,
    // so the total call count is a multiple of 50. We assert the per-attempt
    // ceiling (the guard's actual contract) rather than the retry-aware total.
    expect(mocked.get.mock.calls.length).toBeGreaterThanOrEqual(50);
    expect(mocked.get.mock.calls.length % 50).toBe(0);
    expect((result.current.error as Error).message).toMatch(
      /pagination exceeded/i
    );
  });

  it('does not retry a deterministic 502 (sepRetry short-circuit)', async () => {
    // Without `retry: sepRetry` React Query's default policy would fire 4
    // attempts before surfacing the error on the required Database Host
    // selector. The SEP gateway returns 502 only when the upstream Tasks
    // API is unreachable — retrying just queues failed user-visible loads.
    mocked.get.mockRejectedValue(
      new ApiError({ kind: 'http', status: 502, message: '502 Bad Gateway' })
    );

    const { result } = renderHook(
      () => useServices({ serviceTypes: ['mysql'] as never[] }),
      {
        wrapper: wrapper(),
      }
    );
    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(mocked.get).toHaveBeenCalledTimes(1);
    expect((result.current.error as InstanceType<typeof ApiError>).status).toBe(
      502
    );
  });

  it('respects enabled=false', async () => {
    const { result } = renderHook(() => useServices({ enabled: false }), {
      wrapper: wrapper(),
    });
    // Hook is disabled — no request should fire.
    await new Promise((r) => setTimeout(r, 20));
    expect(mocked.get).not.toHaveBeenCalled();
    expect(result.current.isFetching).toBe(false);
  });
});
