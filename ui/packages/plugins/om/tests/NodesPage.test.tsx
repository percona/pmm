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
import { SnackbarProvider } from 'notistack';
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
    <SnackbarProvider>
      <MemoryRouter>
        <NodesPage />
      </MemoryRouter>
    </SnackbarProvider>
  );
};

const rowFor = (name: string) =>
  screen
    .getAllByRole('row')
    .find((row) => within(row).queryByText(name)) as HTMLElement;

const openRowMenu = (name: string) =>
  fireEvent.click(
    within(rowFor(name)).getByRole('button', { name: /More actions/ })
  );

const openRemoveDialog = async (name: string) => {
  openRowMenu(name);
  fireEvent.click(
    screen.getByRole('menuitem', { name: 'Remove duplicate entry' })
  );
  return screen.findByRole('dialog');
};

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

  it('offers removal only once the row menu is opened', () => {
    renderPage();

    expect(screen.queryByText('Remove duplicate entry')).toBeNull();
    openRowMenu('node00');

    expect(
      screen.getByRole('menuitem', { name: 'Remove duplicate entry' })
    ).toBeInTheDocument();
  });

  // The whole point of the menu move was that removal stayed reachable, not that it
  // went away. This walks the path a user now takes: menu, item, confirm dialog,
  // confirm - and asserts the mutation is actually called with the node.
  it('reaches the confirm dialog from the menu, and removes on confirm', async () => {
    renderPage();

    const dialog = await openRemoveDialog('node00');
    expect(dialog).toHaveTextContent('Remove the entry for node00?');

    fireEvent.click(
      within(dialog).getByRole('button', { name: 'Remove entry' })
    );

    await waitFor(() => expect(forgetOne).toHaveBeenCalledWith('node-1'));
  });

  // Plain words, the loss stated outright, and the confirm carrying the weight: a
  // removal that comes back on the next scan is housekeeping, not a red alert.
  it('says the scan history is lost, and makes confirm the primary button', async () => {
    renderPage([
      host({
        services: [
          { service_id: 's1' } as OmInventoryHost['services'][number],
          { service_id: 's2' } as OmInventoryHost['services'][number],
        ],
      }),
    ]);

    const dialog = await openRemoveDialog('node00');
    expect(dialog).toHaveTextContent('scan history is deleted permanently');
    expect(dialog).toHaveTextContent('the 2 services that Operations recorded');
    expect(dialog).toHaveTextContent('comes back on the next scan');
    expect(dialog).not.toHaveTextContent(/row\(s\)|Operations row/);

    const confirm = within(dialog).getByRole('button', {
      name: 'Remove entry',
    });
    expect(confirm).toHaveClass('MuiButton-contained');
    expect(confirm).not.toHaveClass('MuiButton-colorError');
    expect(within(dialog).getByRole('button', { name: 'Cancel' })).toHaveClass(
      'MuiButton-text'
    );
  });

  it('reports what was removed, and that the node comes back', async () => {
    renderPage();

    const dialog = await openRemoveDialog('node00');
    fireEvent.click(
      within(dialog).getByRole('button', { name: 'Remove entry' })
    );

    expect(
      await screen.findByText(
        'Removed node00 from Operations. If PMM still monitors it, it comes back on the next scan and is counted again.'
      )
    ).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
  });

  // A partial failure keeps the dialog open with the failure named, and claims
  // nothing: a success message over a node that is still there would be a lie.
  it('reports nothing while a removal has failed', async () => {
    forgetOne.mockRejectedValueOnce(new Error('boom'));
    renderPage();

    const dialog = await openRemoveDialog('node00');
    fireEvent.click(
      within(dialog).getByRole('button', { name: 'Remove entry' })
    );

    expect(await within(dialog).findByText('node00: boom')).toBeInTheDocument();
    expect(screen.queryByText(/^Removed /)).toBeNull();
  });

  it('removes several selected nodes from one neutral bulk action', async () => {
    renderPage([
      host(),
      host({ node_id: 'node-2', name: 'node01', address: '10.0.0.2' }),
    ]);

    fireEvent.click(
      within(rowFor('node00')).getByRole('checkbox', { name: /select row/i })
    );
    fireEvent.click(
      within(rowFor('node01')).getByRole('checkbox', { name: /select row/i })
    );
    const bulk = screen.getByRole('button', {
      name: 'Remove duplicate entries',
    });
    expect(bulk).not.toHaveClass('MuiButton-colorError');
    fireEvent.click(bulk);

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toHaveTextContent('Remove the entries for 2 nodes?');
    fireEvent.click(
      within(dialog).getByRole('button', { name: 'Remove entries' })
    );

    await waitFor(() => expect(forgetOne).toHaveBeenCalledTimes(2));
    expect(
      await screen.findByText(
        'Removed 2 nodes from Operations. Any that PMM still monitors come back on the next scan and are counted again.'
      )
    ).toBeInTheDocument();
  });

  it('closes the dialog without removing anything when cancelled', async () => {
    renderPage();

    fireEvent.click(
      within(await openRemoveDialog('node00')).getByRole('button', {
        name: 'Cancel',
      })
    );

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(forgetOne).not.toHaveBeenCalled();
    expect(screen.queryByText(/^Removed /)).toBeNull();
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
