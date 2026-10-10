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
import type { ReactNode } from 'react';
import { DeliverySettingsProvider } from '../src/deliverySettings';
import { IncidentListPage } from '../src/IncidentListPage';
import { ATW_INCIDENT_LIST_LIMIT } from '../src/hooks';
import {
  Messages,
  SUPPORT_DIAGNOSTICS_DOCS_URL,
} from '../src/IncidentsEmptyState.messages';
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

const SETTINGS_PATH = '/settings/servicenow-connection';

const incident: AtwIncident = {
  id: '11111111-1111-4111-8111-111111111111',
  name: 'DB slowness',
  case_ref: 'CS-42',
  created_by: 'engineer',
  created_at: '2026-07-22T10:00:00Z',
  updated_at: null,
  closed_at: null,
  run_count: 3,
  failed_run_count: 2,
  last_activity_at: '2026-07-23T09:00:00Z',
};

const other: AtwIncident = {
  ...incident,
  id: '22222222-2222-4222-8222-222222222222',
  name: 'Lock waits',
  case_ref: null,
  run_count: 0,
  failed_run_count: 0,
  last_activity_at: '2026-07-24T09:00:00Z',
};

function paginated<T>(items: T[]) {
  return { data: { items, total: items.length, offset: 0, limit: 50 } };
}

/**
 * Route GETs for the list page: incidents vs delivery config. Config defaults
 * to configured (no reasons) so most tests stay quiet about ServiceNow.
 */
function routeGet(
  options: {
    incidents?: AtwIncident[] | 'reject';
    config?: { send_disabled_reasons?: string[] };
  } = {}
) {
  mockedApi.get.mockImplementation((url: string) => {
    if (url.includes('/config/')) {
      return Promise.resolve({
        data: {
          send_disabled_reasons: options.config?.send_disabled_reasons ?? [],
          case_search_available: false,
        },
      });
    }
    if (options.incidents === 'reject') {
      return Promise.reject(new Error('Internal Server Error'));
    }
    return Promise.resolve(paginated(options.incidents ?? []));
  });
}

function renderPage(ui: ReactNode, deliverySettingsPath?: string) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/']}>
        <DeliverySettingsProvider path={deliverySettingsPath}>
          <Routes>
            <Route path="/" element={ui} />
            <Route
              path="/:incidentId"
              element={<div data-testid="workspace-route" />}
            />
          </Routes>
        </DeliverySettingsProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

async function chooseAction(
  user: ReturnType<typeof userEvent.setup>,
  incidentName: string,
  action: RegExp
) {
  await user.click(
    screen.getByRole('button', { name: `Actions for ${incidentName}` })
  );
  await user.click(await screen.findByRole('menuitem', { name: action }));
}

function rowNames() {
  return screen
    .getAllByRole('row')
    .slice(1)
    .map((row) => within(row).getByRole('link').textContent);
}

describe('IncidentListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders each incident as a table row with its status, counts and case', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));

    renderPage(<IncidentListPage />);

    const row = (await screen.findByText('DB slowness')).closest('tr');
    expect(row).not.toBeNull();
    const cells = within(row as HTMLElement);
    expect(cells.getByText('CS-42')).toBeInTheDocument();
    expect(cells.getByText('Open')).toBeInTheDocument();
    expect(cells.getByText('3')).toBeInTheDocument();
    expect(cells.getByText('2')).toBeInTheDocument();
    expect(cells.getByText('engineer')).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: /Not collected/ })
    ).toBeInTheDocument();
  });

  it('shows a status chip for a closed incident too', async () => {
    mockedApi.get.mockResolvedValue(
      paginated([{ ...incident, closed_at: '2026-07-30T12:00:00Z' }])
    );

    renderPage(<IncidentListPage />);

    expect(await screen.findByText('Closed')).toBeInTheDocument();
  });

  it('fetches one window of incidents at the side-car ceiling', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    expect(mockedApi.get).toHaveBeenCalledWith('/apps/atw/incidents/', {
      params: { offset: 0, limit: ATW_INCIDENT_LIST_LIMIT },
    });
  });

  it('says so when there are more incidents than the window holds', async () => {
    mockedApi.get.mockResolvedValue({
      data: { items: [incident], total: 240, offset: 0, limit: 200 },
    });

    renderPage(<IncidentListPage />);

    expect(
      await screen.findByText(/Showing the 1 most recently created of 240/)
    ).toBeInTheDocument();
  });

  it('puts the most recently active incident first', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident, other]));

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    expect(rowNames()).toEqual(['Lock waits', 'DB slowness']);
  });

  it('filters the rows by the search box', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident, other]));
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    await user.type(screen.getByPlaceholderText(/Search/i), 'CS-42');

    await waitFor(() => expect(rowNames()).toEqual(['DB slowness']));
  });

  it('shows an explanatory empty state when there are no incidents', async () => {
    routeGet({ incidents: [] });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty')).toBeTruthy();
    });
    expect(screen.getByText(Messages.title)).toBeTruthy();
    expect(screen.getByText(Messages.description)).toBeTruthy();
    expect(screen.getByTestId('atw-incidents-empty-docs')).toHaveAttribute(
      'href',
      SUPPORT_DIAGNOSTICS_DOCS_URL
    );
    expect(screen.getByText(Messages.documentation)).toBeTruthy();
  });

  it('puts New incident inside the empty state, not the header', async () => {
    routeGet({ incidents: [] });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty-create')).toBeTruthy();
    });
    expect(screen.getByTestId('atw-incidents-empty-create')).toHaveTextContent(
      Messages.create
    );
    // Only the empty-state CTA — no second header button.
    expect(
      screen.getAllByRole('button', { name: /New incident/i })
    ).toHaveLength(1);
  });

  it('withholds the create button when the list request failed', async () => {
    mockedApi.get.mockRejectedValue(new Error('Internal Server Error'));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(
        screen.getByText(/Failed to load incidents: Internal Server Error/)
      ).toBeTruthy();
    });
    // Creating would hit the backend that just failed, so the action is gone
    // rather than merely disabled.
    expect(screen.queryByRole('button', { name: /New incident/i })).toBeNull();
  });

  it('withholds the ServiceNow banner when the list request failed', async () => {
    routeGet({
      incidents: 'reject',
      config: {
        send_disabled_reasons: ['Diagnostics delivery is not configured'],
      },
    });

    renderPage(<IncidentListPage />, SETTINGS_PATH);

    await waitFor(() => {
      expect(
        screen.getByText(/Failed to load incidents: Internal Server Error/)
      ).toBeTruthy();
    });
    expect(
      screen.queryByTestId('atw-send-unavailable')
    ).not.toBeInTheDocument();
  });

  it('disables the create button until the list has loaded', async () => {
    mockedApi.get.mockReturnValue(new Promise(() => {}));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /New incident/i })
      ).toBeDisabled();
    });
  });

  it('creates an incident in one click and opens it', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await user.click(await screen.findByTestId('atw-incidents-empty-create'));

    expect(mockedApi.post).toHaveBeenCalledWith('/apps/atw/incidents/', {});
    expect(await screen.findByTestId('workspace-route')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('reports a failed create on the list', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));
    mockedApi.post.mockRejectedValue(new Error('Side-car unavailable'));
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await user.click(await screen.findByTestId('atw-incidents-empty-create'));

    expect(await screen.findByText('Side-car unavailable')).toBeInTheDocument();
  });

  it('opens an incident when its row is clicked', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await user.click(await screen.findByText('engineer'));

    expect(await screen.findByTestId('workspace-route')).toBeInTheDocument();
  });

  it('closes an open incident from the actions menu', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    mockedApi.post.mockResolvedValue({
      data: { ...incident, closed_at: '2026-07-30T12:00:00Z' },
    });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    await chooseAction(user, 'DB slowness', /^Close$/);

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incident.id}/close/`
      );
    });
    // Choosing an action is not a way into the incident.
    expect(screen.queryByTestId('workspace-route')).not.toBeInTheDocument();
  });

  it('reopens a closed incident from the actions menu', async () => {
    const closedIncident = { ...incident, closed_at: '2026-07-30T12:00:00Z' };
    mockedApi.get.mockResolvedValue(paginated([closedIncident]));
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('Closed');
    await chooseAction(user, 'DB slowness', /^Reopen$/);

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incident.id}/reopen/`
      );
    });
  });

  it('shows an error when closing an incident fails', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    mockedApi.post.mockRejectedValue(new Error('Incident is already closed.'));
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    await chooseAction(user, 'DB slowness', /^Close$/);

    expect(
      await screen.findByText('Incident is already closed.')
    ).toBeInTheDocument();
  });

  it('disables Close for an incident whose close is still in flight', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    let resolveClose!: (value: { data: AtwIncident }) => void;
    mockedApi.post.mockReturnValue(
      new Promise((resolve) => {
        resolveClose = resolve;
      })
    );
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    await chooseAction(user, 'DB slowness', /^Close$/);
    await user.click(
      screen.getByRole('button', { name: 'Actions for DB slowness' })
    );
    expect(
      await screen.findByRole('menuitem', { name: /^Close$/ })
    ).toHaveAttribute('aria-disabled', 'true');

    resolveClose({ data: { ...incident, closed_at: '2026-07-30T12:00:00Z' } });
    await waitFor(() =>
      expect(
        screen.getByRole('menuitem', { name: /^Close$/ })
      ).not.toHaveAttribute('aria-disabled')
    );
  });

  it('renames an incident from the actions menu', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    mockedApi.patch.mockResolvedValue({
      data: { ...incident, name: 'Replica lag' },
    });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    await chooseAction(user, 'DB slowness', /^Rename$/);
    const field = await screen.findByLabelText('Name');
    await user.clear(field);
    await user.type(field, '  Replica lag  ');
    await user.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() => {
      expect(mockedApi.patch).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incident.id}`,
        { name: 'Replica lag' }
      );
    });
  });

  it('deletes an incident from the actions menu after confirming', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    mockedApi.delete.mockResolvedValue({});
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    await chooseAction(user, 'DB slowness', /^Delete$/);
    const dialog = await screen.findByRole('dialog');
    expect(
      within(dialog).getByText(/This cannot be undone/)
    ).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: 'Delete' }));

    await waitFor(() => {
      expect(mockedApi.delete).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incident.id}`
      );
    });
  });
});

describe('IncidentListPage — write access', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('offers create and one labelled actions menu with rename, close and delete', async () => {
    mockedApi.get.mockResolvedValue(paginated([incident]));
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    expect(
      screen.getByRole('button', { name: /New incident/i })
    ).toBeInTheDocument();
    await user.click(
      screen.getByRole('button', { name: 'Actions for DB slowness' })
    );
    const items = (await screen.findAllByRole('menuitem')).map(
      (item) => item.textContent
    );
    expect(items).toEqual(['Rename', 'Close', 'Delete']);
  });

  it('omits New incident from the empty state for a non-admin', async () => {
    mockCanMutate = false;
    mockedApi.get.mockResolvedValue(paginated([]));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty')).toBeTruthy();
    });
    expect(screen.getByText(Messages.title)).toBeTruthy();
    expect(screen.getByText(Messages.description)).toBeTruthy();
    expect(
      screen.queryByRole('button', { name: /New incident/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('atw-incidents-empty-create')
    ).not.toBeInTheDocument();
  });

  it('keeps New incident in the empty state for a session that may mutate', async () => {
    mockedApi.get.mockResolvedValue(paginated([]));

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty-create')).toBeTruthy();
    });
  });

  it('renders no create or actions menu for a non-admin', async () => {
    mockCanMutate = false;
    mockedApi.get.mockResolvedValue(paginated([incident]));

    renderPage(<IncidentListPage />);

    await screen.findByText('DB slowness');
    expect(
      screen.queryByRole('button', { name: /New incident/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /Actions for/i })
    ).not.toBeInTheDocument();
  });
});

describe('IncidentListPage — ServiceNow connection banner', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const unconfigured = {
    send_disabled_reasons: ['Diagnostics delivery is not configured'],
  };

  it('shows the connection banner and Settings link for an admin when delivery is missing', async () => {
    routeGet({ incidents: [], config: unconfigured });

    renderPage(<IncidentListPage />, SETTINGS_PATH);

    await waitFor(() => {
      expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
    });
    expect(
      screen.getByText(/Sending requires a valid ServiceNow connection/i)
    ).toBeTruthy();
    expect(screen.getByTestId('atw-send-unavailable-settings')).toHaveAttribute(
      'href',
      SETTINGS_PATH
    );
  });

  it('explains without a Settings control when the host offers no route', async () => {
    routeGet({ incidents: [], config: unconfigured });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
    });
    expect(
      screen.queryByTestId('atw-send-unavailable-settings')
    ).not.toBeInTheDocument();
  });

  it('stays silent while delivery is configured', async () => {
    routeGet({ incidents: [] });

    renderPage(<IncidentListPage />, SETTINGS_PATH);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty')).toBeTruthy();
    });
    expect(
      screen.queryByTestId('atw-send-unavailable')
    ).not.toBeInTheDocument();
  });

  it('hides the connection banner from a non-admin', async () => {
    mockCanMutate = false;
    routeGet({ incidents: [], config: unconfigured });

    renderPage(<IncidentListPage />, SETTINGS_PATH);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty')).toBeTruthy();
    });
    expect(
      screen.queryByTestId('atw-send-unavailable')
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('atw-send-unavailable-settings')
    ).not.toBeInTheDocument();
  });

  it('shows the banner on a populated list when delivery is missing', async () => {
    routeGet({ incidents: [incident], config: unconfigured });

    renderPage(<IncidentListPage />, SETTINGS_PATH);

    await waitFor(() => {
      expect(screen.getByText('DB slowness')).toBeTruthy();
    });
    expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
    expect(screen.getByTestId('atw-send-unavailable-settings')).toHaveAttribute(
      'href',
      SETTINGS_PATH
    );
  });

  it('keeps the banner when a remount refetch of config fails', async () => {
    routeGet({ incidents: [], config: unconfigured });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Infinity } },
    });
    const tree = (ui: ReactNode) => (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeliverySettingsProvider path={SETTINGS_PATH}>
            {ui}
          </DeliverySettingsProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );

    const configCalls = () =>
      mockedApi.get.mock.calls.filter(([url]) =>
        String(url).includes('/config/')
      );

    const { unmount } = render(tree(<IncidentListPage />));

    await waitFor(() => {
      expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
    });
    expect(configCalls()).toHaveLength(1);

    mockedApi.get.mockImplementation((url: string) => {
      if (url.includes('/config/')) {
        return Promise.reject(new Error('config unavailable'));
      }
      return Promise.resolve(paginated([]));
    });

    unmount();
    render(tree(<IncidentListPage />));

    await waitFor(() => {
      expect(configCalls()).toHaveLength(2);
      expect(queryClient.isFetching()).toBe(0);
    });
    expect(screen.getByTestId('atw-send-unavailable')).toBeTruthy();
  });
});
