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

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
}));

import { ApiError, apiClient } from '@sep/api';
import { useHosts } from './useHosts';

const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function wrapper() {
  // Use a QueryClient with no `retry` override so the hook-level `retry`
  // predicate is the only thing governing retries. `retryDelay: 0` keeps
  // the retry tests fast.
  const client = new QueryClient({
    defaultOptions: { queries: { gcTime: 0, staleTime: 0, retryDelay: 0 } },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  mocked.get.mockReset();
});

describe('useHosts retry predicate', () => {
  it('does not retry on a 502 ApiError (upstream Tasks-API failure)', async () => {
    mocked.get.mockRejectedValue(
      new ApiError({ kind: 'http', status: 502, message: 'tasks unreachable' })
    );
    const { result } = renderHook(() => useHosts(), { wrapper: wrapper() });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    expect(result.current.error).toBeInstanceOf(ApiError);
    expect((result.current.error as InstanceType<typeof ApiError>).status).toBe(
      502
    );
    expect(mocked.get).toHaveBeenCalledTimes(1);
  });

  it.each([
    ['401', 401],
    ['403', 403],
    ['404', 404],
  ])('does not retry on a %s ApiError', async (_label, status) => {
    mocked.get.mockRejectedValue(
      new ApiError({ kind: 'http', status, message: `HTTP ${status}` })
    );
    const { result } = renderHook(() => useHosts(), { wrapper: wrapper() });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    expect(mocked.get).toHaveBeenCalledTimes(1);
  });

  it('retries up to twice on a non-ApiError rejection (network blip)', async () => {
    mocked.get.mockRejectedValue(new Error('network blip'));
    const { result } = renderHook(() => useHosts(), { wrapper: wrapper() });
    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });
    // Predicate returns `count < 2` for non-ApiError, so React Query calls
    // queryFn 3 times total (initial + 2 retries) before giving up.
    expect(mocked.get).toHaveBeenCalledTimes(3);
  });
});
