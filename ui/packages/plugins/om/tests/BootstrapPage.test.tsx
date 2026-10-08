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
import { afterEach, describe, expect, it, vi } from 'vitest';
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
  automation_blocked_by_design: false,
  is_pmm_server_node: false,
});

const HOSTS = [host(1), host(2), host(3)];

/**
 * The trigger's result, mutable so a test can put the wizard into its error state.
 *
 * `vi.mock`'s factory is hoisted and evaluated once, so the hook has to read this on
 * every call rather than closing over a value fixed at mock time.
 */
const triggerState: { isError: boolean; error?: { message: string } } = {
  isError: false,
};

/** The fleet the wizard sees, mutable for the same reason as triggerState. */
const hostsState: { data: OmInventoryHost[] } = { data: HOSTS };

/** What the wizard actually asked for, so a test can assert the request itself. */
const triggerCalls: Record<string, unknown>[] = [];

const { enqueueSnackbar } = vi.hoisted(() => ({ enqueueSnackbar: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar }));

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryHosts: () => ({
    data: hostsState.data,
    isLoading: false,
    isError: false,
  }),
  useTriggerHostBootstrap: () => ({
    mutateAsync: async (request: Record<string, unknown>) => {
      triggerCalls.push(request);
      return { run_id: 'run-1' };
    },
    reset: vi.fn(),
    isPending: false,
    isError: triggerState.isError,
    error: triggerState.error,
  }),
}));
vi.mock('../src/topologyHooks', () => ({
  useOmTopology: () => ({ data: undefined }),
}));

/** Land on Configure for all three nodes, with the replica set named. */
function openConfigure() {
  render(
    <MemoryRouter
      initialEntries={['/operations/nodes/install?nodes=node-1,node-2,node-3']}
    >
      <Routes>
        <Route path="/operations/nodes/install" element={<BootstrapPage />} />
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
        initialEntries={[
          '/operations/nodes/install?nodes=node-1,node-2,node-3',
        ]}
      >
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
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

// The wizard end of P6's last clause. The linkifier itself is covered in
// NodeNamesLinked.test.tsx; this asserts the wizard actually renders the refusal
// through it, with the node names it knows about.
describe('BootstrapPage install refusal', () => {
  afterEach(() => {
    triggerState.isError = false;
    triggerState.error = undefined;
  });

  it('links every node the refusal names to that node and its scan', () => {
    triggerState.isError = true;
    triggerState.error = {
      message:
        '2 of the selected node(s) cannot be installed onto -- db01: no scan has reported its operating system; db03: no automation agent is registered for it.',
    };

    render(
      <MemoryRouter
        initialEntries={[
          '/operations/nodes/install?nodes=node-1,node-2,node-3',
        ]}
      >
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
        </Routes>
      </MemoryRouter>
    );
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
    fireEvent.change(screen.getByLabelText(/Replica set name/), {
      target: { value: 'rs-orders' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Review' }));

    expect(screen.getByRole('link', { name: 'db01' })).toHaveAttribute(
      'href',
      '/operations/nodes?node=db01'
    );
    expect(screen.getByRole('link', { name: 'db03' })).toHaveAttribute(
      'href',
      '/operations/nodes?node=db03'
    );
    // db02 is in the selection but not in the refusal, so it is not blamed.
    expect(screen.queryByRole('link', { name: 'db02' })).toBeNull();
  });
});

// Task 4 / P1. Every one of these conditions used to be checked on the wizard's
// final button, inside TriggerHostBootstrap, after the whole form was filled in.
describe('BootstrapPage blockers', () => {
  const blockers = () =>
    screen.getAllByTestId('om-review-blocker').map((item) => item.textContent);

  it('lists everything keeping Review disabled, beside it', () => {
    openConfigure();
    fireEvent.change(screen.getByLabelText(/Replica set name/), {
      target: { value: '' },
    });
    fireEvent.change(screen.getByLabelText(/Data path/), {
      target: { value: '' },
    });
    fireEvent.change(screen.getByLabelText(/^Port/), {
      target: { value: '70000' },
    });

    expect(review()).toBeDisabled();
    expect(blockers()).toEqual([
      'a replica set name',
      'a data path',
      'a port from 1 to 65535',
    ]);
  });

  it('says nothing once Review can go ahead', () => {
    openConfigure();

    expect(review()).toBeEnabled();
    expect(screen.queryByTestId('om-review-blocker')).toBeNull();
  });

  it('gives every reason Configure is unavailable, not only the first', () => {
    hostsState.data = [{ ...host(1), automation_eligible: false }, host(2)];
    try {
      render(
        <MemoryRouter
          initialEntries={['/operations/nodes/install?nodes=node-1,node-2']}
        >
          <Routes>
            <Route
              path="/operations/nodes/install"
              element={<BootstrapPage />}
            />
          </Routes>
        </MemoryRouter>
      );

      const error = screen.getByTestId('om-error');
      expect(error).toHaveTextContent('Select exactly one node');
      expect(error).toHaveTextContent(
        '1 selected node cannot be installed onto'
      );
      expect(screen.getByRole('button', { name: 'Configure' })).toBeDisabled();
    } finally {
      hostsState.data = HOSTS;
    }
  });
});

describe('BootstrapPage step 1 preconditions', () => {
  const renderStep1 = () =>
    render(
      <MemoryRouter
        initialEntries={[
          '/operations/nodes/install?nodes=node-1,node-2,node-3',
        ]}
      >
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
        </Routes>
      </MemoryRouter>
    );

  afterEach(() => {
    hostsState.data = HOSTS;
  });

  const blocked = (overrides: Partial<OmInventoryHost>) => {
    hostsState.data = [{ ...host(1), ...overrides }, host(2), host(3)];
  };

  it('says every node is ready, and lets the user Configure', () => {
    renderStep1();

    expect(screen.getAllByText('Ready')).toHaveLength(3);
    expect(screen.getByRole('button', { name: 'Configure' })).toBeEnabled();
  });

  // Blocks rather than warns: letting the user through would only move the failure
  // to the final button, which is the complaint itself.
  it('blocks Configure while any selected node cannot be installed onto', () => {
    blocked({
      automation_eligible: false,
      automation_blocked_reasons: [
        'no scan has reported its operating system yet',
      ],
    });

    renderStep1();

    expect(screen.getByRole('button', { name: 'Configure' })).toBeDisabled();
    expect(
      screen.getByText(/1 selected node cannot be installed onto/)
    ).toBeInTheDocument();
  });

  it('states the reason on the row, not only in a summary', () => {
    blocked({
      automation_eligible: false,
      automation_blocked_reasons: [
        'no scan has reported its operating system yet',
      ],
    });

    renderStep1();

    expect(
      screen.getByText('no scan has reported its operating system yet')
    ).toBeInTheDocument();
  });

  // P6's last clause again, reached from the other direction: the same destination
  // NodeNamesLinked uses, so following a reason lands in one place either way.
  it('links a faulted node to that node and its latest scan', () => {
    blocked({
      automation_eligible: false,
      automation_blocked_reasons: ['its automation agent is not reachable'],
    });

    renderStep1();

    expect(screen.getByRole('link', { name: /latest scan/ })).toHaveAttribute(
      'href',
      '/operations/nodes?node=db01'
    );
  });

  // A node blocked by design is working exactly as intended, so it reads as a fact
  // and offers no scan to go and read.
  it('calls a by-design block "Not a target" and offers no scan link', () => {
    blocked({
      automation_eligible: false,
      automation_blocked_by_design: true,
      automation_blocked_reasons: [
        'a MongoDB service is already registered on this node',
      ],
    });

    renderStep1();

    expect(screen.getByText('Not a target')).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: /latest scan/ })).toBeNull();
  });

  // P8: the statement of what will be created belongs at the decision, not after
  // the form. It used to be on Review only.
  it('says what will be created on step 1', () => {
    renderStep1();

    expect(
      screen.getByText(/data directories, and systemd services will be created/)
    ).toBeInTheDocument();
  });
});

// Task 5 / P2. The Bind IP field defaulted to 0.0.0.0 with the helper "The
// interface(s) mongod listens on" - a database reachable from every network the
// machine is on, with nothing beside the field saying so.
describe('BootstrapPage security posture', () => {
  const openConfigureStep = () => {
    render(
      <MemoryRouter
        initialEntries={[
          '/operations/nodes/install?nodes=node-1,node-2,node-3',
        ]}
      >
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
        </Routes>
      </MemoryRouter>
    );
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
  };

  const chooseListenOn = (option: RegExp) => {
    fireEvent.mouseDown(screen.getByLabelText('Listen on'));
    fireEvent.click(screen.getByRole('option', { name: option }));
  };

  afterEach(() => {
    triggerCalls.length = 0;
  });

  it('defaults to each node binding its own address, and warns about neither', () => {
    openConfigureStep();

    expect(screen.getByLabelText('Listen on')).toHaveTextContent(
      /Each node's own address/
    );
    // 0.0.0.0 is not selected, so there is nothing to warn about.
    expect(screen.queryByText(/accepts connections from every/)).toBeNull();
  });

  // The warning belongs beside the field, because it is about the value selected
  // right now rather than about the feature in general.
  it('warns beside the field when all interfaces are chosen', () => {
    openConfigureStep();
    chooseListenOn(/All interfaces/);

    expect(
      screen.getByText(/accepts connections from every network/)
    ).toBeInTheDocument();
  });

  it('warns beside the data path when a scan found too little space there', () => {
    hostsState.data = [
      { ...host(1), observed: { data_dir_free_bytes: 1024 ** 3 } },
      host(2),
      host(3),
    ];
    try {
      openConfigureStep();

      expect(screen.getByTestId('om-low-disk-warning')).toHaveTextContent(
        'The install needs at least 5 GiB free at the data path, and the last ' +
          'scan found less at /var/lib/mongo on db01 (1.0 GiB).'
      );

      // The scan measured the default path only; another one it knows nothing about.
      fireEvent.change(screen.getByLabelText(/Data path/), {
        target: { value: '/data/mongo' },
      });
      expect(screen.queryByTestId('om-low-disk-warning')).toBeNull();
    } finally {
      hostsState.data = HOSTS;
    }
  });

  it('does not warn about space on nodes a scan found enough on, or never measured', () => {
    hostsState.data = [
      { ...host(1), observed: { data_dir_free_bytes: 6 * 1024 ** 3 } },
      host(2),
      host(3),
    ];
    try {
      openConfigureStep();

      expect(screen.queryByTestId('om-low-disk-warning')).toBeNull();
    } finally {
      hostsState.data = HOSTS;
    }
  });

  it('reveals a text field only for a custom address', () => {
    openConfigureStep();
    expect(screen.queryByLabelText(/Bind IP/)).toBeNull();

    chooseListenOn(/Custom/);

    expect(screen.getByLabelText(/Bind IP/)).toBeInTheDocument();
  });

  // The tab is gone, but what it carried is not: it was the only place naming the
  // auth mechanism and what is unavailable, and losing that would make P2 worse.
  it('states the posture on the step, with no tab to open', () => {
    openConfigureStep();

    expect(screen.queryByRole('tab', { name: 'Security' })).toBeNull();
    const posture = screen.getByText(/Security in this developer preview/);
    expect(posture).toBeInTheDocument();
    const alert = posture.closest('.MuiAlert-root') as HTMLElement;
    expect(alert.textContent).toMatch(/keyFile/);
    expect(alert.textContent).toMatch(/not encrypted/);
    expect(alert.textContent).toMatch(/LDAP/);
    expect(alert.textContent).toMatch(/KMIP/);
  });

  it('says developer preview, never Tech Preview', () => {
    openConfigureStep();

    expect(document.body.textContent).not.toMatch(/Tech Preview/i);
  });

  // The whole reason BootstrapMemberConfig needed a bind_ip: three members have
  // three different addresses, so one run-level value cannot express this.
  it('asks for each member to bind its own address', async () => {
    openConfigureStep();
    fireEvent.change(screen.getByLabelText(/Replica set name/), {
      target: { value: 'rs-orders' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Review' }));
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));

    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    const request = triggerCalls[0] as {
      memberConfigs: Record<string, { bind_ip?: string }>;
    };
    expect(request.memberConfigs['node-1'].bind_ip).toBe('10.0.0.1');
    expect(request.memberConfigs['node-2'].bind_ip).toBe('10.0.0.2');
    expect(request.memberConfigs['node-3'].bind_ip).toBe('10.0.0.3');
    await vi.waitFor(() =>
      expect(enqueueSnackbar).toHaveBeenCalledWith(
        'Install started on 3 nodes',
        {
          variant: 'success',
        }
      )
    );
  });

  it('sends one run-level address and no per-member ones for all interfaces', async () => {
    openConfigureStep();
    fireEvent.change(screen.getByLabelText(/Replica set name/), {
      target: { value: 'rs-orders' },
    });
    chooseListenOn(/All interfaces/);
    fireEvent.click(screen.getByRole('button', { name: 'Review' }));
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));

    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    const request = triggerCalls[0] as {
      bindIp: string;
      memberConfigs: Record<string, { bind_ip?: string }>;
    };
    expect(request.bindIp).toBe('0.0.0.0');
    // The same fact twice would be a second place for it to drift.
    expect(request.memberConfigs['node-1'].bind_ip).toBeUndefined();
  });
});
