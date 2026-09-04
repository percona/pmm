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
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { IncidentWorkspacePage } from '../src/IncidentWorkspacePage';

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

const incidentId = '11111111-1111-4111-8111-111111111111';
const openIncident = {
  id: incidentId,
  name: 'DB slowness',
  case_ref: 'CS-42',
  created_by: 'engineer',
  created_at: '2026-07-22T10:00:00Z',
  updated_at: null,
  closed_at: null,
};

function renderWorkspace() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/atw/${incidentId}`]}>
        <Routes>
          <Route path="/atw/:incidentId" element={<IncidentWorkspacePage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('IncidentWorkspacePage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedApi.get.mockImplementation(async (url: string) => {
      if (url.includes('/executions/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/send-jobs/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/config/')) {
        return { data: { send_disabled_reasons: [] } };
      }
      if (url === '/apps/atw/') {
        return { data: [] };
      }
      return { data: openIncident };
    });
  });

  it('shows the close action for an open incident', async () => {
    renderWorkspace();

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Close incident/i })
      ).toBeTruthy();
    });
  });

  it('shows the reopen action and closed banner for a closed incident', async () => {
    mockedApi.get.mockImplementation(async (url: string) => {
      if (url.includes('/executions/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/send-jobs/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/config/')) {
        return { data: { send_disabled_reasons: [] } };
      }
      if (url === '/apps/atw/') {
        return { data: [] };
      }
      return {
        data: { ...openIncident, closed_at: '2026-07-30T12:00:00Z' },
      };
    });

    renderWorkspace();

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Reopen incident/i })
      ).toBeTruthy();
      expect(screen.getByText('Closed')).toBeTruthy();
      expect(screen.getByText(/This incident is closed/i)).toBeTruthy();
    });
  });

  it('closes an incident from the workspace header', async () => {
    mockedApi.post.mockResolvedValue({
      data: { ...openIncident, closed_at: '2026-07-30T12:00:00Z' },
    });
    const user = userEvent.setup();

    renderWorkspace();

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /Close incident/i })
      ).toBeTruthy()
    );
    await user.click(screen.getByRole('button', { name: /Close incident/i }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incidentId}/close/`
      );
    });
  });

  it('shows an error when closing the incident fails', async () => {
    mockedApi.post.mockRejectedValue(new Error('Incident is already closed.'));
    const user = userEvent.setup();

    renderWorkspace();

    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /Close incident/i })
      ).toBeTruthy()
    );
    await user.click(screen.getByRole('button', { name: /Close incident/i }));

    await waitFor(() => {
      expect(screen.getByText('Incident is already closed.')).toBeTruthy();
    });
  });
});

describe('IncidentWorkspacePage — write access', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedApi.get.mockImplementation(async (url: string) => {
      if (url.includes('/executions/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/send-jobs/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/config/')) {
        return { data: { send_disabled_reasons: [] } };
      }
      if (url === '/apps/atw/') {
        return { data: [] };
      }
      return { data: openIncident };
    });
  });

  it('shows the close action for a session that may mutate', async () => {
    renderWorkspace();

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Close incident/i })
      ).toBeTruthy();
    });
  });

  it('shows no close action for a non-admin, keeping the incident readable', async () => {
    mockCanMutate = false;
    renderWorkspace();

    await waitFor(() => expect(screen.getByText('DB slowness')).toBeTruthy());
    expect(
      screen.queryByRole('button', { name: /Close incident/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /Reopen incident/i })
    ).not.toBeInTheDocument();
  });

  it('shows the Collect pane for a session that may mutate', async () => {
    renderWorkspace();

    await waitFor(() => expect(screen.getByText('Collect')).toBeTruthy());
    expect(screen.getByText('Results')).toBeTruthy();
    expect(
      screen.queryByTestId('atw-collect-read-only')
    ).not.toBeInTheDocument();
  });

  it('withholds the Collect pane from a non-admin, leaving Results', async () => {
    mockCanMutate = false;
    renderWorkspace();

    await waitFor(() => expect(screen.getByText('Results')).toBeTruthy());
    expect(screen.queryByText('Collect')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('combobox', { name: 'Snippets' })
    ).not.toBeInTheDocument();
    expect(screen.getByTestId('atw-collect-read-only')).toBeTruthy();
    expect(
      screen.getByText(/permission to collect diagnostics for this incident/i)
    ).toBeTruthy();
  });

  it('fetches no snippet categories for a non-admin', async () => {
    mockCanMutate = false;
    renderWorkspace();

    await waitFor(() => expect(screen.getByText('Results')).toBeTruthy());
    expect(mockedApi.get).not.toHaveBeenCalledWith('/apps/atw/');
  });
});
