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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FLEET_NOT_COLLECTED, OPERATIONS_DOC_URL } from '../src/constants';
import { FleetClustersTab } from '../src/FleetClustersTab';
import { cluster, mixedEstate, service, topology } from './fixtures';

const { useOmTopology, useOmInventoryServices } = vi.hoisted(() => ({
  useOmTopology: vi.fn(),
  useOmInventoryServices: vi.fn(),
}));

vi.mock('../src/topologyHooks', () => ({ useOmTopology }));
vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryServices,
  useLastScanFinishedAt: () => ({ status: 'ready', finishedAt: null }),
}));

const renderPage = () =>
  render(
    <MemoryRouter>
      <FleetClustersTab />
    </MemoryRouter>
  );

const clusterRows = () =>
  screen
    .getAllByRole('row')
    .filter(
      (row) =>
        within(row).queryByTestId('om-cluster-health') &&
        !within(row).queryByTestId('om-cluster-group')
    );

const groupHeaders = () => screen.queryAllByTestId('om-cluster-group');

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

    const members = screen
      .getAllByRole('row')
      // The detail panel's own row wraps the whole member table; skip it.
      .filter(
        (row) => within(row).queryAllByTestId('om-service-status').length === 1
      );
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

  describe('counts carry no status colour (Pedro, 2026-10-06 and 2026-10-07)', () => {
    // MUI's default palette, which these tests render under.
    const RED = 'rgb(211, 47, 47)';
    const GREEN = 'rgb(46, 125, 50)';
    const colourOf = (element: HTMLElement) => getComputedStyle(element).color;
    // The number is in a <strong> of its own, so match the line it sits in.
    const line = (text: string) =>
      screen.getAllByText((_, element) => element?.textContent === text)[0];

    it('keeps "up" neutral on the summary lines, and "down" red only above zero', () => {
      renderPage();

      for (const up of screen.getAllByText(/\bup$/)) {
        expect([RED, GREEN]).not.toContain(colourOf(up));
      }
      expect(colourOf(line('1 down'))).toBe(RED);
      expect(colourOf(line('0 down'))).not.toBe(RED);
    });

    it("states a cluster's members as up out of total, in plain text", () => {
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
              cluster({ name: 'empty', services: [] }),
              cluster({
                name: 'dark',
                services: [service({ service_name: 'dark-1', ...down })],
              }),
            ],
          },
        ]),
        isPending: false,
        isError: false,
      });
      renderPage();

      for (const text of ['1/2', '0/0', '0/1']) {
        expect([RED, GREEN]).not.toContain(colourOf(screen.getByText(text)));
      }
    });

    it('marks a group header "down" red only above zero', () => {
      useOmTopology.mockReturnValue({
        data: twoEnvironments(),
        isPending: false,
        isError: false,
      });
      renderPage();

      const [production] = groupHeaders();
      expect(colourOf(within(production).getByText('down'))).toBe(RED);
      expect(colourOf(within(production).getByText('up'))).not.toBe(RED);
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
      const [production, unnamed] = groupHeaders();
      expect(production).toHaveTextContent('production');
      expect(production).toHaveTextContent(
        /2 clusters.*6 services.*5 up.*1 down/
      );
      expect(unnamed).toHaveTextContent('Unassigned environment');
      // The grouped column does not repeat on every row.
      expect(
        screen.queryByRole('columnheader', { name: /^Environment/ })
      ).toBeNull();
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

      const headers = groupHeaders();
      expect(headers).toHaveLength(3);
      expect(headers[1]).toHaveTextContent(
        /Unassigned environment.*2 clusters/
      );
      expect(headers[2]).toHaveTextContent(
        /Unassigned environment.*1 cluster(?!s)/
      );
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

      groupBy('Health');
      expect(
        groupHeaders().map(
          (header) =>
            within(header).getByTestId('om-cluster-health').textContent
        )
      ).toEqual(['Down', 'Degraded', 'Healthy']);
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
      groupBy('Health');
      unmount();

      renderPage();

      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('Health');
      expect(
        groupHeaders().map(
          (header) =>
            within(header).getByTestId('om-cluster-health').textContent
        )
      ).toEqual(['Down', 'Degraded', 'Healthy']);
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
      render(<RouterProvider router={router} />);
      return router;
    };

    /** A filter's URL value, as peak-ui writes it. */
    const encoded = (values: string[]) =>
      encodeURIComponent(JSON.stringify(values));

    const search = () =>
      screen.getByRole('textbox', { name: 'Search clusters' });

    const pick = (filter: string, ...options: string[]) => {
      fireEvent.mouseDown(screen.getByRole('combobox', { name: filter }));
      const listbox = screen.getByRole('listbox');
      for (const option of options) {
        fireEvent.click(within(listbox).getByRole('option', { name: option }));
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

      fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));

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

      groupBy('Health');
      pick('Environment', 'production');

      await waitFor(() =>
        expect(query(router).has('clusters.f.env_name')).toBe(true)
      );
      expect(query(router).has('grouping')).toBe(false);
      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('Health');
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

    it('labels each cluster with its type beside its name', () => {
      renderPage();

      expect(
        within(rowOf('orders')).getByTestId('om-cluster-type')
      ).toHaveTextContent('Sharded');
      expect(
        within(rowOf('quiet')).getByTestId('om-cluster-type')
      ).toHaveTextContent('Replica set');
    });

    it('keeps Type, Last seen, Max CPU and Min conn. free in the chooser', () => {
      renderPage();

      for (const name of ['Type', 'Last seen', 'Max CPU', 'Min conn. free']) {
        expect(
          screen.queryByRole('columnheader', { name: new RegExp(`^${name}`) })
        ).toBeNull();
      }

      showColumn('Type');
      showColumn('Last seen');
      showColumn('Max CPU');
      showColumn('Min conn. free');

      // The cell, not the label beside the name, which says the same.
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
      showColumn('Versions');

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
      showColumn('Versions');

      fireEvent.click(
        within(
          screen.getByRole('columnheader', { name: /^Versions/ })
        ).getByText('Versions')
      );

      expect(
        clusterRows().map((row) =>
          row.textContent?.includes('older') ? 'older' : 'newer'
        )
      ).toEqual(['older', 'newer']);
    });
  });
});
