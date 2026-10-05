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
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { BootstrapPage } from '../src/BootstrapPage';
import type { OmInventoryHost } from '../src/types';

const host = (i: number): OmInventoryHost => ({
  node_id: `node-${i}`,
  name: `db0${i}`,
  address: `10.0.0.${i}`,
  executor_host: `db0${i}`,
  os: 'Ubuntu 24.04 LTS',
  kernel: '6.8',
  executor: { registered: true, reachable: true, driver_healthy: true },
  unregistered_mongods: [],
  observed: {},
  freshness: { consecutive_failures: 0 },
  services: [],
  pmm_agent_connected: true,
  automation_eligible: true,
  automation_blocked_reasons: [],
});

const HOSTS = [host(1), host(2), host(3)];

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryHosts: () => ({
    data: HOSTS,
    isLoading: false,
    isError: false,
  }),
  useTriggerHostBootstrap: () => ({
    mutateAsync: vi.fn(),
    reset: vi.fn(),
    isPending: false,
    isError: false,
  }),
}));
vi.mock('../src/topologyHooks', () => ({
  useOmTopology: () => ({ data: undefined }),
}));

/** Land on Configure for all three hosts, with the replica set named. */
function openConfigure() {
  render(
    <MemoryRouter
      initialEntries={['/om/hosts/bootstrap?hosts=node-1,node-2,node-3']}
    >
      <Routes>
        <Route path="/om/hosts/bootstrap" element={<BootstrapPage />} />
      </Routes>
    </MemoryRouter>
  );
  fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
  fireEvent.change(screen.getByLabelText(/Replica set name/), {
    target: { value: 'rs-orders' },
  });
  fireEvent.click(screen.getByText('Advanced: election settings'));
}

const review = () => screen.getByRole('button', { name: 'Review' });

describe('BootstrapPage election settings', () => {
  it('folds the settings away with their defaults stated', () => {
    render(
      <MemoryRouter
        initialEntries={['/om/hosts/bootstrap?hosts=node-1,node-2,node-3']}
      >
        <Routes>
          <Route path="/om/hosts/bootstrap" element={<BootstrapPage />} />
        </Routes>
      </MemoryRouter>
    );
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));

    expect(
      screen.getByText('Defaults: priority 1, votes on, not hidden, no delay.')
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Advanced: election settings/ })
    ).toHaveAttribute('aria-expanded', 'false');
  });

  it('ties the settings region to its heading, with one id each', () => {
    openConfigure();

    const summary = screen.getByRole('button', {
      name: /Advanced: election settings/,
    });
    const region = document.getElementById(
      summary.getAttribute('aria-controls')!
    );
    expect(region).not.toBeNull();
    expect(region).toHaveAttribute('aria-labelledby', summary.id);
    expect(document.querySelectorAll('#om-election-settings')).toHaveLength(1);
  });

  it('lets the default plan through with no warning', () => {
    openConfigure();

    expect(review()).toBeEnabled();
    expect(screen.queryByTestId('om-election-error')).toBeNull();
    expect(screen.queryByTestId('om-election-warning')).toBeNull();
  });

  it('blocks Review and says why when no member can become primary', () => {
    openConfigure();

    for (const name of ['db01', 'db02', 'db03']) {
      fireEvent.change(screen.getByLabelText(`Priority for ${name}`), {
        target: { value: '0' },
      });
    }

    expect(review()).toBeDisabled();
    expect(screen.getByTestId('om-election-error')).toHaveTextContent(
      'No member can become primary: db01, db02 and db03'
    );

    fireEvent.change(screen.getByLabelText('Priority for db02'), {
      target: { value: '1' },
    });
    expect(review()).toBeEnabled();
    expect(screen.queryByTestId('om-election-error')).toBeNull();
  });

  it('holds a hidden member at priority 0 and says so beside the row', () => {
    openConfigure();

    fireEvent.click(screen.getByLabelText('Hidden for db02'));

    const priority = screen.getByLabelText('Priority for db02');
    expect(priority).toHaveValue(0);
    expect(priority).toBeDisabled();
    const row = screen
      .getAllByTestId('om-election-row')
      .find((r) => within(r).queryByText('db02'))!;
    expect(row).toHaveTextContent(
      'Priority set to 0: a hidden member cannot become primary.'
    );
    expect(review()).toBeEnabled();
  });

  it('announces a delay turning off priority and votes, without a hover', () => {
    openConfigure();

    fireEvent.change(screen.getByLabelText('Delay for db03'), {
      target: { value: '3600' },
    });

    expect(screen.getByLabelText('Priority for db03')).toBeDisabled();
    expect(screen.getByLabelText('Votes for db03')).not.toBeChecked();
    expect(screen.getByLabelText('Hidden for db03')).toBeChecked();
    expect(screen.getByLabelText('Hidden for db03')).toBeDisabled();
    expect(
      screen.getByText(
        'Priority set to 0, votes turned off and hidden: a delayed member cannot vote or become primary, and applications must not read its delayed data.'
      )
    ).toBeInTheDocument();
    expect(screen.queryByTestId('om-election-error')).toBeNull();
  });

  it('warns on an even number of voters but keeps Review available', () => {
    openConfigure();

    fireEvent.click(screen.getByLabelText('Votes for db03'));

    expect(screen.getByTestId('om-election-warning')).toHaveTextContent(
      '2 voting members'
    );
    expect(review()).toBeEnabled();
  });
});
