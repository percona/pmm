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
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OverviewPage } from '../src/OverviewPage';
import { mixedEstate } from './fixtures';

const { useOmTopology } = vi.hoisted(() => ({ useOmTopology: vi.fn() }));

vi.mock('../src/topologyHooks', () => ({ useOmTopology }));
// The Sync action brings its own queries and is not what these tests are about.
vi.mock('../src/components/SyncButton', () => ({ SyncButton: () => null }));

const renderPage = () =>
  render(
    <MemoryRouter>
      <OverviewPage />
    </MemoryRouter>
  );

const clusterRows = () =>
  screen
    .getAllByRole('row')
    .filter((row) => within(row).queryByTestId('om-cluster-health'));

describe('OverviewPage', () => {
  beforeEach(() => {
    useOmTopology.mockReturnValue({
      data: mixedEstate(),
      isPending: false,
      isError: false,
    });
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
});
