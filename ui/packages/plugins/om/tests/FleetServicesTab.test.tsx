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
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { browserTimezone } from '@pmm-extensions/framework';
import { FleetServicesTab } from '../src/FleetServicesTab';
import {
  FLEET_NOT_COLLECTED,
  OPERATIONS_DOC_URL,
  SERVICE_SCAN_ERROR_KIND,
} from '../src/constants';
import type {
  OmInventoryService,
  OmServiceStatus,
  OmTopologyResponse,
} from '../src/types';
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

const scanned = (failingSince: string | null) =>
  useOmInventoryServices.mockReturnValue({
    data: [
      {
        service_id: 's1',
        node_id: 'n1',
        freshness: { failing_since: failingSince, consecutive_failures: 1 },
      } as OmInventoryService,
    ],
    isPending: false,
    isError: false,
  });

const serve = (data: OmTopologyResponse) =>
  useOmTopology.mockReturnValue({ data, isPending: false, isError: false });

const ordersEstate = (
  orders1Status: OmServiceStatus | null = 'SERVICE_STATUS_UP'
) =>
  topology([
    {
      env_name: 'production',
      clusters: [
        cluster({
          name: 'orders',
          services: [
            ...(orders1Status
              ? [
                  service({
                    service_name: 'orders-1',
                    service_id: 's1',
                    status: orders1Status,
                  }),
                ]
              : []),
            service({ service_name: 'orders-2', service_id: 's2' }),
          ],
        }),
      ],
    },
  ]);

const renderTab = () =>
  render(
    <MemoryRouter>
      <FleetServicesTab />
    </MemoryRouter>
  );

const rowFor = (name: string) =>
  screen
    .getAllByRole('row')
    .find((row) => within(row).queryByText(name)) as HTMLElement;

describe('FleetServicesTab', () => {
  beforeEach(() => {
    serve(mixedEstate());
    useOmInventoryServices.mockReturnValue({
      data: [],
      isPending: false,
      isError: false,
    });
  });

  it('opens with the down service as the first row', () => {
    render(
      <MemoryRouter>
        <FleetServicesTab />
      </MemoryRouter>
    );

    const rows = screen
      .getAllByRole('row')
      .filter((row) => within(row).queryByTestId('om-service-status'));
    expect(within(rows[0]).getByRole('link')).toHaveTextContent('orders-2');
    expect(within(rows[0]).getByTestId('om-service-status')).toHaveTextContent(
      'Down'
    );
  });

  it('labels the process type "Process", never "Role"', () => {
    render(
      <MemoryRouter>
        <FleetServicesTab />
      </MemoryRouter>
    );

    // The column budget hides Process by default (PMM-15659), so the label is read
    // where a hidden column is still named: the column chooser.
    fireEvent.click(screen.getByRole('button', { name: 'Show/Hide columns' }));
    const chooser = screen.getByRole('menu');

    expect(within(chooser).getByText('Process')).toBeInTheDocument();
    expect(within(chooser).queryByText(/^Role$/)).toBeNull();
    expect(screen.queryByRole('columnheader', { name: /^Role/ })).toBeNull();
  });

  it('opens the drawer from a row, but not from the service link in it', () => {
    serve(ordersEstate());
    renderTab();

    fireEvent.click(within(rowFor('orders-1')).getByRole('link'));
    expect(screen.queryByRole('dialog')).toBeNull();

    fireEvent.click(
      within(rowFor('orders-1')).getByTestId('om-service-status')
    );
    expect(screen.getByRole('dialog')).toHaveAccessibleName('orders-1');
  });

  it('keeps the drawer on the live row, and closes it when the service goes', async () => {
    serve(ordersEstate());
    const { rerender } = renderTab();
    fireEvent.click(
      within(rowFor('orders-1')).getByTestId('om-service-status')
    );
    expect(
      within(screen.getByRole('dialog')).getByTestId('om-service-status')
    ).toHaveTextContent('Up');

    serve(ordersEstate('SERVICE_STATUS_DOWN'));
    rerender(
      <MemoryRouter>
        <FleetServicesTab />
      </MemoryRouter>
    );
    expect(
      within(screen.getByRole('dialog')).getByTestId('om-service-status')
    ).toHaveTextContent('Down');

    serve(ordersEstate(null));
    rerender(
      <MemoryRouter>
        <FleetServicesTab />
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });

  it('says the fleet has not been read yet before the first snapshot', () => {
    const cold = topology([]);
    cold.snapshot.generated_at = undefined;
    serve(cold);
    renderTab();

    const empty = screen.getByTestId('om-empty-state');
    expect(empty).toHaveTextContent(FLEET_NOT_COLLECTED);
    expect(empty).not.toHaveTextContent('PMM has no MongoDB services');
  });

  it('blames the filter, not PMM, when the failing filter empties the table', () => {
    serve(ordersEstate());
    scanned('2026-10-06T09:00:00Z');
    const { rerender } = renderTab();
    fireEvent.click(screen.getByText('1 failing a scan'));

    scanned(null);
    rerender(
      <MemoryRouter>
        <FleetServicesTab />
      </MemoryRouter>
    );

    const empty = screen.getByTestId('om-empty-state');
    expect(empty).toHaveTextContent('No service is failing a scan right now');
    expect(empty).not.toHaveTextContent('PMM has no MongoDB services');
  });

  it('puts the local collection time and zone on the Collected hover', () => {
    const collected = new Date(Date.now() - 3 * 60 * 1000).toISOString();
    // Every fixture service shares service_id 30, so this one row of inventory joins
    // onto each of them.
    useOmInventoryServices.mockReturnValue({
      data: [
        {
          service_id: '30',
          node_id: 'node-1',
          observed: {},
          freshness: { consecutive_failures: 0, last_success_at: collected },
        },
      ],
      isPending: false,
      isError: false,
    });
    renderTab();

    expect(screen.getAllByText('3m ago')[0]).toHaveAttribute(
      'title',
      `${new Date(collected).toLocaleString()} (${browserTimezone()})`
    );
  });
  describe('Group by (PMM-15712)', () => {
    const groupBy = (label: string) => {
      fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Group by' }));
      fireEvent.click(screen.getByRole('option', { name: label }));
    };

    it('opens flat, with no drag-to-group handle on any column header', () => {
      renderTab();

      expect(
        screen.getByRole('combobox', { name: 'Group by' })
      ).toHaveTextContent('None');
      for (const header of screen.getAllByRole('columnheader')) {
        expect(
          within(header).queryByRole('button', { name: /move/i })
        ).toBeNull();
      }
    });

    it('names an unset environment, never "null" or "undefined"', () => {
      serve(
        topology([
          {
            env_name: null,
            clusters: [
              cluster({
                name: 'a',
                services: [service({ service_name: 'a-1' })],
              }),
            ],
          },
          {
            env_name: undefined,
            clusters: [
              cluster({
                name: 'b',
                services: [service({ service_name: 'b-1' })],
              }),
            ],
          },
        ])
      );
      renderTab();

      groupBy('Environment');

      const header = rowFor('Unassigned environment');
      expect(header).toHaveTextContent('(2)');
      expect(
        within(header).getByRole('button', {
          name: 'Collapse Unassigned environment',
        })
      ).toBeInTheDocument();
      expect(screen.queryByText(/^(null|undefined)$/)).toBeNull();
    });

    it('keeps two clusters sharing a name, or both unnamed, in separate groups', () => {
      serve(
        topology([
          {
            env_name: 'production',
            clusters: [
              cluster({
                name: 'orders',
                services: [service({ service_name: 'orders-a' })],
              }),
              cluster({
                name: 'orders',
                services: [service({ service_name: 'orders-b' })],
              }),
              cluster({
                name: null,
                services: [service({ service_name: 'solo-a' })],
              }),
              cluster({
                name: null,
                services: [service({ service_name: 'solo-b' })],
              }),
            ],
          },
        ])
      );
      renderTab();

      groupBy('Cluster');

      expect(
        screen.getAllByRole('button', { name: 'Collapse orders' })
      ).toHaveLength(2);
      expect(
        screen.getAllByRole('button', { name: 'Collapse No cluster' })
      ).toHaveLength(2);
    });

    it('groups by cluster from the select, with every group open', () => {
      renderTab();

      groupBy('Cluster');

      const header = rowFor('orders');
      expect(header).toHaveTextContent('(4)');
      expect(
        within(header).getByRole('button', { name: 'Collapse orders' })
      ).toHaveAttribute('aria-expanded', 'true');
      expect(
        screen.getByRole('link', { name: 'orders-2' })
      ).toBeInTheDocument();

      groupBy('None');
      expect(screen.queryByText('(4)')).toBeNull();
    });
  });

  describe('why a service is failing (PMM-15805)', () => {
    const FAILING_SINCE = '2026-10-06T09:00:00Z';
    const withScan = (overrides: Partial<OmInventoryService>) =>
      useOmInventoryServices.mockReturnValue({
        data: [
          {
            service_id: 's1',
            node_id: 'n1',
            observed: {},
            ...overrides,
            freshness: {
              last_success_at: '2026-10-06T08:00:00Z',
              last_attempt_at: '2026-10-06T10:00:00Z',
              consecutive_failures: 0,
              ...overrides.freshness,
            },
          } as OmInventoryService,
        ],
        isPending: false,
        isError: false,
      });
    const failingWith = (
      code: string,
      overrides: Partial<OmInventoryService> = {}
    ) =>
      withScan({
        ...overrides,
        freshness: {
          failing_since: FAILING_SINCE,
          consecutive_failures: 2,
          last_error: 'could not query the database: connection refused',
          last_error_code: code,
          ...overrides.freshness,
        } as OmInventoryService['freshness'],
      });

    it.each(
      Object.entries(SERVICE_SCAN_ERROR_KIND).filter(
        ([code]) => code !== 'unknown'
      )
    )(
      'shows a %s failure by its label in the default columns',
      (code, kind) => {
        serve(ordersEstate());
        failingWith(code);
        renderTab();

        expect(
          within(rowFor('orders-1')).getByTestId('om-scan-failure-reason')
        ).toHaveTextContent(kind.label);
        expect(
          screen.queryByRole('columnheader', { name: /^Scan/ })
        ).toBeNull();
      }
    );

    it('shows an unknown failure by the first line of its raw error', () => {
      serve(ordersEstate());
      failingWith('something_new', {
        freshness: {
          last_error: 'payload exploded\nTraceback ...',
        } as OmInventoryService['freshness'],
      });
      renderTab();

      expect(
        within(rowFor('orders-1')).getByTestId('om-scan-failure-reason')
      ).toHaveTextContent('payload exploded');
    });

    it('keeps the Scan column in the chooser', () => {
      renderTab();

      fireEvent.click(
        screen.getByRole('button', { name: 'Show/Hide columns' })
      );
      expect(
        within(screen.getByRole('menu')).getByText('Scan')
      ).toBeInTheDocument();
    });

    it('states the label, raw error and hint in the drawer', () => {
      serve(ordersEstate());
      failingWith('no_output');
      renderTab();
      fireEvent.click(
        within(rowFor('orders-1')).getByTestId('om-service-status')
      );

      const panel = within(
        within(screen.getByRole('dialog')).getByTestId('scan-failure')
      );
      expect(
        panel.getByText(
          `Scans failing: ${SERVICE_SCAN_ERROR_KIND.no_output.label}`
        )
      ).toBeInTheDocument();
      expect(panel.getByTestId('scan-error')).toHaveTextContent(
        'could not query the database: connection refused'
      );
      expect(
        panel.getByText(SERVICE_SCAN_ERROR_KIND.no_output.hint)
      ).toBeInTheDocument();
    });

    it('reads a down service whose scan found no server process as "not running", dated by that scan', () => {
      serve(ordersEstate('SERVICE_STATUS_DOWN'));
      failingWith('database_unreachable', { server_running: false });
      renderTab();

      const caption = within(rowFor('orders-1')).getByTestId('om-not-running');
      expect(caption).toHaveTextContent('not running');
      expect(caption).toHaveAttribute(
        'title',
        expect.stringMatching(/last scanned, .* ago \(/)
      );
      expect(
        within(rowFor('orders-1')).queryByTestId('om-down-for')
      ).toBeNull();
    });

    it('leaves a down row as it is when a scan that reached the server read no process', () => {
      serve(ordersEstate('SERVICE_STATUS_DOWN'));
      withScan({ server_running: false });
      renderTab();

      const row = within(rowFor('orders-1'));
      expect(row.queryByTestId('om-not-running')).toBeNull();
      expect(row.getByTestId('om-down-for')).toBeInTheDocument();
    });

    it('leaves a down row as it is when server_running is unset', () => {
      serve(ordersEstate('SERVICE_STATUS_DOWN'));
      withScan({ server_running: null });
      renderTab();

      const row = within(rowFor('orders-1'));
      expect(row.queryByTestId('om-not-running')).toBeNull();
      expect(row.getByTestId('om-down-for')).toBeInTheDocument();
    });

    it('leaves a down row as it is when the failed scan brought no process facts', () => {
      serve(ordersEstate('SERVICE_STATUS_DOWN'));
      failingWith('timed_out', { server_running: false });
      renderTab();

      const row = within(rowFor('orders-1'));
      expect(row.queryByTestId('om-not-running')).toBeNull();
      expect(row.getByTestId('om-down-for')).toBeInTheDocument();
    });

    it('links an empty fleet to the Operations for MongoDB guide', () => {
      serve(topology([]));
      renderTab();

      expect(
        within(screen.getByTestId('om-empty-state')).getByRole('link', {
          name: 'Read about Operations for MongoDB',
        })
      ).toHaveAttribute('href', OPERATIONS_DOC_URL);
    });
  });
});
