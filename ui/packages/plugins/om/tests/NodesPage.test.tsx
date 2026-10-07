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
  is_pmm_server_node: false,
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

const renderPage = (hosts: OmInventoryHost[] = [host()], route = '/') => {
  useOmInventoryHosts.mockReturnValue({
    data: hosts,
    isPending: false,
    isError: false,
  });
  return render(
    <MemoryRouter initialEntries={[route]}>
      <NodesPage />
    </MemoryRouter>
  );
};

const rowFor = (name: string) =>
  screen
    .getAllByRole('row')
    .find((row) => within(row).queryByText(name)) as HTMLElement;

/**
 * The bulk Install button in the selection bar.
 *
 * Found by *not* being inside a row: it carries the same accessible name as the
 * per-row button, so `getByRole` would be ambiguous. Its tooltip is read off the
 * wrapper span rather than the button, because a disabled MUI button fires no
 * pointer events - which is why that span exists in the first place.
 */
const bulkInstall = () =>
  screen
    .getAllByRole('button', { name: 'Install MongoDB' })
    .find((button) => !button.closest('tr')) as HTMLElement;

const selectRows = (...names: string[]) => {
  for (const name of names) {
    fireEvent.click(within(rowFor(name)).getByRole('checkbox'));
  }
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
  // PMM Server's own node gets no actions menu at all. Forget there would clear
  // Operations' record of the machine PMM runs on, the next scan would put it
  // straight back, and in between the fleet would be wrong - so there is nothing
  // for the menu to hold. P1 names Forget alongside Install for this row.
  it('offers no row actions menu on the PMM Server node', () => {
    renderPage([
      host({
        name: 'pmm-server',
        is_pmm_server_node: true,
        automation_eligible: false,
        automation_blocked_by_design: true,
        automation_blocked_reasons: [
          'this is the node PMM Server itself runs on, which Operations never installs onto',
        ],
      }),
    ]);

    const row = rowFor('pmm-server');
    expect(
      within(row).queryByRole('button', { name: /More actions/ })
    ).toBeNull();
    expect(screen.queryByText('Forget')).toBeNull();
  });

  // Any other node keeps it, including one blocked by design: a registered
  // replica-set member is not a target for an install, but forgetting it is a
  // perfectly reasonable thing to want. This is why is_pmm_server_node is its own
  // field rather than read off automation_blocked_by_design.
  it('keeps the row actions menu on a node blocked by design that is not PMM Server', () => {
    renderPage([
      host({
        name: 'member00',
        automation_eligible: false,
        automation_blocked_by_design: true,
        automation_blocked_reasons: [
          'a MongoDB service is already registered on this node',
        ],
      }),
    ]);

    expect(
      within(rowFor('member00')).getByRole('button', { name: /More actions/ })
    ).toBeInTheDocument();
  });

  // The bulk button's tooltip has to give the reason, not only the remedy (P7).
  // Two counts are refused for unrelated reasons and a single sentence would state
  // something false about MongoDB, so these two assertions are a pair: the first
  // checks the majority rule is taught, the second that it is *not* claimed when it
  // does not apply.
  it('explains the majority rule when exactly two nodes are selected', async () => {
    renderPage([
      host({ node_id: 'n1', name: 'node00' }),
      host({ node_id: 'n2', name: 'node01' }),
    ]);

    selectRows('node00', 'node01');
    fireEvent.mouseOver(bulkInstall().parentElement as HTMLElement);

    const title = await screen.findByRole('tooltip');
    expect(title.textContent).toMatch(/cannot form a majority/i);
    expect(title.textContent).toMatch(/Select one node, or three/i);
  });

  it('blames this preview, not MongoDB, when more than three are selected', async () => {
    renderPage([
      host({ node_id: 'n1', name: 'node00' }),
      host({ node_id: 'n2', name: 'node01' }),
      host({ node_id: 'n3', name: 'node02' }),
      host({ node_id: 'n4', name: 'node03' }),
    ]);

    selectRows('node00', 'node01', 'node02', 'node03');
    fireEvent.mouseOver(bulkInstall().parentElement as HTMLElement);

    const title = await screen.findByRole('tooltip');
    expect(title.textContent).toMatch(/This preview installs/i);
    expect(title.textContent).toMatch(/MongoDB itself supports larger sets/i);
    expect(title.textContent).not.toMatch(/majority/i);
  });

  // Task 6 / F7. Forget is housekeeping, but running it against a node an install
  // is mid-way through would clear the record of the machine being changed. The
  // tooltip is asserted to be the Install button's own wording, not a lookalike:
  // the point of Forget explaining itself is that it matches the other controls.
  it('disables Forget, with the install reason, while a node is mid-install', async () => {
    useOmBootstrapRuns.mockReturnValue({
      data: [{ status: 'running', hosts: [{ host: 'exec-1' }] }],
    });
    renderPage([host({ name: 'node00', executor_host: 'exec-1' })]);

    fireEvent.click(
      within(rowFor('node00')).getByRole('button', { name: /More actions/ })
    );

    const forget = screen.getByText('Forget').closest('li') as HTMLElement;
    expect(forget).toHaveAttribute('aria-disabled', 'true');

    fireEvent.mouseOver(forget.parentElement as HTMLElement);
    expect((await screen.findByRole('tooltip')).textContent).toBe(
      'Already part of an install in progress.'
    );
  });

  // The other half of the pair: an idle node's Forget still works, and carries no
  // tooltip at all rather than an empty one.
  it('leaves Forget usable on a node with no install running', () => {
    renderPage([host({ name: 'node00', executor_host: 'exec-1' })]);

    fireEvent.click(
      within(rowFor('node00')).getByRole('button', { name: /More actions/ })
    );

    const forget = screen.getByText('Forget').closest('li') as HTMLElement;
    expect(forget).not.toHaveAttribute('aria-disabled', 'true');
  });

  // The destination for P6's "link to the scan result". An error elsewhere names a
  // node; following it has to land on that node's row, not on a fleet the reader then
  // has to search by hand.
  it('focuses the node named in ?node=', () => {
    renderPage(
      [
        host({ node_id: 'n1', name: 'node00' }),
        host({ node_id: 'n2', name: 'node01' }),
      ],
      '/?node=node01'
    );

    expect(screen.getByText('node01')).toBeInTheDocument();
    expect(screen.queryByText('node00')).toBeNull();
  });

  it('shows the whole fleet when no node is named', () => {
    renderPage([
      host({ node_id: 'n1', name: 'node00' }),
      host({ node_id: 'n2', name: 'node01' }),
    ]);

    expect(screen.getByText('node00')).toBeInTheDocument();
    expect(screen.getByText('node01')).toBeInTheDocument();
  });

  it('disables the row Scan while a sweep is running', () => {
    useIsEstateRefreshing.mockReturnValue(true);
    renderPage();

    expect(
      within(rowFor('node00')).getByRole('button', { name: 'Scan' })
    ).toBeDisabled();
  });

  it('sends an empty page to the scans, not to PMM, since its rows are scan results', () => {
    renderPage([]);

    const empty = screen.getByTestId('om-empty-state');
    expect(empty).toHaveTextContent('There are no scan results yet');
    expect(empty).not.toHaveTextContent('PMM has no nodes registered');
    expect(
      within(empty).getByRole('link', { name: 'Go to Scans' })
    ).toHaveAttribute(
      'href',
      expect.stringMatching(/\/automations\?tab=scans$/)
    );
  });
});
