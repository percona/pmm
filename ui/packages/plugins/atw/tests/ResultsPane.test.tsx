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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
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
      ])
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
      ])
    );

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText('Unknown')).toBeTruthy();
    });
  });
});

// ── Diagnostics send ─────────────────────────────────────────────────────

const FINISHED_EXECUTION = {
  id: 'exec-1',
  snippet_filename: 'diag/slow-query.sh',
  task_history_id: 7,
  created_at: '2026-07-22T10:00:00Z',
  task_status: 'success',
  started_at: null,
  finished_at: null,
  has_logs: true,
};

const RUNNING_EXECUTION = {
  id: 'exec-2',
  snippet_filename: 'diag/dmesg.sh',
  task_history_id: 8,
  created_at: '2026-07-22T10:01:00Z',
  task_status: 'running',
  started_at: null,
  finished_at: null,
  has_logs: true,
};

const STALE_EXECUTION = {
  id: 'exec-3',
  snippet_filename: 'diag/vmstat.sh',
  task_history_id: 9,
  created_at: '2026-07-22T10:02:00Z',
  task_status: 'stale',
  started_at: null,
  finished_at: null,
  has_logs: true,
};

/** An execution recorded on a past attempt but absent from the current page. */
const OFF_PAGE_EXECUTION = {
  id: 'exec-99',
  task_history_id: 99,
  snippet_filename: 'diag/off-page.sh',
};

function failedJob(detail: unknown) {
  return {
    id: 'job-1',
    incident_id: 'inc-1',
    case_ref: 'CS0042',
    requested_by: 'alice',
    status: 'failed',
    started_at: null,
    finished_at: '2026-07-24T10:01:00Z',
    created_at: '2026-07-24T10:00:00Z',
    detail,
  };
}

/**
 * Route each GET by URL, so the executions list, config probe and send-job
 * history can answer with their own shapes rather than one shared envelope.
 */
function routeGet(routes: {
  executions?: unknown;
  config?: unknown;
  sendJobs?: unknown;
  incident?: unknown;
}) {
  mockedApi.get.mockImplementation((url: string) => {
    if (url.includes('/send-jobs/')) {
      return Promise.resolve({
        data: routes.sendJobs ?? { items: [], total: 0, offset: 0, limit: 50 },
      });
    }
    if (url.includes('/executions/')) {
      return Promise.resolve({
        data: routes.executions ?? {
          items: [],
          total: 0,
          offset: 0,
          limit: 20,
        },
      });
    }
    if (url.includes('/config/')) {
      return Promise.resolve({
        data: routes.config ?? { send_disabled_reasons: [] },
      });
    }
    return Promise.resolve({
      data: routes.incident ?? { id: 'inc-1', case_ref: 'CS0001' },
    });
  });
}

describe('ResultsPane diagnostics send', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('offers a checkbox only for finished executions', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION, RUNNING_EXECUTION],
        total: 2,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByLabelText('Select diag/slow-query.sh')).toBeTruthy();
    });
    expect(
      (screen.getByLabelText('Select diag/slow-query.sh') as HTMLInputElement)
        .disabled
    ).toBe(false);
    expect(
      (screen.getByLabelText('Select diag/dmesg.sh') as HTMLInputElement)
        .disabled
    ).toBe(true);
  });

  it('enables the send action once an execution is selected', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByLabelText('Select diag/slow-query.sh')).toBeTruthy();
    });
    const sendButton = () =>
      screen.getByRole('button', {
        name: /Send to support case/i,
      }) as HTMLButtonElement;
    expect(sendButton().disabled).toBe(true);

    fireEvent.click(screen.getByLabelText('Select diag/slow-query.sh'));

    await waitFor(() => {
      expect(sendButton().disabled).toBe(false);
    });
    expect(screen.getByText('1 selected')).toBeTruthy();
  });

  it('keeps the send action disabled while delivery is unconfigured', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      config: {
        send_disabled_reasons: ['Diagnostics delivery is not configured'],
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByLabelText('Select diag/slow-query.sh')).toBeTruthy();
    });
    fireEvent.click(screen.getByLabelText('Select diag/slow-query.sh'));

    await waitFor(() => {
      expect(
        (
          screen.getByRole('button', {
            name: /Send to support case/i,
          }) as HTMLButtonElement
        ).disabled
      ).toBe(true);
    });
  });

  it('lists past attempts and offers Re-send on a failed one', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [
          {
            id: 'job-1',
            incident_id: 'inc-1',
            case_ref: 'CS0042',
            requested_by: 'alice',
            status: 'failed',
            started_at: null,
            finished_at: '2026-07-24T10:01:00Z',
            created_at: '2026-07-24T10:00:00Z',
            detail: {
              error: 'upstream exploded',
              executions: [FINISHED_EXECUTION],
            },
          },
        ],
        total: 1,
        offset: 0,
        limit: 50,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText('Send history')).toBeTruthy();
    });
    expect(screen.getByText(/CS0042/)).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Re-send' })).toBeTruthy();
  });

  it('keeps Re-send disabled while delivery is unconfigured', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [
          failedJob({
            error: 'upstream exploded',
            executions: [FINISHED_EXECUTION],
          }),
        ],
        total: 1,
        offset: 0,
        limit: 50,
      },
      config: {
        send_disabled_reasons: ['Diagnostics delivery is not configured'],
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Re-send' })).toBeTruthy();
    });
    expect(
      (screen.getByRole('button', { name: 'Re-send' }) as HTMLButtonElement)
        .disabled
    ).toBe(true);
  });

  it('renders pagination controls once the list exceeds one page', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 40,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Go to next page/i })
      ).toBeTruthy();
    });
    expect(screen.getByText(/of 40/)).toBeTruthy();
  });

  it('omits pagination controls when a single page holds everything', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText('diag/slow-query.sh')).toBeTruthy();
    });
    expect(
      screen.queryByRole('button', { name: /Go to next page/i })
    ).toBeNull();
  });

  it('offers a checkbox for a stale execution, which the backend counts finished', async () => {
    routeGet({
      executions: { items: [STALE_EXECUTION], total: 1, offset: 0, limit: 20 },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByLabelText('Select diag/vmstat.sh')).toBeTruthy();
    });
    expect(
      (screen.getByLabelText('Select diag/vmstat.sh') as HTMLInputElement)
        .disabled
    ).toBe(false);
  });

  it('shows why a past attempt failed', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [
          failedJob({
            error: 'upstream exploded',
            executions: [FINISHED_EXECUTION],
          }),
        ],
        total: 1,
        offset: 0,
        limit: 50,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText('upstream exploded')).toBeTruthy();
    });
  });

  it('re-sends every execution the attempt recorded, not just the current page', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 40,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [failedJob({ error: 'boom', executions: [OFF_PAGE_EXECUTION] })],
        total: 1,
        offset: 0,
        limit: 50,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Re-send' })).toBeTruthy();
    });
    fireEvent.click(screen.getByRole('button', { name: 'Re-send' }));

    await waitFor(() => {
      expect(screen.getByText('diag/off-page.sh')).toBeTruthy();
    });
    expect(
      screen.queryByText(/None of the selected executions still exist/i)
    ).toBeNull();
  });

  it('flags a send history it could not load', async () => {
    mockedApi.get.mockImplementation((url: string) => {
      if (url.includes('/send-jobs/')) {
        return Promise.reject(new Error('history unavailable'));
      }
      if (url.includes('/executions/')) {
        return Promise.resolve({
          data: { items: [FINISHED_EXECUTION], total: 1, offset: 0, limit: 20 },
        });
      }
      if (url.includes('/config/')) {
        return Promise.resolve({ data: { send_disabled_reasons: [] } });
      }
      return Promise.resolve({ data: { id: 'inc-1', case_ref: 'CS0001' } });
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText(/Could not load the send history/i)).toBeTruthy();
    });
  });

  it('cues that the send history is truncated', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [failedJob({ error: 'boom', executions: [FINISHED_EXECUTION] })],
        total: 73,
        offset: 0,
        limit: 50,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(
        screen.getByText(/Showing the 1 most recent of 73 attempts/i)
      ).toBeTruthy();
    });
  });
});
