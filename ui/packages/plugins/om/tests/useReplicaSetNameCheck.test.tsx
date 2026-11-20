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
import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useReplicaSetNameCheck } from '../src/inventoryHooks';

const IN_USE =
  'Replica set name rs0 is already used in this environment and cluster by the MongoDB service(s) orders-1. Choose another name.';

const fetchMock = vi.fn(async (url: string) => {
  const name = new URL(url, 'https://pmm').searchParams.get('replica_set_name');
  return new Response(
    JSON.stringify(name === 'rs0' ? { in_use: IN_USE } : {}),
    {
      status: 200,
    }
  );
});

const wrapper = ({ children }: { children: ReactNode }) => (
  <QueryClientProvider client={new QueryClient()}>
    {children}
  </QueryClientProvider>
);

describe('useReplicaSetNameCheck', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    fetchMock.mockClear();
  });

  it('asks about the name, environment and cluster once typing pauses', async () => {
    vi.stubGlobal('fetch', fetchMock);

    const { result } = renderHook(
      () => useReplicaSetNameCheck('rs0', 'prod', '', true),
      { wrapper }
    );

    await waitFor(() => expect(result.current.data).toBe(IN_USE));
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const url = new URL(fetchMock.mock.calls[0][0], 'https://pmm');
    expect(url.pathname).toMatch(/\/inventory\/replicaSetNames:check$/);
    expect(Object.fromEntries(url.searchParams)).toEqual({
      replica_set_name: 'rs0',
      environment: 'prod',
      cluster: '',
    });
  });

  it('never answers for a new name with what it learned about the previous one', async () => {
    vi.stubGlobal('fetch', fetchMock);
    const { result, rerender } = renderHook(
      ({ name }) => useReplicaSetNameCheck(name, '', '', true),
      { wrapper, initialProps: { name: 'rs0' } }
    );
    await waitFor(() => expect(result.current.data).toBe(IN_USE));

    rerender({ name: 'rs1' });

    expect(result.current.data).toBeUndefined();
    await waitFor(() => expect(result.current.data).toBeNull());
  });

  it('waits for a fresh answer on going back to a name it already checked', async () => {
    vi.stubGlobal('fetch', fetchMock);
    const { result, rerender } = renderHook(
      ({ name }) => useReplicaSetNameCheck(name, '', '', true),
      { wrapper, initialProps: { name: 'rs1' } }
    );
    await waitFor(() => expect(result.current.data).toBeNull());
    rerender({ name: 'rs0' });
    await waitFor(() => expect(result.current.data).toBe(IN_USE));

    rerender({ name: 'rs1' });

    expect(result.current.data).toBeUndefined();
    await waitFor(() => expect(result.current.data).toBeNull());
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it('asks nothing while disabled', async () => {
    vi.stubGlobal('fetch', fetchMock);

    renderHook(() => useReplicaSetNameCheck('rs 0', '', '', false), {
      wrapper,
    });

    await new Promise((resolve) => setTimeout(resolve, 400));
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
