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
import { browserTimezone } from '@pmm-extensions/framework';
import { OmApiError } from '../src/api';
import { NodesPage } from '../src/NodesPage';
import type { OmInventoryHost } from '../src/types';

const {
  useOmInventoryHosts,
  useRefreshInventory,
  useIsEstateRefreshing,
  useForgetHost,
  useOmBootstrapRuns,
  useActiveInventoryRun,
} = vi.hoisted(() => ({
  useOmInventoryHosts: vi.fn(),
  useRefreshInventory: vi.fn(),
  useIsEstateRefreshing: vi.fn(),
  useForgetHost: vi.fn(),
  useOmBootstrapRuns: vi.fn(),
  useActiveInventoryRun: vi.fn(),
}));

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryHosts,
  useRefreshInventory,
  useIsEstateRefreshing,
  useForgetHost,
  useOmBootstrapRuns,
  useActiveInventoryRun,
  useScanInFlight: () => ({ run: undefined, expectedSeconds: null }),
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
    <SnackbarProvider>
      <MemoryRouter initialEntries={[route]}>
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
    useActiveInventoryRun.mockReturnValue({ run: undefined, updatedAt: 0 });
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
    expect(screen.queryByText('Remove duplicate entry')).toBeNull();
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
  it('disables Remove duplicate entry, with the install reason, while a node is mid-install', async () => {
    useOmBootstrapRuns.mockReturnValue({
      data: [{ status: 'running', hosts: [{ host: 'exec-1' }] }],
    });
    renderPage([host({ name: 'node00', executor_host: 'exec-1' })]);

    openRowMenu('node00');

    const forget = screen.getByRole('menuitem', {
      name: 'Remove duplicate entry',
    });
    expect(forget).toHaveAttribute('aria-disabled', 'true');

    fireEvent.mouseOver(forget.parentElement as HTMLElement);
    expect((await screen.findByRole('tooltip')).textContent).toBe(
      'Already part of an install in progress.'
    );
  });

  // The other half of the pair: an idle node's Forget still works, and carries no
  // tooltip at all rather than an empty one.
  it('leaves Remove duplicate entry usable on a node with no install running', () => {
    renderPage([host({ name: 'node00', executor_host: 'exec-1' })]);

    openRowMenu('node00');

    const forget = screen.getByRole('menuitem', {
      name: 'Remove duplicate entry',
    });
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

  it('puts the local collection time and zone on the Collected hover', () => {
    const collected = new Date(Date.now() - 3 * 60 * 1000).toISOString();
    renderPage([
      host({
        freshness: { consecutive_failures: 0, last_success_at: collected },
      }),
    ]);

    expect(within(rowFor('node00')).getByText('3m ago')).toHaveAttribute(
      'title',
      `${new Date(collected).toLocaleString()} (${browserTimezone()})`
    );
  });

  it("puts the local collection time and zone on a failing node's last collection", () => {
    const collected = new Date(Date.now() - 40 * 60 * 1000).toISOString();
    renderPage([
      host({
        freshness: {
          consecutive_failures: 4,
          last_success_at: collected,
          failing_since: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
          last_error: 'ssh: connection refused.',
        },
      }),
    ]);

    expect(rowFor('node00')).toHaveTextContent(
      'Last collected 40m ago. Expand for the full error.'
    );
    expect(within(rowFor('node00')).getByText('40m ago')).toHaveAttribute(
      'title',
      `${new Date(collected).toLocaleString()} (${browserTimezone()})`
    );
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

  describe('a node whose scans are failing', () => {
    const TRACEBACK =
      'scan failed: Step \'run-script\' failed (exit code 1).\nTraceback (most recent call last):\n  File "probe.py", line 3\nSyntaxError: invalid syntax';
    const twoHoursAgo = () =>
      new Date(Date.now() - 2 * 3600 * 1000).toISOString();

    // The shape PMM Extensions answers with for a node that has never had a good scan:
    // no success time at all, which is exactly what used to hide its error.
    const failingHost = (
      freshness: Partial<OmInventoryHost['freshness']> = {}
    ) =>
      host({
        automation_eligible: false,
        automation_blocked_reasons: [
          "no scan has reported this node's operating system yet",
        ],
        freshness: {
          last_success_at: null,
          failing_since: twoHoursAgo(),
          consecutive_failures: 4,
          last_error: TRACEBACK,
          last_error_code: 'scan_crashed',
          last_run_id: 'run-42',
          ...freshness,
        },
      });

    const expand = (name: string) =>
      fireEvent.click(
        within(rowFor(name)).getByRole('button', { name: /expand/i })
      );

    it('states the failure on the row even though it never succeeded', () => {
      renderPage([failingHost()]);

      const row = rowFor('node00');
      // `failing_since` is two hours before the fixture was built, and a slow run can
      // render a second or more later - so "2h" may read "2h 1s". Exact text here
      // made the test pass or fail with the machine's load.
      expect(row).toHaveTextContent(
        /Failing for 2h( \d+s)?, 4 failed scans in a row: Scan crashed/
      );
      expect(row).toHaveTextContent('Never collected.');
    });

    // The install gate's sentence is a symptom of the failing scan, not its cause.
    it('says why it needs attention without hovering, naming the scan failure', () => {
      renderPage([failingHost()]);

      const row = rowFor('node00');
      expect(within(row).getByText('Needs attention')).toBeInTheDocument();
      expect(row).toHaveTextContent('Scans failing: Scan crashed');
      expect(row).not.toHaveTextContent('no scan has reported');
    });

    it('shows the whole error, its kind, the hint and the run in the expanded row', () => {
      renderPage([failingHost()]);
      expand('node00');

      const panel = within(screen.getByTestId('scan-failure'));
      // Whole and with its line breaks: a traceback folded onto one line is unreadable.
      expect(panel.getByTestId('scan-error').textContent).toBe(TRACEBACK);
      expect(
        panel.getByText('Scans failing: Scan crashed')
      ).toBeInTheDocument();
      expect(
        panel.getByText(/The scan crashed on the node\./)
      ).toBeInTheDocument();
      expect(panel.getByText(/4 failed scans in a row\./)).toBeInTheDocument();
      expect(
        panel.getByRole('link', { name: 'Open the scan that failed' })
      ).toHaveAttribute('href', '/automations?tab=scans&expand=run-42');
    });

    it('draws no run link when the server names no run', () => {
      renderPage([failingHost({ last_run_id: undefined })]);
      expand('node00');

      expect(screen.getByTestId('scan-error')).toBeInTheDocument();
      expect(
        screen.queryByRole('link', { name: 'Open the scan that failed' })
      ).toBeNull();
    });

    // Unrecognised and absent fold together: the raw error is the only honest answer.
    it.each([
      ['an unrecognised code', 'something_new'],
      ['no code at all', undefined],
    ])('shows the raw error and no hint for %s', (_label, code) => {
      renderPage([
        failingHost({
          last_error_code: code,
          last_error: 'probe exploded\nsecond line',
        }),
      ]);

      expect(rowFor('node00')).toHaveTextContent(
        /Failing for 2h( \d+s)?, 4 failed scans in a row: probe exploded/
      );
      expand('node00');
      expect(screen.getByTestId('scan-error').textContent).toBe(
        'probe exploded\nsecond line'
      );
      expect(
        within(screen.getByTestId('scan-failure')).getByText(
          'Scans failing: Scan failed'
        )
      ).toBeInTheDocument();
      // No sentence from the hint table for any kind.
      expect(
        screen.queryByText(
          /Check that|retried on the next scan|Install python3/
        )
      ).toBeNull();
    });

    it('shows none of it for a healthy node', () => {
      renderPage([host()]);
      const row = rowFor('node00');
      expect(row).not.toHaveTextContent('Failing');
      expect(row).not.toHaveTextContent('Needs attention');

      expand('node00');
      expect(screen.queryByTestId('scan-error')).toBeNull();
      expect(screen.queryByText(/Scans failing/)).toBeNull();
      expect(
        screen.queryByRole('link', { name: 'Open the scan that failed' })
      ).toBeNull();
    });
  });

  // Pedro, 2026-10-06: a failed scan of the PMM Server's own node is not a fleet
  // problem, so it is not counted - but it is still not hidden.
  describe("the PMM Server's own node", () => {
    const RED = 'rgb(211, 47, 47)';
    const failing = () => ({
      consecutive_failures: 3,
      failing_since: new Date().toISOString(),
      last_error: 'boom',
    });
    const nodes = () => [
      host({
        node_id: 'pmm',
        name: 'pmm-server',
        is_pmm_server_node: true,
        automation_eligible: false,
        automation_blocked_by_design: true,
        freshness: failing(),
      }),
      host({ node_id: 'node-2', name: 'node01', freshness: failing() }),
    ];

    it('is left out of the failing count and its filter', () => {
      renderPage(nodes());

      fireEvent.click(screen.getByText('1 failing'));

      expect(rowFor('node01')).toBeTruthy();
      expect(screen.queryByText('pmm-server')).toBeNull();
    });

    it('still states its failure, without the alarm colour', () => {
      renderPage(nodes());

      const statement = (name: string) =>
        within(rowFor(name)).getByText(/^Failing for/);
      expect(getComputedStyle(statement('node01')).color).toBe(RED);
      expect(getComputedStyle(statement('pmm-server')).color).not.toBe(RED);

      fireEvent.click(
        within(rowFor('pmm-server')).getByRole('button', { name: /expand/i })
      );
      expect(
        getComputedStyle(
          within(screen.getByTestId('scan-failure')).getByText(/^Scans failing/)
        ).color
      ).not.toBe(RED);
    });
  });

  describe('feedback on the page itself', () => {
    const conflict = new OmApiError(
      409,
      'A scan is already running on node00. The nodes update when it finishes.'
    );

    it('says a scan is already running, and links to that scan', () => {
      useRefreshInventory.mockReturnValue({
        refreshAll: vi.fn(),
        refreshHosts: vi.fn(),
        isPending: false,
        isError: true,
        error: conflict,
        submittedAt: 10,
        reset: vi.fn(),
      });
      useActiveInventoryRun.mockReturnValue({
        run: { run_id: 'run-7', status: 'RUN_STATUS_RUNNING' },
        updatedAt: 20,
      });

      renderPage();

      expect(screen.getByText(conflict.message)).toBeInTheDocument();
      expect(
        screen.getByRole('link', { name: 'Open the running scan' })
      ).toHaveAttribute('href', expect.stringContaining('expand=run-7'));
    });

    it('takes the notice down once that scan is over', () => {
      const reset = vi.fn();
      useRefreshInventory.mockReturnValue({
        refreshAll: vi.fn(),
        refreshHosts: vi.fn(),
        isPending: false,
        isError: true,
        error: conflict,
        submittedAt: 10,
        reset,
      });
      useActiveInventoryRun.mockReturnValue({ run: undefined, updatedAt: 20 });

      renderPage();

      expect(reset).toHaveBeenCalled();
    });

    it('says a scan could not start when the request failed outright', () => {
      useRefreshInventory.mockReturnValue({
        refreshAll: vi.fn(),
        refreshHosts: vi.fn(),
        isPending: false,
        isError: true,
        error: new OmApiError(502, 'PMM Extensions did not answer'),
        submittedAt: 10,
        reset: vi.fn(),
      });

      renderPage();

      const error = screen.getByTestId('om-error');
      expect(error).toHaveTextContent('Could not start a scan');
      expect(error).toHaveTextContent('PMM Extensions did not answer');
    });

    it('filters to the failing nodes from their count, and back', () => {
      renderPage([
        host(),
        host({
          node_id: 'node-2',
          name: 'node01',
          freshness: {
            consecutive_failures: 2,
            failing_since: new Date().toISOString(),
            last_error: 'boom',
          },
        }),
      ]);

      fireEvent.click(screen.getByText('1 failing'));
      expect(rowFor('node01')).toBeTruthy();
      expect(screen.queryByText('node00')).toBeNull();

      fireEvent.click(screen.getByText('1 failing'));
      expect(rowFor('node00')).toBeTruthy();
    });

    it('says it reports readiness, not database health, and where health is', () => {
      renderPage();

      expect(
        screen.getByText(/not how its databases are doing/)
      ).toBeInTheDocument();
      expect(
        screen.getByRole('link', { name: 'Clusters' })
      ).toBeInTheDocument();
    });

    it('makes Scan all the heavier action', () => {
      renderPage();

      expect(screen.getByRole('button', { name: 'Scan all' })).toHaveClass(
        'MuiButton-contained'
      );
    });
  });
});
