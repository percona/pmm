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
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import type { ReactNode } from 'react';
import { IncidentListPage } from '../src/IncidentListPage';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
}));

import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
  patch: ReturnType<typeof vi.fn>;
  delete: ReturnType<typeof vi.fn>;
};

const incident = {
  id: '11111111-1111-4111-8111-111111111111',
  name: 'DB slowness',
  case_ref: 'CS-42',
  created_by: 'engineer',
  created_at: '2026-07-22T10:00:00Z',
  updated_at: null,
};

function paginated<T>(items: T[]) {
  return { data: { items, total: items.length, offset: 0, limit: 50 } };
}

function renderPage(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>
  );
}

describe('IncidentListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the incident list', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByText('DB slowness')).toBeTruthy();
    });
    expect(screen.getByText(/Case CS-42/)).toBeTruthy();
  });

  it('shows an empty state when there are no incidents', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByText(/No incidents yet/i)).toBeTruthy();
    });
  });

  it('creates an incident from the dialog, sending the trimmed name', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() =>
      expect(screen.getByText(/No incidents yet/i)).toBeTruthy()
    );

    await user.click(screen.getByRole('button', { name: /New incident/i }));
    await user.type(
      screen.getByLabelText(/Name \(optional\)/i),
      '  Prod outage  '
    );
    await user.click(screen.getByRole('button', { name: /^Create$/ }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith('/apps/atw/incidents/', {
        name: 'Prod outage',
      });
    });
  });

  it('omits the name when the create field is left blank', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() =>
      expect(screen.getByText(/No incidents yet/i)).toBeTruthy()
    );

    await user.click(screen.getByRole('button', { name: /New incident/i }));
    await user.click(screen.getByRole('button', { name: /^Create$/ }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith('/apps/atw/incidents/', {});
    });
  });

  it('renders pagination controls once the list exceeds one page', async () => {
    mockedApi.get.mockResolvedValue({
      data: { items: [incident], total: 40, offset: 0, limit: 20 },
    });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Go to next page/i })
      ).toBeTruthy();
    });
    expect(screen.getByText(/of 40/)).toBeTruthy();
  });

  it('omits pagination controls when a single page holds everything', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByText('DB slowness')).toBeTruthy();
    });
    expect(
      screen.queryByRole('button', { name: /Go to next page/i })
    ).toBeNull();
  });
});
