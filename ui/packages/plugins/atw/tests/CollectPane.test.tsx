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

import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { CollectPane } from '../src/CollectPane';

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

import { apiClient } from '@pmm-extensions/api';
const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

function renderPane(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/atw/incident']}>
        <Routes>
          <Route path="/atw/incident" element={ui} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('CollectPane', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedApi.get.mockResolvedValue({ data: { shared: [], per_snippet: [] } });
  });

  it('gives its instructions as the picker helper text, not as an info alert', async () => {
    renderPane(
      <CollectPane incidentId="11111111-1111-4111-8111-111111111111" />
    );

    const picker = await screen.findByRole('combobox', {
      name: 'Search scripts',
    });
    expect(picker).toHaveAccessibleDescription(
      /Search by name or description, or filter by category/
    );
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('replaces the picker and its instructions with one line when the incident is closed', async () => {
    renderPane(
      <CollectPane incidentId="11111111-1111-4111-8111-111111111111" isClosed />
    );

    await waitFor(() => {
      expect(screen.getByText(/This incident is closed/i)).toBeTruthy();
    });
    expect(screen.queryByRole('button', { name: /Execute batch/i })).toBeNull();
    expect(screen.queryByText('Filter by category')).toBeNull();
    expect(
      screen.queryByRole('combobox', { name: 'Search scripts' })
    ).toBeNull();
    expect(screen.queryByText(/Search by name or description/)).toBeNull();
  });
});
