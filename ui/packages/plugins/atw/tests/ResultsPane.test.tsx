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

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

// No test here mounts the log viewer: rows that stay collapsed leave it unmounted
// (unmountOnExit), and the rows that expand report no logs. The files dialog stays
// closed throughout — so none of them fires a query.

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
      expect(
        screen.getByText(/Run snippets from the Collect pane/i)
      ).toBeTruthy();
    });
  });

  it('does not point a read-only session at the withheld Collect pane', async () => {
    mockCanMutate = false;
    mockedApi.get.mockResolvedValue(paginated([]));

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText(/No executions yet/i)).toBeTruthy();
    });
    expect(screen.queryByText(/Collect pane/i)).not.toBeInTheDocument();
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

// ── Recorded arguments ───────────────────────────────────────────────────

/** Build one execution row, overriding only what a case cares about. */
function executionWithArgs(overrides: Record<string, unknown>) {
  return {
    id: 'exec-args',
    snippet_filename: 'diag/mongo.sh',
    task_history_id: 21,
    created_at: '2026-07-22T10:00:00Z',
    task_status: 'success',
    started_at: null,
    finished_at: null,
    // Keeps the expanded body on its no-logs branch, so these cases can open a
    // row without mounting the log viewer and its token/stream machinery.
    has_logs: false,
    masked_args: null,
    args_withheld: false,
    ...overrides,
  };
}

const MASKED_ARGS = '--port 27017 --password ***';

describe('ResultsPane recorded arguments', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the masked arguments in the collapsed summary', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([executionWithArgs({ masked_args: MASKED_ARGS })])
    );

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(screen.getByText(MASKED_ARGS)).toBeTruthy();
    });
  });

  it('renders the arguments in the expanded body as well as the summary', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([executionWithArgs({ masked_args: MASKED_ARGS })])
    );

    renderPane(<ResultsPane incidentId="inc-1" />);
    await waitFor(() => {
      expect(screen.getByText('diag/mongo.sh')).toBeTruthy();
    });

    fireEvent.click(screen.getByText('diag/mongo.sh'));

    await waitFor(() => {
      expect(screen.getAllByText(MASKED_ARGS)).toHaveLength(2);
    });
  });

  it('wraps the arguments in the body while the summary keeps them on one line', async () => {
    const longArgs = `--dest /var/tmp/${'long-path-segment/'.repeat(12)} --password ***`;
    mockedApi.get.mockResolvedValue(
      paginated([executionWithArgs({ masked_args: longArgs })])
    );

    renderPane(<ResultsPane incidentId="inc-1" />);
    await waitFor(() => {
      expect(screen.getByText('diag/mongo.sh')).toBeTruthy();
    });

    fireEvent.click(screen.getByText('diag/mongo.sh'));

    await waitFor(() => {
      expect(screen.getAllByText(longArgs)).toHaveLength(2);
    });
    const [summaryLine, bodyLine] = screen.getAllByText(longArgs);
    expect(getComputedStyle(summaryLine).whiteSpace).toBe('nowrap');
    expect(getComputedStyle(summaryLine).textOverflow).toBe('ellipsis');
    expect(getComputedStyle(bodyLine).whiteSpace).toBe('pre-wrap');
  });

  it('shows an empty state when the execution recorded no arguments', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([
        executionWithArgs({ masked_args: null, args_withheld: false }),
      ])
    );

    renderPane(<ResultsPane incidentId="inc-1" />);
    await waitFor(() => {
      expect(screen.getByText('diag/mongo.sh')).toBeTruthy();
    });

    fireEvent.click(screen.getByText('diag/mongo.sh'));

    await waitFor(() => {
      expect(screen.getByText('No arguments')).toBeTruthy();
    });
  });

  it('reports unavailable arguments distinctly from an execution that had none', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([executionWithArgs({ masked_args: null, args_withheld: true })])
    );

    renderPane(<ResultsPane incidentId="inc-1" />);
    await waitFor(() => {
      expect(screen.getByText('diag/mongo.sh')).toBeTruthy();
    });

    fireEvent.click(screen.getByText('diag/mongo.sh'));

    await waitFor(() => {
      expect(screen.getByText('Arguments unavailable')).toBeTruthy();
    });
    expect(screen.queryByText('No arguments')).toBeNull();
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
interface ExecutionsPage {
  items: unknown[];
  total: number;
  offset: number;
  limit: number;
}

function routeGet(routes: {
  executions?: ExecutionsPage | ((offset: number) => ExecutionsPage);
  config?: unknown;
  sendJobs?: unknown;
  incident?: unknown;
}) {
  mockedApi.get.mockImplementation(
    (url: string, config?: { params?: { offset?: number } }) => {
      if (url.includes('/send-jobs/')) {
        return Promise.resolve({
          data: routes.sendJobs ?? {
            items: [],
            total: 0,
            offset: 0,
            limit: 50,
          },
        });
      }
      if (url.includes('/executions/')) {
        // A function route answers per requested offset, so a case can span pages;
        // the offset arrives in the axios params, not the URL.
        const executions =
          typeof routes.executions === 'function'
            ? routes.executions(config?.params?.offset ?? 0)
            : routes.executions;
        return Promise.resolve({
          data: executions ?? { items: [], total: 0, offset: 0, limit: 20 },
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
    }
  );
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

// ── Page-scoped select all ───────────────────────────────────────────────

/** A second page's finished execution, so a selection can span two pages. */
const SECOND_PAGE_EXECUTION = {
  id: 'exec-4',
  snippet_filename: 'diag/iostat.sh',
  task_history_id: 11,
  created_at: '2026-07-22T10:03:00Z',
  task_status: 'success',
  started_at: null,
  finished_at: null,
  has_logs: true,
};

const SELECT_ALL_LABEL = 'Select all finished executions on this page';

const selectAllToggle = () =>
  screen.getByLabelText(SELECT_ALL_LABEL) as HTMLInputElement;

const rowToggle = (filename: string) =>
  screen.getByLabelText(`Select ${filename}`) as HTMLInputElement;

/**
 * How the toggle presents itself to a screen reader.
 *
 * MUI leaves the native `checked` property false in the mixed state, so the
 * indeterminate case has to be read off an attribute. `aria-checked="mixed"`
 * is the contract a screen reader consumes, but MUI only started emitting it
 * after 7.3.7 — the version this repo's lockfile resolves — so fall back to
 * the `data-indeterminate` attribute MUI documents for that release.
 */
const toggleState = () => {
  const toggle = selectAllToggle();
  const indeterminate =
    toggle.getAttribute('aria-checked') === 'mixed' ||
    toggle.getAttribute('data-indeterminate') === 'true';
  if (indeterminate) {
    return 'mixed';
  }
  return toggle.checked ? 'checked' : 'unchecked';
};

/** Serve `pages` keyed by offset, so a case can select across a page flip. */
function pagesByOffset(pages: Record<number, unknown[]>, total: number) {
  return (offset: number) => ({
    items: pages[offset] ?? [],
    total,
    offset,
    limit: 20,
  });
}

describe('ResultsPane select all on the page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('selects every finished execution on the page and skips the unfinished ones', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION, RUNNING_EXECUTION, STALE_EXECUTION],
        total: 3,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(selectAllToggle()).toBeTruthy();
    });
    fireEvent.click(selectAllToggle());

    await waitFor(() => {
      expect(screen.getByText('2 selected')).toBeTruthy();
    });
    expect(rowToggle('diag/slow-query.sh').checked).toBe(true);
    expect(rowToggle('diag/vmstat.sh').checked).toBe(true);
    expect(rowToggle('diag/dmesg.sh').checked).toBe(false);
    expect(toggleState()).toBe('checked');
    expect(
      (
        screen.getByRole('button', {
          name: /Send to support case/i,
        }) as HTMLButtonElement
      ).disabled
    ).toBe(false);
  });

  it('deselects exactly the page it selected', async () => {
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
      expect(selectAllToggle()).toBeTruthy();
    });
    fireEvent.click(selectAllToggle());
    await waitFor(() => {
      expect(screen.getByText('1 selected')).toBeTruthy();
    });

    fireEvent.click(selectAllToggle());

    await waitFor(() => {
      expect(screen.getByText('0 selected')).toBeTruthy();
    });
    expect(rowToggle('diag/slow-query.sh').checked).toBe(false);
  });

  it('renders the toggle indeterminate while only some finished rows are selected', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION, STALE_EXECUTION],
        total: 2,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(selectAllToggle()).toBeTruthy();
    });
    expect(toggleState()).toBe('unchecked');

    fireEvent.click(rowToggle('diag/slow-query.sh'));

    await waitFor(() => {
      expect(toggleState()).toBe('mixed');
    });

    fireEvent.click(rowToggle('diag/vmstat.sh'));

    await waitFor(() => {
      expect(toggleState()).toBe('checked');
    });
  });

  it('disables the toggle when the page holds no finished execution', async () => {
    routeGet({
      executions: {
        items: [RUNNING_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(selectAllToggle()).toBeTruthy();
    });
    expect(selectAllToggle().disabled).toBe(true);
    expect(toggleState()).toBe('unchecked');
  });

  it('leaves another page selected when deselecting the current one', async () => {
    routeGet({
      executions: pagesByOffset(
        {
          0: [FINISHED_EXECUTION, RUNNING_EXECUTION],
          20: [SECOND_PAGE_EXECUTION],
        },
        40
      ),
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(selectAllToggle()).toBeTruthy();
    });
    fireEvent.click(selectAllToggle());
    await waitFor(() => {
      expect(screen.getByText('1 selected')).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: /Go to next page/i }));
    await waitFor(() => {
      expect(screen.getByText('diag/iostat.sh')).toBeTruthy();
    });
    fireEvent.click(selectAllToggle());
    await waitFor(() => {
      expect(screen.getByText('2 selected')).toBeTruthy();
    });

    // Unchecking page 2 must not touch the execution chosen on page 1.
    fireEvent.click(selectAllToggle());

    await waitFor(() => {
      expect(screen.getByText('1 selected')).toBeTruthy();
    });
    expect(rowToggle('diag/iostat.sh').checked).toBe(false);
    expect(
      (
        screen.getByRole('button', {
          name: /Send to support case/i,
        }) as HTMLButtonElement
      ).disabled
    ).toBe(false);
  });

  it('labels an execution the toggle selected after the user pages away from it', async () => {
    routeGet({
      executions: pagesByOffset(
        { 0: [FINISHED_EXECUTION], 20: [SECOND_PAGE_EXECUTION] },
        40
      ),
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => {
      expect(selectAllToggle()).toBeTruthy();
    });
    fireEvent.click(selectAllToggle());
    await waitFor(() => {
      expect(screen.getByText('1 selected')).toBeTruthy();
    });

    fireEvent.click(screen.getByRole('button', { name: /Go to next page/i }));
    await waitFor(() => {
      expect(screen.getByText('diag/iostat.sh')).toBeTruthy();
    });
    expect(screen.queryByText('diag/slow-query.sh')).toBeNull();

    fireEvent.click(
      screen.getByRole('button', { name: /Send to support case/i })
    );

    await waitFor(() => {
      expect(screen.getByText('diag/slow-query.sh')).toBeTruthy();
    });
  });
});

describe('ResultsPane — write access', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the send and re-send actions for a session that may mutate', async () => {
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [failedJob({ error: 'upstream exploded', executions: [] })],
        total: 1,
        offset: 0,
        limit: 50,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => expect(screen.getByText('Send history')).toBeTruthy());
    expect(
      screen.getByRole('button', { name: /Send to support case/i })
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Re-send' })).toBeInTheDocument();
  });

  it('renders no send, re-send or selection controls for a non-admin', async () => {
    mockCanMutate = false;
    routeGet({
      executions: {
        items: [FINISHED_EXECUTION],
        total: 1,
        offset: 0,
        limit: 20,
      },
      sendJobs: {
        items: [failedJob({ error: 'upstream exploded', executions: [] })],
        total: 1,
        offset: 0,
        limit: 50,
      },
    });

    renderPane(<ResultsPane incidentId="inc-1" />);

    await waitFor(() => expect(screen.getByText('Send history')).toBeTruthy());
    expect(
      screen.queryByRole('button', { name: /Send to support case/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Re-send' })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByLabelText('Select diag/slow-query.sh')
    ).not.toBeInTheDocument();
    // Results and their history stay readable.
    expect(screen.getByText(/CS0042/)).toBeTruthy();
  });
});
