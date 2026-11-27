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
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import type * as Api from '../src/api';
import { useOmInventoryRuns, useRefreshInventory } from '../src/inventoryHooks';

const { request } = vi.hoisted(() => ({ request: vi.fn() }));

vi.mock('../src/api', async (importOriginal) => ({
  ...(await importOriginal<typeof Api>()),
  request,
}));

describe('useRefreshInventory', () => {
  // The Scan buttons and the 409 notice read that history, and a request settled
  // before it would let them act on a history without the scan just asked for.
  it('settles a scan request only once the scan history has been read again', async () => {
    const reads: ((body: { runs: [] }) => void)[] = [];
    request.mockImplementation((path: string) =>
      path.endsWith(':trigger')
        ? Promise.resolve({ run_id: 'run-1' })
        : new Promise((resolve) => reads.push(resolve))
    );
    const client = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    const { result } = renderHook(
      () => ({ runs: useOmInventoryRuns(), refresh: useRefreshInventory() }),
      {
        wrapper: ({ children }: { children: ReactNode }) => (
          <QueryClientProvider client={client}>{children}</QueryClientProvider>
        ),
      }
    );
    await waitFor(() => expect(reads).toHaveLength(1));
    act(() => reads[0]({ runs: [] }));
    await waitFor(() => expect(result.current.runs.isSuccess).toBe(true));

    act(() => result.current.refresh.refreshHosts(['n1']));
    await waitFor(() => expect(reads).toHaveLength(2));
    expect(result.current.refresh.isPending).toBe(true);

    act(() => reads[1]({ runs: [] }));
    await waitFor(() => expect(result.current.refresh.isSuccess).toBe(true));
  });
});
