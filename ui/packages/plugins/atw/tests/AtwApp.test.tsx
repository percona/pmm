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
import { AtwApp } from '../src/AtwApp';

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn() },
  useAuth: () => ({ isAdmin: true, canMutate: true }),
}));

import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

const incidentId = '11111111-1111-4111-8111-111111111111';
const SETTINGS_PATH = '/settings/servicenow-connection';

const incident = {
  id: incidentId,
  name: 'DB slowness',
  case_ref: 'CS-42',
  created_by: 'engineer',
  created_at: '2026-07-22T10:00:00Z',
  updated_at: null,
  closed_at: null,
};

const execution = {
  id: 'exec-1',
  snippet_filename: 'diag/slow-query.sh',
  task_history_id: 7,
  created_at: '2026-07-22T10:00:00Z',
  task_status: 'success',
  started_at: null,
  finished_at: null,
  has_logs: false,
};

/** The app mounted the way the shell mounts it: at a splat, inside a router. */
function renderApp(deliverySettingsPath?: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/atw/${incidentId}`]}>
        <Routes>
          <Route
            path="/atw/*"
            element={<AtwApp deliverySettingsPath={deliverySettingsPath} />}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

/**
 * The prop is the whole point of the seam, and every other test supplies the
 * context by hand — so this is the one place that proves the route the shell
 * passes actually reaches the control that offers it.
 */
describe('AtwApp delivery settings seam', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedApi.get.mockImplementation(async (url: string) => {
      if (url.includes('/executions/')) {
        return { data: { items: [execution], total: 1, offset: 0, limit: 20 } };
      }
      if (url.includes('/send-jobs/')) {
        return { data: { items: [], total: 0, offset: 0, limit: 20 } };
      }
      if (url.includes('/config/')) {
        return {
          data: {
            send_disabled_reasons: ['Diagnostics delivery is not configured'],
          },
        };
      }
      if (url === '/apps/atw/') {
        return { data: [] };
      }
      return { data: incident };
    });
  });

  it('hands the host route down to the disabled send control', async () => {
    renderApp(SETTINGS_PATH);

    await waitFor(() => {
      expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
    });
    expect(
      screen.getByTestId('atw-send-unavailable-settings').getAttribute('href')
    ).toBe(SETTINGS_PATH);
  });

  it('still explains the disabled send when the host names no route', async () => {
    renderApp(undefined);

    await waitFor(() => {
      expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
    });
    expect(
      screen.queryByTestId('atw-send-unavailable-settings')
    ).not.toBeInTheDocument();
  });
});
