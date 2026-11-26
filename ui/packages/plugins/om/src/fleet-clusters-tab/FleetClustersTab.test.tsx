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

import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import {
  MemoryRouter,
  RouterProvider,
  createMemoryRouter,
} from 'react-router-dom';
import type { ReactNode } from 'react';
import { hexToRgb } from '@mui/material/styles';
import { ThemeContextProvider, pmmThemeOptions } from '@percona/peak-ui';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FLEET_NOT_COLLECTED, OPERATIONS_DOC_URL } from '../constants';
import { FleetClustersTab } from './FleetClustersTab';
import { cluster, mixedEstate, service, topology } from '../../tests/fixtures';

const { useOmTopology, useOmInventoryServices } = vi.hoisted(() => ({
  useOmTopology: vi.fn(),
  useOmInventoryServices: vi.fn(),
}));

vi.mock('../topologyHooks', () => ({ useOmTopology }));
vi.mock('../inventoryHooks', () => ({
  useOmInventoryServices,
  useLastScanFinishedAt: () => ({ status: 'ready', finishedAt: null }),
}));

/** The app's theme, which peak-ui's status icons read their colours from. */
const Theme = ({ children }: { children: ReactNode }) => (
  <ThemeContextProvider themeOptions={pmmThemeOptions}>
    {children}
  </ThemeContextProvider>
);

const renderPage = () =>
  render(
    <MemoryRouter>
      <FleetClustersTab />
    </MemoryRouter>,
    { wrapper: Theme }
  );

const clusterRows = () =>
  screen
    .getAllByRole('row')
    .filter(
      (row) =>
        within(row).queryByTestId('om-cluster-health') &&
        !within(row).queryByTestId('om-cluster-group')
    );

const memberRows = () =>
  screen
    .getAllByRole('row')
    // The detail panel's own row wraps the whole member table; skip it.
    .filter(
      (row) => within(row).queryAllByTestId('om-service-status').length === 1
    );

const groupHeaders = () => screen.queryAllByTestId('om-cluster-group');

/** Each group header's name and cluster count, as "name: count". */
const groupSummaries = () =>
  groupHeaders().map(
    (header) =>
      `${within(header).getByRole('heading').textContent}: ${
        within(header).getByTestId('om-cluster-group-count').textContent
      }`
  );

/** Turn a hidden column on from the chooser, the way a reader would. */
const showColumn = (name: string) => {
  fireEvent.click(screen.getByRole('button', { name: /show\/hide columns/i }));
  const item = screen
    .getAllByRole('menuitem')
    .find((menuItem) => menuItem.textContent === name) as HTMLElement;
  fireEvent.click(item.querySelector('input') as HTMLInputElement);
  fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' });
};

const groupBy = (label: string) => {
  fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Group by' }));
  fireEvent.click(screen.getByRole('option', { name: label }));
};

const down = { status: 'SERVICE_STATUS_DOWN' } as const;

/** Two environments, one unnamed, each with a down-or-degraded cluster named last. */
const twoEnvironments = () =>
  topology([
    {
      env_name: 'production',
      clusters: [
        cluster({
          name: 'alpha',
          services: [
            service({ service_name: 'alpha-1' }),
            service({ service_name: 'alpha-2' }),
            service({ service_name: 'alpha-3' }),
          ],
        }),
        cluster({
          name: 'zulu',
          services: [
            service({ service_name: 'zulu-1' }),
            service({ service_name: 'zulu-2', ...down }),
            service({ service_name: 'zulu-3' }),
          ],
        }),
      ],
    },
    {
      env_name: null,
      clusters: [
        cluster({
          name: 'bravo',
          services: [service({ service_name: 'bravo-1' })],
        }),
        cluster({
          name: 'yankee',
          services: [service({ service_name: 'yankee-1', ...down })],
        }),
      ],
    },
  ]);

const clusterNames = () =>
  clusterRows().map(
    (row) =>
      ['alpha', 'zulu', 'bravo', 'yankee'].find((name) =>
        row.textContent?.includes(name)
      ) ?? '?'
  );

describe('FleetClustersTab', () => {
  beforeEach(() => {
    useOmTopology.mockReturnValue({
      data: mixedEstate(),
      isPending: false,
      isError: false,
    });
    useOmInventoryServices.mockReturnValue({
      data: [],
      isPending: false,
      isError: false,
      error: null,
    });
  });

  it('warns, with the server error, when the scan results cannot be read', () => {
    useOmInventoryServices.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
      error: new Error('upstream connect error: PMM Extensions unavailable'),
    });
    renderPage();

    const alert = screen.getByRole('alert');
    expect(alert).toHaveTextContent('Could not read scan results');
    expect(alert).toHaveTextContent(
      'upstream connect error: PMM Extensions unavailable'
    );
    // The fleet itself is PMM's own data and still renders.
    expect(clusterRows().length).toBeGreaterThan(0);
  });

  it('shows no scan warning while the scan results read fine', () => {
    renderPage();

    expect(screen.queryByText('Could not read scan results')).toBeNull();
  });

  it('links an empty fleet to the Operations for MongoDB guide', () => {
    useOmTopology.mockReturnValue({
      data: topology([]),
      isPending: false,
      isError: false,
    });
    renderPage();

    expect(
      within(screen.getByTestId('om-empty-state')).getByRole('link', {
        name: 'Read about Operations for MongoDB',
      })
    ).toHaveAttribute('href', OPERATIONS_DOC_URL);
  });

  it('names each cluster state and leads with the degraded one', () => {
    renderPage();

    const [first, second] = clusterRows();
    expect(within(first).getByTestId('om-cluster-health')).toHaveTextContent(
      'Degraded'
    );
    expect(first).toHaveTextContent('orders');
    expect(within(second).getByTestId('om-cluster-health')).toHaveTextContent(
      'Healthy'
    );
  });

  it('labels the process type "Process", never "Role"', () => {
    renderPage();
    showColumn('Process');

    expect(
      screen.getByRole('columnheader', { name: /Process/ })
    ).toBeInTheDocument();
    expect(screen.queryByRole('columnheader', { name: /Role/ })).toBeNull();
  });

  it('lists an unfolded cluster down member first, with member state as text', () => {
    renderPage();

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );

    const members = memberRows();
    expect(
      members.map((row) => within(row).getByRole('link').textContent)
    ).toEqual(['orders-2', 'orders-1', 'orders-3', 'orders-router']);
    expect(members[0]).toHaveTextContent('SECONDARY');
    expect(members[1]).toHaveTextContent('PRIMARY');
    // A router is no replica-set member: its process says so, not a state.
    expect(members[3]).toHaveTextContent('Router');
    expect(members[3]).not.toHaveTextContent(/PRIMARY|SECONDARY/);
    expect(
      screen.getByRole('columnheader', { name: 'Member state' })
    ).toBeInTheDocument();
  });

  it('says beside a down member how long it has been down', () => {
    useOmTopology.mockReturnValue({
      data: topology([
        {
          env_name: 'production',
          clusters: [
            cluster({
              name: 'orders',
              services: [
                service({ service_name: 'orders-1', state: 'PRIMARY' }),
                service({
                  service_name: 'orders-2',
                  status: 'SERVICE_STATUS_DOWN',
                  last_up_at: new Date(
                    Date.now() - 2 * 3600 * 1000
                  ).toISOString(),
                }),
              ],
            }),
          ],
        },
      ]),
      isPending: false,
      isError: false,
    });
    renderPage();

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );

    const down = screen
      .getAllByRole('row')
      .find((row) => within(row).queryByText('orders-2')) as HTMLElement;
    expect(within(down).getByTestId('om-down-for')).toHaveTextContent(
      /^for 2h( \d+s)?$/
    );
  });

  it('names an arbiter in its member row and in the cluster process summary', () => {
    useOmTopology.mockReturnValue({
      data: topology([
        {
          env_name: 'production',
          clusters: [
            cluster({
              name: 'orders',
              services: [
                service({ service_name: 'orders-1', state: 'PRIMARY' }),
                service({
                  service_name: 'orders-arbiter',
                  state: 'ARBITER',
                  process_role: 'PROCESS_ROLE_ARBITER',
                }),
              ],
            }),
          ],
        },
      ]),
      isPending: false,
      isError: false,
    });
    renderPage();
    showColumn('Process');

    expect(clusterRows()[0]).toHaveTextContent('1 mongod · 1 Arbiter');

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );
    const arbiter = memberRows().find((row) =>
      within(row).queryByText('orders-arbiter')
    ) as HTMLElement;
    expect(arbiter).toHaveTextContent('ARBITER');
    expect(arbiter).toHaveTextContent('Arbiter');
  });

  it('links each member to its PMM dashboard, a down one included', () => {
    renderPage();

    fireEvent.click(
      within(clusterRows()[0]).getByRole('button', { name: /expand/i })
    );

    expect(screen.getByRole('link', { name: 'orders-2' })).toHaveAttribute(
      'href',
      expect.stringContaining('var-service_name=orders-2')
    );
  });

  it('blames PMM for an empty fleet only once the snapshot has been built', () => {
    useOmTopology.mockReturnValue({
      data: topology([]),
      isPending: false,
      isError: false,
    });
    renderPage();

    expect(screen.getByTestId('om-empty-state')).toHaveTextContent(
      'PMM has no MongoDB services registered yet'
    );
  });

  it('says the fleet has not been read yet before the first snapshot', () => {
    const cold = topology([]);
    cold.snapshot.generated_at = undefined;
    useOmTopology.mockReturnValue({
      data: cold,
      isPending: false,
      isError: false,
    });
    renderPage();

    const empty = screen.getByTestId('om-empty-state');
    expect(empty).toHaveTextContent(FLEET_NOT_COLLECTED);
    expect(empty).not.toHaveTextContent('PMM has no MongoDB services');
  });

  describe('members (PMM-15878)', () => {
    const { error } = pmmThemeOptions('light').palette as Record<
      'error',
      { main: string }
    >;
    const RED = hexToRgb(error.main);
    const colourOf = (element: HTMLElement) => getComputedStyle(element).color;
    const silent = { status: 'SERVICE_STATUS_UNSPECIFIED' } as const;

    beforeEach(() => {
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'mixed',
                services: [
                  service({ service_name: 'mixed-1' }),
                  service({ service_name: 'mixed-2', ...down }),
                ],
              }),
              cluster({
                name: 'calm',
                services: [
                  service({ service_name: 'calm-1' }),
                  service({ service_name: 'calm-2' }),
                ],
              }),
              cluster({ name: 'empty', services: [] }),
              cluster({
                name: 'quiet',
                services: [
                  service({ service_name: 'quiet-1', ...silent }),
                  service({ service_name: 'quiet-2', ...silent }),
                ],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
    });

    it('marks the count red while a member is down', () => {
      renderPage();

      expect(colourOf(screen.getByText('1/2'))).toBe(RED);
      expect(colourOf(screen.getByText('2/2'))).not.toBe(RED);
      expect(colourOf(screen.getByText('0/0'))).not.toBe(RED);
    });

    it('shows ? for up while no member has reported', async () => {
      renderPage();

      const unreported = screen.getByText('?/2');
      fireEvent.mouseOver(unreported);
      expect(await screen.findByRole('tooltip')).toHaveTextContent(
        'No member has reported in this snapshot'
      );
    });
  });

  describe('one table, grouped by a visible control (PMM-15712)', () => {
    beforeEach(() => {
      useOmTopology.mockReturnValue({
        data: twoEnvironments(),
        isPending: false,
        isError: false,
      });
    });

    it('renders every environment in one table, grouped by environment', () => {
      renderPage();

      expect(screen.getAllByRole('table')).toHaveLength(1);
      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('Environment');
      expect(groupSummaries()).toEqual([
        'production: 2',
        'Unassigned environment: 2',
      ]);
      // The grouped column keeps its place, as in the design.
      expect(
        screen.getByRole('columnheader', { name: /^Environment/ })
      ).toBeInTheDocument();
    });

    it('keeps an environment named like the fallback apart from the unnamed one', () => {
      useOmTopology.mockReturnValue({
        data: topology([
          ...twoEnvironments().environments,
          {
            env_name: 'Unassigned environment',
            clusters: [
              cluster({
                name: 'xray',
                services: [service({ service_name: 'xray-1' })],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
      renderPage();

      // Named ones alphabetically, the unnamed one last.
      expect(groupSummaries()).toEqual([
        'production: 2',
        'Unassigned environment: 1',
        'Unassigned environment: 2',
      ]);
    });

    it('labels an empty environment name as unassigned', () => {
      const [named, unnamed] = twoEnvironments().environments;
      useOmTopology.mockReturnValue({
        data: topology([named, { ...unnamed, env_name: '' }]),
        isPending: false,
        isError: false,
      });
      renderPage();

      expect(groupSummaries()).toEqual([
        'production: 2',
        'Unassigned environment: 2',
      ]);
      expect(
        screen.getAllByRole('cell', { name: 'Unassigned environment' })
      ).toHaveLength(2);
    });

    it('leads each group with its down or degraded cluster', () => {
      renderPage();

      expect(clusterNames()).toEqual(['zulu', 'alpha', 'yankee', 'bravo']);
    });

    it('folds a group away, keeping its header', () => {
      renderPage();

      const toggle = within(groupHeaders()[0]).getByRole('button', {
        name: 'Collapse production',
      });
      expect(toggle).toHaveAttribute('aria-expanded', 'true');
      fireEvent.click(toggle);

      expect(clusterNames()).toEqual(['yankee', 'bravo']);
      expect(
        within(groupHeaders()[0]).getByRole('button', {
          name: 'Expand production',
        })
      ).toHaveAttribute('aria-expanded', 'false');
      expect(groupHeaders()).toHaveLength(2);
    });

    it('regroups by health, worst first, and flattens on None', () => {
      renderPage();

      groupBy('Health status');
      expect(groupSummaries()).toEqual([
        'Down: 1',
        'Degraded: 1',
        'Healthy: 2',
      ]);
      expect(
        screen.getByRole('columnheader', { name: /^Environment/ })
      ).toBeInTheDocument();

      groupBy('None');
      expect(groupHeaders()).toHaveLength(0);
      expect(clusterNames()).toEqual(['yankee', 'zulu', 'alpha', 'bravo']);
    });

    it('reads Members as 2/3, with no Up, Down or Process column by default', () => {
      renderPage();

      expect(
        screen.getByRole('columnheader', { name: /^Members/ })
      ).toBeInTheDocument();
      expect(within(clusterRows()[0]).getByText('2/3')).toBeInTheDocument();
      for (const name of ['Up', 'Down', 'Process']) {
        expect(
          screen.queryByRole('columnheader', { name: new RegExp(`^${name}`) })
        ).toBeNull();
      }

      showColumn('Process');
      expect(
        screen.getByRole('columnheader', { name: /^Process/ })
      ).toBeInTheDocument();
    });

    it('has no drag-to-group handle on any column header', () => {
      renderPage();

      for (const header of screen.getAllByRole('columnheader')) {
        expect(
          within(header).queryByRole('button', { name: /move/i })
        ).toBeNull();
      }
    });
  });

  describe('Group by kept for the session (PMM-15767)', () => {
    beforeEach(() => {
      useOmTopology.mockReturnValue({
        data: twoEnvironments(),
        isPending: false,
        isError: false,
      });
    });

    it('reopens grouped as the reader left it, e.g. after a tab switch', () => {
      const { unmount } = renderPage();
      groupBy('Health status');
      unmount();

      renderPage();

      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('Health');
      expect(groupSummaries()).toEqual([
        'Down: 1',
        'Degraded: 1',
        'Healthy: 2',
      ]);
    });

    it('opens by environment when the stored grouping is not an option', () => {
      sessionStorage.setItem('om.fleet.clusters.grouping', 'cluster_id');

      renderPage();

      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('Environment');
    });
  });

  describe('search and filters, kept in the URL (PMM-15806)', () => {
    beforeEach(() => {
      useOmTopology.mockReturnValue({
        data: twoEnvironments(),
        isPending: false,
        isError: false,
      });
    });

    /** On a real router, so the URL and its history can be read and walked. */
    const renderAt = (url: string) => {
      const router = createMemoryRouter(
        [{ path: '/fleet', element: <FleetClustersTab /> }],
        { initialEntries: [url] }
      );
      render(<RouterProvider router={router} />, { wrapper: Theme });
      return router;
    };

    /** A filter's URL value, as peak-ui writes it. */
    const encoded = (values: string[]) =>
      encodeURIComponent(JSON.stringify(values));

    const search = () =>
      screen.getByRole('textbox', { name: 'Filter clusters by name' });

    const pick = (filter: string, ...options: string[]) => {
      fireEvent.mouseDown(screen.getByRole('combobox', { name: filter }));
      const listbox = screen.getByRole('listbox');
      for (const option of options) {
        // An option's name ends in its count: "Down · 1".
        fireEvent.click(
          within(listbox).getByRole('option', {
            name: new RegExp(`^${option} · \\d+$`),
          })
        );
      }
      fireEvent.keyDown(listbox, { key: 'Escape' });
    };

    const query = (router: ReturnType<typeof renderAt>) =>
      new URLSearchParams(router.state.location.search);

    it('searches cluster names case-insensitively, keeping the tab', async () => {
      const router = renderAt('/fleet?tab=clusters');

      fireEvent.change(search(), { target: { value: 'ZU' } });

      await waitFor(() => expect(clusterNames()).toEqual(['zulu']));
      await waitFor(() =>
        expect(query(router).toString()).toBe('tab=clusters&clusters.q=ZU')
      );
      expect(screen.getByTestId('om-cluster-match-count')).toHaveTextContent(
        '1 of 4 clusters'
      );
    });

    it('searches the cluster name alone', async () => {
      renderAt('/fleet');

      // "production" is an environment, and "Healthy" a health: neither is a name.
      fireEvent.change(search(), { target: { value: 'production' } });
      await waitFor(() => expect(clusterNames()).toEqual([]));

      fireEvent.change(search(), { target: { value: 'healthy' } });
      await waitFor(() => expect(clusterNames()).toEqual([]));
    });

    it('filters by health, several values ORed', async () => {
      const router = renderAt('/fleet');

      pick('Health', 'Down', 'Degraded');

      expect(clusterNames()).toEqual(['zulu', 'yankee']);
      await waitFor(() =>
        expect(query(router).get('clusters.f.health')).toBe(
          '["down","degraded"]'
        )
      );
    });

    it('filters by type', async () => {
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'alpha',
                type: 'CLUSTER_TYPE_SHARDED',
                services: [service({ service_name: 'alpha-1' })],
              }),
              cluster({
                name: 'bravo',
                services: [service({ service_name: 'bravo-1' })],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
      const router = renderAt('/fleet');

      pick('Type', 'Sharded');

      expect(clusterNames()).toEqual(['alpha']);
      await waitFor(() =>
        expect(query(router).get('clusters.f.type')).toBe('["sharded"]')
      );
    });

    it('filters by environment, the unnamed one included', async () => {
      const router = renderAt('/fleet');

      pick('Environment', 'Unassigned environment');

      expect(clusterNames()).toEqual(['yankee', 'bravo']);
      // Not '', which peak-ui would drop as no filter at all.
      await waitFor(() =>
        expect(query(router).get('clusters.f.env_name')).toBe('["(none)"]')
      );
    });

    it('matches an environment by its whole name', () => {
      useOmTopology.mockReturnValue({
        data: topology(
          ['prod', 'preprod'].map((env_name) => ({
            env_name,
            clusters: [
              cluster({
                name: `${env_name}-db`,
                services: [service({ service_name: `${env_name}-1` })],
              }),
            ],
          }))
        ),
        isPending: false,
        isError: false,
      });
      renderAt('/fleet');

      pick('Environment', 'prod');

      expect(
        clusterRows().map((row) =>
          row.textContent?.includes('preprod-db') ? 'preprod' : 'prod'
        )
      ).toEqual(['prod']);
    });

    it('ANDs search and filters', async () => {
      renderAt(
        `/fleet?clusters.f.env_name=${encoded(['production'])}&clusters.f.health=${encoded(['healthy', 'down'])}`
      );

      expect(clusterNames()).toEqual(['alpha']);

      fireEvent.change(search(), { target: { value: 'b' } });
      await waitFor(() => expect(clusterNames()).toEqual([]));
    });

    it('opens a shared link filtered, showing a value the fleet lacks so it can be cleared', () => {
      const router = renderAt(
        `/fleet?tab=clusters&clusters.q=a&clusters.f.env_name=${encoded(['staging'])}`
      );

      expect(search()).toHaveValue('a');
      expect(
        screen.getByRole('combobox', { name: 'Environment' })
      ).toHaveTextContent('staging');
      expect(clusterNames()).toEqual([]);
      expect(
        screen.getByText('No cluster matches the search and filters.')
      ).toBeInTheDocument();

      fireEvent.click(screen.getByRole('button', { name: 'Clear all' }));

      expect(clusterNames()).toHaveLength(4);
      expect(search()).toHaveValue('');
      return waitFor(() =>
        expect(query(router).toString()).toBe('tab=clusters')
      );
    });

    it('replaces the history entry rather than adding one', async () => {
      const router = renderAt('/fleet');

      pick('Health', 'Down');

      await waitFor(() =>
        expect(query(router).has('clusters.f.health')).toBe(true)
      );
      expect(router.state.historyAction).toBe('REPLACE');
    });

    it('follows the URL on Back', async () => {
      const router = renderAt('/fleet');
      await act(async () => {
        await router.navigate(`/fleet?clusters.f.health=${encoded(['down'])}`);
      });
      expect(clusterNames()).toEqual(['yankee']);

      await act(async () => {
        await router.navigate(-1);
      });
      expect(clusterNames()).toHaveLength(4);
      expect(
        screen.getByRole('combobox', { name: 'Health' })
      ).toHaveTextContent('Any');
    });

    it('leaves the default sort out of the URL', async () => {
      const router = renderAt('/fleet');

      pick('Health', 'Down');

      await waitFor(() =>
        expect(query(router).has('clusters.f.health')).toBe(true)
      );
      expect(query(router).has('clusters.sort')).toBe(false);
    });

    it('clears the search from its own button without writing "undefined"', async () => {
      const router = renderAt('/fleet?clusters.q=zu');
      expect(clusterNames()).toEqual(['zulu']);

      fireEvent.click(screen.getByRole('button', { name: 'Clear search' }));

      await waitFor(() => expect(clusterNames()).toHaveLength(4));
      await waitFor(() => expect(query(router).toString()).toBe(''));
    });

    it('keeps Group by out of the URL and as it was', async () => {
      const router = renderAt('/fleet');

      groupBy('Health status');
      pick('Environment', 'production');

      await waitFor(() =>
        expect(query(router).has('clusters.f.env_name')).toBe(true)
      );
      expect(query(router).has('grouping')).toBe(false);
      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('Health');
    });

    describe('filter card and Group by bar (PMM-15878)', () => {
      /** Three clusters, one of each type, on two versions and a mix of them. */
      const threeTypes = () =>
        topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'standalone',
                type: 'CLUSTER_TYPE_STANDALONE',
                services: [
                  service({ service_name: 's-1', version: '7.0.9-1' }),
                  service({ service_name: 's-2', version: '8.0.4-1' }),
                ],
              }),
              cluster({
                name: 'replica',
                services: [
                  service({ service_name: 'r-1', version: '7.0.9-1' }),
                ],
              }),
              cluster({
                name: 'sharded',
                type: 'CLUSTER_TYPE_SHARDED',
                services: [
                  service({ service_name: 'h-1', version: '8.0.4-1' }),
                ],
              }),
            ],
          },
        ]);

      const optionNames = (filter: string) => {
        fireEvent.mouseDown(screen.getByRole('combobox', { name: filter }));
        const names = within(screen.getByRole('listbox'))
          .getAllByRole('option')
          .map((option) => option.textContent);
        fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape' });
        return names;
      };

      it('groups by environment when there are several, else by health status', () => {
        renderAt('/fleet');
        expect(
          screen.getByRole('combobox', { name: 'Group by' })
        ).toHaveTextContent('Environment');
        cleanup();
        sessionStorage.clear();

        useOmTopology.mockReturnValue({
          data: threeTypes(),
          isPending: false,
          isError: false,
        });
        renderAt('/fleet');
        expect(
          screen.getByRole('combobox', { name: 'Group by' })
        ).toHaveTextContent('Health status');
      });

      it('orders type groups sharded first, and version groups newest first with the mixed last', () => {
        useOmTopology.mockReturnValue({
          data: threeTypes(),
          isPending: false,
          isError: false,
        });
        renderAt('/fleet');

        groupBy('Type');
        expect(groupSummaries()).toEqual([
          'Sharded: 1',
          'Replica set: 1',
          'Standalone: 1',
        ]);

        groupBy('Version');
        expect(groupSummaries()).toEqual([
          '8.0.4-1: 1',
          '7.0.9-1: 1',
          'Mixed versions: 1',
        ]);
      });

      it('lists the Type options in the order Group by draws the type groups', () => {
        useOmTopology.mockReturnValue({
          data: threeTypes(),
          isPending: false,
          isError: false,
        });
        renderAt('/fleet');

        expect(optionNames('Type')).toEqual([
          'Sharded · 1',
          'Replica set · 1',
          'Standalone · 1',
        ]);
      });

      it('filters by version, a mixed cluster matching each of its versions', async () => {
        useOmTopology.mockReturnValue({
          data: threeTypes(),
          isPending: false,
          isError: false,
        });
        const router = renderAt('/fleet');

        expect(optionNames('Version')).toEqual(['8.0.4-1 · 2', '7.0.9-1 · 2']);
        pick('Version', '8.0.4-1');

        expect(
          clusterRows().map((row) =>
            row.textContent?.includes('sharded') ? 'sharded' : 'standalone'
          )
        ).toEqual(['sharded', 'standalone']);
        await waitFor(() =>
          expect(query(router).get('clusters.f.versions')).toBe('["8.0.4-1"]')
        );
      });

      it('counts each option under the other filters, a zero kept to pick', () => {
        renderAt('/fleet');

        expect(optionNames('Health')).toEqual([
          'Down · 1',
          'Degraded · 1',
          'Unknown · 0',
          'Healthy · 2',
        ]);
        pick('Environment', 'production');
        expect(optionNames('Health')).toEqual([
          'Down · 0',
          'Degraded · 1',
          'Unknown · 0',
          'Healthy · 1',
        ]);
        // Its own pick does not narrow its counts.
        pick('Health', 'Degraded');
        expect(optionNames('Health')).toEqual([
          'Down · 0',
          'Degraded · 1',
          'Unknown · 0',
          'Healthy · 1',
        ]);
      });

      it('lists the unnamed environment after the named ones, as Group by does', () => {
        renderAt('/fleet');

        expect(optionNames('Environment')).toEqual([
          'production · 2',
          'Unassigned environment · 2',
        ]);
      });

      it('offers Clear all only while a search or filter is on', () => {
        renderAt('/fleet');
        expect(screen.queryByRole('button', { name: 'Clear all' })).toBeNull();

        fireEvent.change(search(), { target: { value: 'a' } });

        expect(
          screen.getByRole('button', { name: 'Clear all' })
        ).toBeInTheDocument();
      });

      it('counts the groups and the matching clusters, and says how fresh they are', () => {
        useOmTopology.mockReturnValue({
          data: twoEnvironments(),
          isPending: false,
          isError: false,
          dataUpdatedAt: Date.now() - 5000,
        });
        renderAt('/fleet');

        expect(screen.getByText(/^2 groups ·/)).toHaveTextContent(
          '2 groups · 4 clusters'
        );
        expect(screen.getByTestId('om-cluster-refreshed')).toHaveTextContent(
          'Refreshed 5s ago'
        );

        pick('Health', 'Down');
        expect(screen.getByText(/^1 group ·/)).toHaveTextContent(
          '1 group · 1 of 4 clusters'
        );
      });
    });
  });

  describe('type, freshness and load columns (PMM-15806)', () => {
    const observed = (minutes: number) =>
      new Date(Date.now() - minutes * 60 * 1000).toISOString();

    beforeEach(() => {
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'orders',
                type: 'CLUSTER_TYPE_SHARDED',
                services: [
                  service({
                    service_name: 'orders-1',
                    replication_set: 'shard0',
                    version: '7.0.24-13',
                    observed_at: observed(1),
                    cpu_usage_percent: 40,
                    connections_free_percent: 70,
                  }),
                  service({
                    service_name: 'orders-2',
                    replication_set: 'shard0',
                    version: '7.0.9-1',
                    observed_at: observed(15),
                    cpu_usage_percent: 85.5,
                    connections_free_percent: -1,
                  }),
                ],
              }),
              cluster({
                name: 'quiet',
                services: [
                  service({
                    service_name: 'quiet-1',
                    observed_at: null,
                    cpu_usage_percent: -1,
                    connections_free_percent: -1,
                  }),
                ],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
    });

    const rowOf = (name: string) =>
      clusterRows().find((row) =>
        row.textContent?.includes(name)
      ) as HTMLElement;

    it("shows each cluster's type in its own column", () => {
      renderPage();

      expect(
        within(rowOf('orders')).getByTestId('om-cluster-type')
      ).toHaveTextContent('Sharded');
      expect(
        within(rowOf('quiet')).getByTestId('om-cluster-type')
      ).toHaveTextContent('Replica set');
    });

    it("opens in the ticket's column order, the rest in the chooser", () => {
      renderPage();

      expect(
        screen
          .getAllByRole('columnheader')
          // Less the sort-order badge MRT appends to a sortable header.
          .map((header) => header.textContent?.replace(/\d+$/, ''))
          .filter(Boolean)
      ).toEqual([
        'Cluster name',
        'Type',
        'Members',
        'Environment',
        'Health',
        'Version',
        'Last seen',
      ]);

      showColumn('Max CPU');
      showColumn('Min conn. free');

      const typeColumn = screen
        .getAllByRole('columnheader')
        .findIndex((header) => header.textContent?.startsWith('Type'));
      expect(
        within(rowOf('orders')).getAllByRole('cell')[typeColumn]
      ).toHaveTextContent(/^Sharded$/);
      expect(
        within(rowOf('quiet')).getAllByRole('cell')[typeColumn]
      ).toHaveTextContent(/^Replica set$/);

      const orders = rowOf('orders');
      // A slow runner can tick past the minute between fixture and render.
      expect(orders).toHaveTextContent(/15m( \d+s)? ago/);
      expect(orders).toHaveTextContent('85.5%');
      expect(orders).toHaveTextContent('70.0%');
      // Nobody reporting is a dash with its reason, never a zero.
      const quiet = rowOf('quiet');
      expect(quiet).not.toHaveTextContent('%');
      expect(within(quiet).getAllByText('—').length).toBeGreaterThanOrEqual(3);
    });

    it('reads drift as Mixed, listing each set and its versions', async () => {
      renderPage();

      fireEvent.mouseOver(within(rowOf('orders')).getByText('Mixed'));

      expect(await screen.findByTestId('om-version-sets')).toHaveTextContent(
        'shard0: 7.0.9-1, 7.0.24-13'
      );
    });

    it('sorts Versions as versions, not as strings', () => {
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'newer',
                services: [service({ version: '7.0.24-13' })],
              }),
              cluster({
                name: 'older',
                services: [service({ version: '7.0.9-1' })],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
      renderPage();

      fireEvent.click(
        within(
          screen.getByRole('columnheader', { name: /^Version/ })
        ).getByText('Version')
      );

      expect(
        clusterRows().map((row) =>
          row.textContent?.includes('older') ? 'older' : 'newer'
        )
      ).toEqual(['older', 'newer']);
    });

    it('sorts Members by the share of members up, clusters with none first', () => {
      const members = (name: string, up: number, total: number) =>
        cluster({
          name,
          services: Array.from({ length: total }, (_, index) =>
            service({
              service_name: `${name}-${index + 1}`,
              ...(index < up ? {} : down),
            })
          ),
        });
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              members('alpha', 1, 1),
              members('bravo', 1, 5),
              members('charlie', 0, 3),
              members('delta', 0, 0),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
      renderPage();
      groupBy('None');

      fireEvent.click(
        within(
          screen.getByRole('columnheader', { name: /^Members/ })
        ).getByText('Members')
      );

      const names = ['alpha', 'bravo', 'charlie', 'delta'];
      expect(
        clusterRows().map((row) =>
          names.find((name) => row.textContent?.includes(name))
        )
      ).toEqual(['delta', 'charlie', 'bravo', 'alpha']);
    });
  });

  describe('health reasons and threshold cues (PMM-15878)', () => {
    beforeEach(() => {
      useOmTopology.mockReturnValue({
        data: topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'lagging',
                services: [
                  service({
                    service_name: 'lagging-1',
                    replication_lag_seconds: 47,
                    oplog_window_seconds: 52 * 60,
                  }),
                  service({ service_name: 'lagging-2', ...down }),
                ],
              }),
              cluster({
                name: 'calm',
                services: [
                  service({
                    service_name: 'calm-1',
                    replication_lag_seconds: 2,
                    oplog_window_seconds: 9 * 60 * 60,
                  }),
                ],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
    });

    const rowNamed = (name: string) =>
      clusterRows().find((row) =>
        row.textContent?.includes(name)
      ) as HTMLElement;

    it('says under the badge which member is down', () => {
      renderPage();

      expect(
        within(rowNamed('lagging')).getByTestId('om-cluster-health-reason')
      ).toHaveTextContent('lagging-2 down');
      expect(
        within(rowNamed('calm')).queryByTestId('om-cluster-health-reason')
      ).toBeNull();
    });

    it('marks lag over 30s and an oplog window under 1h with an icon and a tooltip', async () => {
      renderPage();
      showColumn('Max repl. lag');
      showColumn('Oplog window');

      const lag = within(rowNamed('lagging')).getByTestId('om-threshold-error');
      expect(lag).toHaveTextContent('47s');
      expect(within(lag).getByTestId('om-threshold-icon-error')).toBeTruthy();
      const oplog = within(rowNamed('lagging')).getByTestId(
        'om-threshold-warning'
      );
      expect(oplog).toHaveTextContent('52m');
      expect(
        within(oplog).getByTestId('om-threshold-icon-warning')
      ).toBeTruthy();
      expect(
        within(rowNamed('calm')).queryByTestId(/om-threshold-/)
      ).toBeNull();

      fireEvent.mouseOver(lag);
      expect(await screen.findByRole('tooltip')).toHaveTextContent(
        'Above 30s, the alert threshold'
      );
    });
  });
});
