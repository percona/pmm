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

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import { IncidentWorkspacePage } from '../src/IncidentWorkspacePage';
import type { AtwIncident } from '../src/types';

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@pmm-extensions/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@pmm-extensions/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

import { apiClient } from '@pmm-extensions/api';
const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
  patch: ReturnType<typeof vi.fn>;
  delete: ReturnType<typeof vi.fn>;
};

const incidentId = '11111111-1111-4111-8111-111111111111';
const openIncident: AtwIncident = {
  id: incidentId,
  name: 'DB slowness',
  case_ref: 'CS-42',
  created_by: 'engineer',
  created_at: '2026-07-22T10:00:00Z',
  updated_at: null,
  closed_at: null,
  run_count: 0,
  failed_run_count: 0,
};
const closedIncident = { ...openIncident, closed_at: '2026-07-30T12:00:00Z' };

function serveIncident(incident: AtwIncident) {
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
    return { data: incident };
  });
}

async function chooseAction(
  user: ReturnType<typeof userEvent.setup>,
  action: RegExp
) {
  await user.click(
    await screen.findByRole('button', { name: 'Actions for DB slowness' })
  );
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

function renderWorkspace() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <SnackbarProvider>
        <MemoryRouter initialEntries={[`/atw/${incidentId}`]}>
          <Routes>
            <Route
              path="/atw/:incidentId"
              element={<IncidentWorkspacePage />}
            />
            {/* Where the workspace's route-relative `..` lands in this flat tree. */}
            <Route path="/" element={<div data-testid="list-route" />} />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
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

  it('shows an Open status chip beside the title of an open incident', async () => {
    renderWorkspace();

    const heading = await screen.findByRole('heading', { name: 'DB slowness' });
    expect(heading).toBeInTheDocument();
    expect(screen.getByTestId('atw-incident-status')).toHaveTextContent('Open');
  });

  it('puts close in the actions menu rather than in the header', async () => {
    const user = userEvent.setup();
    renderWorkspace();

    await screen.findByRole('heading', { name: 'DB slowness' });
    expect(
      screen.queryByRole('button', { name: /Close incident/i })
    ).not.toBeInTheDocument();
    await user.click(
      screen.getByRole('button', { name: 'Actions for DB slowness' })
    );
    const items = (await screen.findAllByRole('menuitem')).map(
      (item) => item.textContent
    );
    expect(items).toEqual(['Rename', 'Close', 'Delete']);
  });

  it('closes an incident from the actions menu', async () => {
    mockedApi.post.mockResolvedValue({ data: closedIncident });
    const user = userEvent.setup();

    renderWorkspace();
    await chooseAction(user, /^Close$/);

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
    await chooseAction(user, /^Close$/);

    expect(
      await screen.findByText('Incident is already closed.')
    ).toBeInTheDocument();
  });

  it('renames the incident in place', async () => {
    mockedApi.patch.mockResolvedValue({
      data: { ...openIncident, name: 'Replica lag' },
    });
    const user = userEvent.setup();

    renderWorkspace();
    await user.click(
      await screen.findByRole('button', { name: 'DB slowness' })
    );
    const field = screen.getByRole('textbox', { name: 'Incident name' });
    await user.clear(field);
    await user.type(field, '  Replica lag  {Enter}');

    await waitFor(() => {
      expect(mockedApi.patch).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incidentId}`,
        { name: 'Replica lag' }
      );
    });
    expect(mockedApi.patch).toHaveBeenCalledTimes(1);
  });

  it('starts the in-place rename from the actions menu', async () => {
    const user = userEvent.setup();

    renderWorkspace();
    await chooseAction(user, /^Rename$/);

    const field = await screen.findByRole('textbox', {
      name: 'Incident name',
    });
    expect(field).toHaveValue('DB slowness');
  });

  it('refuses an empty name and leaves the incident untouched', async () => {
    const user = userEvent.setup();

    renderWorkspace();
    await user.click(
      await screen.findByRole('button', { name: 'DB slowness' })
    );
    const field = screen.getByRole('textbox', { name: 'Incident name' });
    await user.clear(field);
    await user.keyboard('{Enter}');

    expect(mockedApi.patch).not.toHaveBeenCalled();
    expect(
      await screen.findByRole('heading', { name: 'DB slowness' })
    ).toBeInTheDocument();
  });

  it('cancels an in-place rename on Escape', async () => {
    const user = userEvent.setup();

    renderWorkspace();
    await user.click(
      await screen.findByRole('button', { name: 'DB slowness' })
    );
    await user.type(
      screen.getByRole('textbox', { name: 'Incident name' }),
      ' extra{Escape}'
    );

    expect(mockedApi.patch).not.toHaveBeenCalled();
    expect(
      screen.queryByRole('textbox', { name: 'Incident name' })
    ).not.toBeInTheDocument();
  });

  it('edits the case reference in place, clearing it to null', async () => {
    mockedApi.patch.mockResolvedValue({
      data: { ...openIncident, case_ref: null },
    });
    const user = userEvent.setup();

    renderWorkspace();
    await user.click(await screen.findByRole('button', { name: 'CS-42' }));
    const field = screen.getByRole('textbox', { name: 'Case reference' });
    await user.clear(field);
    await user.keyboard('{Enter}');

    await waitFor(() => {
      expect(mockedApi.patch).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incidentId}`,
        { case_ref: null }
      );
    });
  });

  it('deletes the incident from the actions menu and returns to the list', async () => {
    mockedApi.delete.mockResolvedValue({});
    const user = userEvent.setup();

    renderWorkspace();
    await chooseAction(user, /^Delete$/);
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }));

    await waitFor(() => {
      expect(mockedApi.delete).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incidentId}`
      );
    });
    expect(await screen.findByTestId('list-route')).toBeInTheDocument();
  });

  it('shows a closed incident with its status, one line, and no picker', async () => {
    serveIncident(closedIncident);

    renderWorkspace();

    expect(
      await screen.findByTestId('atw-incident-closed-notice')
    ).toHaveTextContent(/This incident is closed/);
    expect(screen.getByTestId('atw-incident-status')).toHaveTextContent(
      'Closed'
    );
    expect(screen.getByText('Results')).toBeInTheDocument();
    expect(screen.queryByText('Collect')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('combobox', { name: 'Snippets' })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(/Search for a snippet by name or description/)
    ).not.toBeInTheDocument();
  });

  it('offers Reopen, with the open-lock icon, for a closed incident', async () => {
    serveIncident(closedIncident);
    mockedApi.post.mockResolvedValue({ data: openIncident });
    const user = userEvent.setup();

    renderWorkspace();
    await user.click(
      await screen.findByRole('button', { name: 'Actions for DB slowness' })
    );
    const reopen = await screen.findByRole('menuitem', { name: /^Reopen$/ });
    expect(
      within(reopen).getByTestId('LockOpenOutlinedIcon')
    ).toBeInTheDocument();
    await user.click(reopen);

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incidentId}/reopen/`
      );
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

  it('shows the actions menu for a session that may mutate', async () => {
    renderWorkspace();

    expect(
      await screen.findByRole('button', { name: 'Actions for DB slowness' })
    ).toBeInTheDocument();
  });

  it('shows no actions or editable title for a non-admin, keeping the incident readable', async () => {
    mockCanMutate = false;
    renderWorkspace();

    expect(
      await screen.findByRole('heading', { name: 'DB slowness' })
    ).toBeInTheDocument();
    expect(screen.getByTestId('atw-incident-status')).toHaveTextContent('Open');
    expect(
      screen.queryByRole('button', { name: /Actions for/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'DB slowness' })
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

describe('IncidentWorkspacePage — edit parameters and run again', () => {
  const EXECUTION = {
    id: 'exec-1',
    snippet_filename: 'diag/vmstat.sh',
    task_history_id: 55,
    created_at: '2026-07-22T10:00:00Z',
    task_status: 'success',
    started_at: null,
    finished_at: null,
    has_logs: false,
  };
  const SNIPPET = {
    name: 'diag/vmstat.sh',
    title: 'VM Stat Snapshot',
    description: 'Captures vmstat output.',
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockedApi.get.mockImplementation(async (url: string) => {
      if (url.startsWith('/apps/atw/snippets/')) {
        return { data: { items: [SNIPPET], total: 1, offset: 0, limit: 50 } };
      }
      if (url.includes('/execution-schema/')) {
        return { data: { shared: [], per_snippet: [] } };
      }
      if (url.includes('/executions/')) {
        return {
          data: { items: [EXECUTION], total: 1, offset: 0, limit: 20 },
        };
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

  it('reopens the Collect pane pre-filled with the execution’s snippet', async () => {
    const user = userEvent.setup();
    renderWorkspace();

    await waitFor(() => {
      expect(screen.getByText('diag/vmstat.sh')).toBeTruthy();
    });
    // The row's actions live in AccordionDetails, unmounted until expanded.
    await user.click(screen.getByText('diag/vmstat.sh'));
    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Edit parameters and run again' })
      ).toBeTruthy();
    });

    await user.click(
      screen.getByRole('button', { name: 'Edit parameters and run again' })
    );

    // Resolved by name through the exact-name search, since the execution row
    // carries no reusable parameter payload — the snippet's title then shows
    // as the Collect form's selected chip.
    await waitFor(
      () => {
        expect(screen.getByText(SNIPPET.title)).toBeInTheDocument();
      },
      { timeout: 3000 }
    );
  });
});

describe('IncidentWorkspacePage — feedback for a run that just started', () => {
  const SNIPPET = {
    name: 'diag/vmstat.sh',
    title: 'VM Stat Snapshot',
    description: 'Captures vmstat output.',
  };
  const EXECUTION = {
    id: 'exec-1',
    snippet_filename: 'diag/vmstat.sh',
    task_history_id: 55,
    created_at: '2026-07-22T10:00:00Z',
    task_status: 'running',
    started_at: '2026-07-22T10:00:01Z',
    finished_at: null,
    has_logs: false,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockedApi.get.mockImplementation(async (url: string) => {
      if (url.startsWith('/apps/atw/snippets/')) {
        return { data: { items: [SNIPPET], total: 1, offset: 0, limit: 50 } };
      }
      if (url.includes('/execution-schema/')) {
        return { data: { shared: [], per_snippet: [] } };
      }
      if (url.includes('/executions/')) {
        return { data: { items: [EXECUTION], total: 1, offset: 0, limit: 20 } };
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
    mockedApi.post.mockResolvedValue({
      data: {
        items: [
          {
            snippet_filename: SNIPPET.name,
            task_history_id: EXECUTION.task_history_id,
            error: null,
          },
        ],
      },
    });
  });

  /** Pick the one searchable snippet and press the run button. */
  async function runOneSnippet() {
    const user = userEvent.setup();
    await user.type(
      await screen.findByRole('combobox', { name: 'Snippets' }),
      'vmstat'
    );
    await user.click(
      await screen.findByRole(
        'option',
        { name: /VM Stat Snapshot/ },
        { timeout: 3000 }
      )
    );
    await user.click(
      await screen.findByRole(
        'button',
        { name: 'Execute batch' },
        { timeout: 3000 }
      )
    );
  }

  it('names how many scripts started, and brings Results into view', async () => {
    const scrollIntoView = vi.fn();
    Element.prototype.scrollIntoView = scrollIntoView;

    renderWorkspace();
    await runOneSnippet();

    expect(await screen.findByText('Started 1 script')).toBeVisible();
    await waitFor(() => expect(scrollIntoView).toHaveBeenCalled());
  });

  it('marks out the executions the run produced', async () => {
    Element.prototype.scrollIntoView = vi.fn();

    renderWorkspace();
    await runOneSnippet();

    expect(
      await screen.findByTestId('atw-execution-row-new', undefined, {
        timeout: 3000,
      })
    ).toBeVisible();
  });
});
