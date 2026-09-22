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
import { DeliverySettingsProvider } from '../src/deliverySettings';
import { IncidentListPage } from '../src/IncidentListPage';
import {
  Messages,
  SUPPORT_DIAGNOSTICS_DOCS_URL,
} from '../src/IncidentsEmptyState.messages';
import type { AtwIncident } from '../src/types';

/** Flipped per test to cover the read-only (non-admin) rendering. */
let mockCanMutate = true;

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), delete: vi.fn() },
  useAuth: () => ({ isAdmin: mockCanMutate, canMutate: mockCanMutate }),
}));

beforeEach(() => {
  mockCanMutate = true;
});

import { apiClient } from '@sep/api';
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
  run_count: 0,
  failed_run_count: 0,
};

function paginated<T>(items: T[], total = items.length, limit = 50) {
  return { data: { items, total, offset: 0, limit } };
}

/**
 * Route GETs for the list page: incidents vs delivery config. Config defaults
 * to configured (no reasons) so most tests stay quiet about ServiceNow.
 */
function routeGet(
  options: {
    incidents?: AtwIncident[] | 'reject' | 'hang';
    incidentsPage?: {
      items: AtwIncident[];
      total: number;
      offset: number;
      limit: number;
    };
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
    if (options.incidents === 'hang') {
      return new Promise(() => {});
    }
    if (options.incidents === 'reject') {
      return Promise.reject(new Error('Internal Server Error'));
    }
    if (options.incidentsPage) {
      return Promise.resolve({ data: options.incidentsPage });
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
      <MemoryRouter>
        <DeliverySettingsProvider path={deliverySettingsPath}>
          {ui}
        </DeliverySettingsProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('IncidentListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the incident list', async () => {
    routeGet({ incidents: [incident] });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByText('DB slowness')).toBeTruthy();
    });
    expect(screen.getByText(/Case CS-42/)).toBeTruthy();
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
    expect(screen.getByText(Messages.howItWorks)).toBeTruthy();
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
    routeGet({ incidents: 'reject' });

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

  it('disables the create button until the list has loaded', async () => {
    routeGet({ incidents: 'hang' });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /New incident/i })
      ).toBeDisabled();
    });
  });

  it('creates an incident from the dialog, sending the trimmed name', async () => {
    routeGet({ incidents: [] });
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() =>
      expect(screen.getByTestId('atw-incidents-empty-create')).toBeTruthy()
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
    routeGet({ incidents: [] });
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() =>
      expect(screen.getByTestId('atw-incidents-empty-create')).toBeTruthy()
    );

    await user.click(screen.getByRole('button', { name: /New incident/i }));
    await user.click(screen.getByRole('button', { name: /^Create$/ }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith('/apps/atw/incidents/', {});
    });
  });

  it('renders pagination controls once the list exceeds one page', async () => {
    routeGet({
      incidentsPage: { items: [incident], total: 40, offset: 0, limit: 20 },
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
    routeGet({ incidents: [incident] });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByText('DB slowness')).toBeTruthy();
    });
    expect(
      screen.queryByRole('button', { name: /Go to next page/i })
    ).toBeNull();
  });

  it('closes an open incident from the row action', async () => {
    routeGet({ incidents: [incident] });
    mockedApi.post.mockResolvedValue({
      data: { ...incident, closed_at: '2026-07-30T12:00:00Z' },
    });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() => expect(screen.getByText('DB slowness')).toBeTruthy());
    await user.click(
      screen.getByRole('button', { name: /Close DB slowness/i })
    );

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incident.id}/close/`
      );
    });
  });

  it('reopens a closed incident from the row action', async () => {
    const closedIncident = { ...incident, closed_at: '2026-07-30T12:00:00Z' };
    routeGet({ incidents: [closedIncident] });
    mockedApi.post.mockResolvedValue({ data: incident });
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() => expect(screen.getByText('Closed')).toBeTruthy());
    await user.click(
      screen.getByRole('button', { name: /Reopen DB slowness/i })
    );

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        `/apps/atw/incidents/${incident.id}/reopen/`
      );
    });
  });

  it('shows an error when closing an incident fails', async () => {
    routeGet({ incidents: [incident] });
    mockedApi.post.mockRejectedValue(new Error('Incident is already closed.'));
    const user = userEvent.setup();

    renderPage(<IncidentListPage />);

    await waitFor(() => expect(screen.getByText('DB slowness')).toBeTruthy());
    await user.click(
      screen.getByRole('button', { name: /Close DB slowness/i })
    );

    await waitFor(() => {
      expect(screen.getByText('Incident is already closed.')).toBeTruthy();
    });
  });

  it('re-enables the first row close button after overlapping closes settle', async () => {
    const other = {
      ...incident,
      id: '22222222-2222-4222-8222-222222222222',
      name: 'Lock waits',
    };
    routeGet({ incidents: [incident, other] });

    let resolveFirst!: (value: { data: typeof incident }) => void;
    let resolveSecond!: (value: { data: typeof other }) => void;
    const firstClose = new Promise<{ data: typeof incident }>((resolve) => {
      resolveFirst = resolve;
    });
    const secondClose = new Promise<{ data: typeof other }>((resolve) => {
      resolveSecond = resolve;
    });
    mockedApi.post.mockImplementation((url: string) => {
      if (url.endsWith(`/${incident.id}/close/`)) {
        return firstClose;
      }
      if (url.endsWith(`/${other.id}/close/`)) {
        return secondClose;
      }
      return Promise.reject(new Error(`Unexpected POST ${url}`));
    });

    const user = userEvent.setup();
    renderPage(<IncidentListPage />);

    await waitFor(() => expect(screen.getByText('DB slowness')).toBeTruthy());
    const firstCloseButton = screen.getByRole('button', {
      name: /Close DB slowness/i,
    });
    const secondCloseButton = screen.getByRole('button', {
      name: /Close Lock waits/i,
    });

    await user.click(firstCloseButton);
    await waitFor(() => expect(firstCloseButton).toBeDisabled());

    await user.click(secondCloseButton);
    await waitFor(() => {
      expect(firstCloseButton).toBeDisabled();
      expect(secondCloseButton).toBeDisabled();
    });

    resolveSecond({ data: { ...other, closed_at: '2026-07-30T12:00:00Z' } });

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Close Lock waits/i })
      ).not.toBeDisabled();
    });
    expect(
      screen.getByRole('button', { name: /Close DB slowness/i })
    ).toBeDisabled();

    resolveFirst({ data: { ...incident, closed_at: '2026-07-30T12:00:00Z' } });

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /Close DB slowness/i })
      ).not.toBeDisabled();
    });
  });
});

describe('IncidentListPage — write access', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders create, close, rename and delete for a session that may mutate', async () => {
    routeGet({ incidents: [incident] });

    renderPage(<IncidentListPage />);

    await waitFor(() => expect(screen.getByText('DB slowness')).toBeTruthy());
    expect(
      screen.getByRole('button', { name: /New incident/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Close DB slowness/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Rename DB slowness/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Delete DB slowness/i })
    ).toBeInTheDocument();
  });

  it('omits New incident from the empty state for a non-admin', async () => {
    mockCanMutate = false;
    routeGet({ incidents: [] });

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
    routeGet({ incidents: [] });

    renderPage(<IncidentListPage />);

    await waitFor(() => {
      expect(screen.getByTestId('atw-incidents-empty-create')).toBeTruthy();
    });
  });

  it('renders no create, close, rename or delete for a non-admin', async () => {
    mockCanMutate = false;
    routeGet({ incidents: [incident] });

    renderPage(<IncidentListPage />);

    await waitFor(() => expect(screen.getByText('DB slowness')).toBeTruthy());
    expect(
      screen.queryByRole('button', { name: /New incident/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /Close DB slowness/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /Rename DB slowness/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /Delete DB slowness/i })
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
});
