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

import { fireEvent, render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { FleetServicesTab } from '../src/FleetServicesTab';
import { mixedEstate } from './fixtures';

vi.mock('../src/topologyHooks', () => ({
  useOmTopology: () => ({
    data: mixedEstate(),
    isPending: false,
    isError: false,
  }),
}));
vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryServices: () => ({
    data: [],
    isPending: false,
    isError: false,
  }),
}));

describe('FleetServicesTab', () => {
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
});
