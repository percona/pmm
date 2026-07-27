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
import type { ReactNode } from 'react';
import { ResultsPane } from '../src/ResultsPane';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn() },
}));

// The rows render collapsed, so the log viewer never mounts (unmountOnExit) and
// the files dialog stays closed — neither fires a query in these tests.

import { apiClient } from '@sep/api';
const mockedApi = apiClient as unknown as { get: ReturnType<typeof vi.fn> };

function paginated<T>(items: T[]) {
  return { data: { items, total: items.length, offset: 0, limit: 50 } };
}

function renderPane(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('ResultsPane', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders each execution with its snippet name and status', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([
        {
          id: 'exec-1',
          snippet_filename: 'diag/slow-query.sh',
          task_history_id: 7,
          created_at: '2026-07-22T10:00:00Z',
          task_status: 'success',
          started_at: null,
          finished_at: null,
          has_logs: true,
        },
        {
          id: 'exec-2',
          snippet_filename: 'diag/dmesg.sh',
          task_history_id: 8,
          created_at: '2026-07-22T10:01:00Z',
          task_status: 'failed',
          started_at: null,
          finished_at: null,
          has_logs: true,
        },
      ]),
    );

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText('diag/slow-query.sh')).toBeTruthy();
    });
    expect(screen.getByText('diag/dmesg.sh')).toBeTruthy();
    expect(screen.getByText('Done')).toBeTruthy();
    expect(screen.getByText('Failed')).toBeTruthy();
  });

  it('shows an empty state when the incident has no executions', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText(/No executions yet/i)).toBeTruthy();
    });
  });

  it('renders an Unknown chip when the task status could not be hydrated', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([
        {
          id: 'exec-3',
          snippet_filename: 'diag/x.sh',
          task_history_id: 9,
          created_at: '2026-07-22T10:02:00Z',
          task_status: null,
          started_at: null,
          finished_at: null,
          has_logs: null,
        },
      ]),
    );

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText('Unknown')).toBeTruthy();
    });
  });
});
