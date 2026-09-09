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

import { apiClient } from '@sep/api';
import { useRemoteChoices } from './useRemoteChoices';

const mocked = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function newClient() {
  return new QueryClient({
    defaultOptions: { queries: { gcTime: 0, staleTime: 0, retryDelay: 0 } },
  });
}

function wrapperFor(client: QueryClient) {
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
}

beforeEach(() => {
  mocked.get.mockReset();
  mocked.get.mockResolvedValue({ data: [{ value: 'a', label: 'A' }] });
});

describe('useRemoteChoices', () => {
  it('fetches the endpoint URL when there is no cascade dependency', async () => {
    const { result } = renderHook(
      () => useRemoteChoices({ endpointUrl: '/apps/restore/backups' }),
      {
        wrapper: wrapperFor(newClient()),
      }
    );
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mocked.get).toHaveBeenCalledWith('/apps/restore/backups');
    expect(result.current.data).toEqual([{ value: 'a', label: 'A' }]);
  });

  it('does not fetch until a cascade parent value is present', async () => {
    const { result, rerender } = renderHook(
      ({ v }: { v: string | null }) =>
        useRemoteChoices({
          endpointUrl: '/apps/restore/backups',
          dependsOnName: 'cluster',
          dependsOnValue: v,
        }),
      {
        wrapper: wrapperFor(newClient()),
        initialProps: { v: null as string | null },
      }
    );

    expect(mocked.get).not.toHaveBeenCalled();
    expect(result.current.fetchStatus).toBe('idle');

    rerender({ v: 'cluster-a' });
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    expect(mocked.get).toHaveBeenCalledWith(
      '/apps/restore/backups?cluster=cluster-a'
    );
  });

  it('encodes special characters in the cascade value', async () => {
    renderHook(
      () =>
        useRemoteChoices({
          endpointUrl: '/apps/restore/backups',
          dependsOnName: 'cluster',
          dependsOnValue: 'a b/c&d',
        }),
      { wrapper: wrapperFor(newClient()) }
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    expect(mocked.get).toHaveBeenCalledWith(
      '/apps/restore/backups?cluster=a+b%2Fc%26d'
    );
  });

  it('preserves an existing query string on the endpoint URL', async () => {
    renderHook(
      () =>
        useRemoteChoices({
          endpointUrl: '/apps/restore/backups?scope=all',
          dependsOnName: 'cluster',
          dependsOnValue: 'c1',
        }),
      { wrapper: wrapperFor(newClient()) }
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalled());
    expect(mocked.get).toHaveBeenCalledWith(
      '/apps/restore/backups?scope=all&cluster=c1'
    );
  });

  it('uses distinct cache keys for the same endpoint+value under different dependsOn names', async () => {
    const client = newClient();
    const wrapper = wrapperFor(client);
    renderHook(
      () =>
        useRemoteChoices({
          endpointUrl: '/e',
          dependsOnName: 'a',
          dependsOnValue: 'x',
        }),
      {
        wrapper,
      }
    );
    renderHook(
      () =>
        useRemoteChoices({
          endpointUrl: '/e',
          dependsOnName: 'b',
          dependsOnValue: 'x',
        }),
      {
        wrapper,
      }
    );
    await waitFor(() => expect(mocked.get).toHaveBeenCalledTimes(2));
    const urls = mocked.get.mock.calls.map((call) => call[0]);
    expect(urls).toEqual(expect.arrayContaining(['/e?a=x', '/e?b=x']));
  });

  it('surfaces fetch errors via the error slot', async () => {
    mocked.get.mockRejectedValue(new Error('boom'));
    const { result } = renderHook(
      () => useRemoteChoices({ endpointUrl: '/apps/restore/backups' }),
      {
        wrapper: wrapperFor(newClient()),
      }
    );
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error).toBeInstanceOf(Error);
  });
});
