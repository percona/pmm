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
import { NodesPage } from '../src/NodesPage';
import type { OmInventoryHost } from '../src/types';

const {
  useOmInventoryHosts,
  useRefreshInventory,
  useIsEstateRefreshing,
  useForgetHost,
  useOmBootstrapRuns,
} = vi.hoisted(() => ({
  useOmInventoryHosts: vi.fn(),
  useRefreshInventory: vi.fn(),
  useIsEstateRefreshing: vi.fn(),
  useForgetHost: vi.fn(),
  useOmBootstrapRuns: vi.fn(),
}));

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryHosts,
  useRefreshInventory,
  useIsEstateRefreshing,
  useForgetHost,
  useOmBootstrapRuns,
}));

const host = (overrides: Partial<OmInventoryHost> = {}): OmInventoryHost => ({
  node_id: 'node-1',
  name: 'node00',
  address: '10.0.0.1',
  services: [],
  unregistered_mongods: [],
  automation_eligible: true,
  automation_blocked_reasons: [],
  automation_blocked_by_design: false,
  pmm_agent_connected: true,
  executor: { registered: true, reachable: true, driver_healthy: true },
  observed: {},
  freshness: {
    consecutive_failures: 0,
    last_success_at: '2026-10-06T09:00:00Z',
  },
  ...overrides,
});

const forgetOne = vi.fn();

const renderPage = (hosts: OmInventoryHost[] = [host()]) => {
  useOmInventoryHosts.mockReturnValue({
    data: hosts,
    isPending: false,
    isError: false,
  });
  return render(
    <MemoryRouter>
      <NodesPage />
    </MemoryRouter>
  );
};

const rowFor = (name: string) =>
  screen
    .getAllByRole('row')
    .find((row) => within(row).queryByText(name)) as HTMLElement;

describe('NodesPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useRefreshInventory.mockReturnValue({
      refreshAll: vi.fn(),
      refreshHosts: vi.fn(),
      isPending: false,
      isError: false,
    });
    useIsEstateRefreshing.mockReturnValue(false);
    useForgetHost.mockReturnValue({ mutateAsync: forgetOne, isPending: false });
    useOmBootstrapRuns.mockReturnValue({ data: [] });
    forgetOne.mockResolvedValue(undefined);
  });

  // The row's daily actions stay on the row. Forget does not, because three text
  // buttons did not fit the width the page gets - see the overflow menu's own
  // comment. These two assertions are a pair: the first says what is reachable in
  // one click, the second that the destructive one is not.
  it('keeps Scan and Install MongoDB on the row', () => {
    renderPage();

    const row = rowFor('node00');
    expect(
      within(row).getByRole('button', { name: 'Scan' })
    ).toBeInTheDocument();
    expect(
      within(row).getByRole('button', { name: 'Install MongoDB' })
    ).toBeInTheDocument();
  });

  it('does not offer Forget until the row menu is opened', () => {
    renderPage();

    expect(screen.queryByText('Forget')).toBeNull();
    fireEvent.click(
      within(rowFor('node00')).getByRole('button', { name: /More actions/ })
    );

    expect(screen.getByText('Forget')).toBeInTheDocument();
  });

  // The whole point of the menu move was that Forget stayed reachable, not that it
  // went away. This walks the path a user now takes: menu, item, confirm dialog,
  // confirm - and asserts the mutation is actually called with the node.
  it('reaches the confirm dialog from the menu, and forgets on confirm', async () => {
    renderPage();

    fireEvent.click(
      within(rowFor('node00')).getByRole('button', { name: /More actions/ })
    );
    fireEvent.click(screen.getByText('Forget'));

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent('Forget node00?');

    fireEvent.click(within(dialog).getByRole('button', { name: 'Forget' }));

    await waitFor(() => expect(forgetOne).toHaveBeenCalledWith('node-1'));
  });

  it('closes the dialog without forgetting when cancelled', async () => {
    renderPage();

    fireEvent.click(
      within(rowFor('node00')).getByRole('button', { name: /More actions/ })
    );
    fireEvent.click(screen.getByText('Forget'));
    fireEvent.click(
      within(await screen.findByRole('dialog')).getByRole('button', {
        name: 'Cancel',
      })
    );

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(forgetOne).not.toHaveBeenCalled();
  });

  // Two kinds of ineligible, and telling them apart is the whole point: a healthy
  // replica-set member is not a node that needs attention, it is one Operations
  // deliberately leaves alone. Conflating them put an amber warning on every
  // working cluster.
  it('says a node is not a target when the block is by design', () => {
    renderPage([
      host({
        automation_eligible: false,
        automation_blocked_by_design: true,
        automation_blocked_reasons: [
          'a MongoDB service is already registered on this node',
        ],
      }),
    ]);

    const row = rowFor('node00');
    expect(within(row).getByText('Not a target')).toBeInTheDocument();
    expect(within(row).queryByText('Needs attention')).toBeNull();
  });

  it('still says needs attention when the block is a fault', () => {
    renderPage([
      host({
        automation_eligible: false,
        automation_blocked_by_design: false,
        automation_blocked_reasons: [
          'host is not reachable by the Nomad client',
        ],
      }),
    ]);

    const row = rowFor('node00');
    expect(within(row).getByText('Needs attention')).toBeInTheDocument();
    expect(within(row).queryByText('Not a target')).toBeNull();
  });

  // A node Operations cannot act on must not offer the action that would fail.
  it('disables Install MongoDB on a node that is not eligible', () => {
    renderPage([
      host({
        automation_eligible: false,
        automation_blocked_reasons: [
          'PMM-Client is not installed or not connected',
        ],
      }),
    ]);

    expect(
      within(rowFor('node00')).getByRole('button', { name: 'Install MongoDB' })
    ).toBeDisabled();
  });

  // Scanning is estate-wide, so a sweep in flight has to disable the per-row trigger
  // too - otherwise a reader queues a second scan of a node already being scanned.
  it('disables the row Scan while a sweep is running', () => {
    useIsEstateRefreshing.mockReturnValue(true);
    renderPage();

    expect(
      within(rowFor('node00')).getByRole('button', { name: 'Scan' })
    ).toBeDisabled();
  });
});
