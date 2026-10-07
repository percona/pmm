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
import { FleetServicesTab } from '../src/FleetServicesTab';
import type { OmServiceStatus, OmTopologyResponse } from '../src/types';
import { cluster, mixedEstate, service, topology } from './fixtures';

const { useOmTopology } = vi.hoisted(() => ({ useOmTopology: vi.fn() }));

vi.mock('../src/topologyHooks', () => ({ useOmTopology }));
vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryServices: () => ({
    data: [],
    isPending: false,
    isError: false,
  }),
}));

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
  beforeEach(() => serve(mixedEstate()));

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
});
