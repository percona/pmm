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
import { useState } from 'react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { BootstrapPage } from '../src/BootstrapPage';
import type { OmHostBootstrapAccepted, OmInventoryHost } from '../src/types';

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
  supported_mongodb_versions: ['7.0', '8.0'],
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
const hostsState: { data: OmInventoryHost[]; isError: boolean } = {
  data: HOSTS,
  isError: false,
};

/** What the wizard actually asked for, so a test can assert the request itself. */
const triggerCalls: Record<string, unknown>[] = [];

/** PMM's answer about the replica set name, and every question the wizard asked. */
const nameCheckState: {
  inUse: string | null;
  pending: boolean;
  error: string | null;
  calls: {
    name: string;
    environment: string;
    cluster: string;
    enabled: boolean;
  }[];
} = { inUse: null, pending: false, error: null, calls: [] };

/** How many times the wizard asked PMM to check the name again. */
const nameCheckRefetch = vi.fn();

const { enqueueSnackbar } = vi.hoisted(() => ({ enqueueSnackbar: vi.fn() }));
vi.mock('notistack', () => ({ enqueueSnackbar }));

vi.mock('../src/inventoryHooks', () => ({
  useOmInventoryHosts: () => ({
    data: hostsState.data,
    isLoading: false,
    isError: hostsState.isError,
    error: hostsState.isError ? new Error('refresh failed') : null,
  }),
  useReplicaSetNameCheck: (
    name: string,
    environment: string,
    cluster: string,
    enabled: boolean
  ) => {
    nameCheckState.calls.push({ name, environment, cluster, enabled });
    if (enabled && nameCheckState.error) {
      return {
        data: undefined,
        isError: true,
        error: new Error(nameCheckState.error),
        refetch: nameCheckRefetch,
      };
    }
    return {
      data:
        enabled && !nameCheckState.pending ? nameCheckState.inUse : undefined,
      isError: false,
    };
  },
  // Keeps the accepted run as `data`, re-rendering like useMutation does.
  useTriggerHostBootstrap: () => {
    const [data, setData] = useState<OmHostBootstrapAccepted>();
    return {
      mutateAsync: async (request: Record<string, unknown>) => {
        triggerCalls.push(request);
        const accepted: OmHostBootstrapAccepted = request.adminPassword
          ? { run_id: 'run-1' }
          : { run_id: 'run-1', generated_admin_password: 'generated-secret' };
        setData(accepted);
        return accepted;
      },
      data,
      reset: () => setData(undefined),
      isPending: false,
      isError: triggerState.isError,
      error: triggerState.error,
    };
  },
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

  it('lists each kind of problem PMM reports on its own line', () => {
    triggerState.isError = true;
    triggerState.error = {
      message:
        '1 of the selected node(s) cannot be installed onto -- db01: the last scan found no package manager on this node.\n' +
        'Replica set name rs-orders is already used in this environment and cluster by an install that is still running. Choose another name.',
    };

    render(
      <MemoryRouter initialEntries={['/operations/nodes/install?nodes=node-1']}>
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

    const items = within(screen.getByTestId('om-error')).getAllByRole(
      'listitem'
    );
    expect(items.map((item) => item.textContent)).toEqual([
      '1 of the selected node(s) cannot be installed onto -- db01: the last scan found no package manager on this node.',
      'Replica set name rs-orders is already used in this environment and cluster by an install that is still running. Choose another name.',
    ]);
  });
});

// Task 4 / P1. Every one of these conditions used to be checked on the wizard's
// final button, inside TriggerHostBootstrap, after the whole form was filled in.
describe('BootstrapPage blockers', () => {
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

  it('blocks Configure when the selected nodes run different distributions', () => {
    hostsState.data = [
      { ...host(1), observed: { os_id: 'ubuntu' } },
      { ...host(2), observed: { os_id: 'rocky' } },
      { ...host(3), observed: { os_id: 'ubuntu' } },
    ];

    renderStep1();

    expect(screen.getByRole('button', { name: 'Configure' })).toBeDisabled();
    expect(
      screen.getByText(
        'The selected nodes run different Linux distributions (ubuntu on db01, db03; rocky on db02). Go back and select nodes that all run the same one.'
      )
    ).toBeInTheDocument();
  });

  it('blocks Configure when the selected nodes share no MongoDB version', () => {
    blocked({ supported_mongodb_versions: [] });

    renderStep1();

    expect(screen.getByRole('button', { name: 'Configure' })).toBeDisabled();
    expect(
      screen.getByText(/The selected nodes have no MongoDB version in common/)
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

  it('blocks Review when a scan found too little space at the default data path', () => {
    hostsState.data = [
      { ...host(1), observed: { data_dir_free_bytes: 1024 ** 3 } },
      host(2),
      host(3),
    ];
    try {
      openConfigureStep();
      fireEvent.change(screen.getByLabelText(/Replica set name/), {
        target: { value: 'rs0' },
      });

      expect(screen.getByTestId('om-low-disk')).toHaveTextContent(
        'The install needs at least 5 GiB free at the data path, and the last ' +
          'scan found less at /var/lib/mongo on db01 (1.0 GiB).'
      );
      expect(screen.getByRole('button', { name: 'Review' })).toBeDisabled();
      expect(screen.getByTestId('om-review-blockers')).toHaveTextContent(
        'Free space at the data path on db01 (1.0 GiB)'
      );

      // The scan measured the default path only; another is left to the install's pre-check.
      fireEvent.change(screen.getByLabelText(/Data path/), {
        target: { value: '/data/mongo' },
      });
      expect(screen.queryByTestId('om-low-disk')).toBeNull();
      expect(screen.getByRole('button', { name: 'Review' })).toBeEnabled();
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

      expect(screen.queryByTestId('om-low-disk')).toBeNull();
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

  it('states the same posture again at Review', () => {
    openConfigureStep();
    fireEvent.change(screen.getByLabelText(/Replica set name/), {
      target: { value: 'rs0' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Review' }));

    expect(
      screen.getByRole('button', { name: 'Install MongoDB' })
    ).toBeInTheDocument();
    const alert = screen
      .getByText('Security in this developer preview')
      .closest('.MuiAlert-root') as HTMLElement;
    expect(alert.textContent).toMatch(/keyFile/);
    expect(alert.textContent).toMatch(/not encrypted/);
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

describe('BootstrapPage configure guidance', () => {
  const openConfigureStep = () => {
    render(
      <MemoryRouter initialEntries={['/operations/nodes/install?nodes=node-1']}>
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
        </Routes>
      </MemoryRouter>
    );
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
  };

  const typeInto = (label: RegExp | string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), { target: { value } });

  afterEach(() => {
    triggerCalls.length = 0;
    hostsState.data = HOSTS;
  });

  it('refuses a replica set name PMM says is already used, beside the field', () => {
    nameCheckState.inUse =
      'Replica set name rs0 is already used in this environment and cluster by the MongoDB service(s) orders-1. Choose another name.';
    try {
      openConfigureStep();
      typeInto(/Replica set name/, 'rs0');
      typeInto(/Environment/, 'prod ');

      expect(
        screen.getAllByText(
          'Replica set name rs0 is already used in this environment and cluster by the MongoDB service(s) orders-1. Choose another name.'
        )
      ).toHaveLength(2);
      expect(screen.getByRole('button', { name: 'Review' })).toBeDisabled();
      // Asked with the labels the install would carry, trimmed as they are sent.
      expect(nameCheckState.calls.at(-1)).toEqual({
        name: 'rs0',
        environment: 'prod',
        cluster: '',
        enabled: true,
      });
    } finally {
      nameCheckState.inUse = null;
      nameCheckState.calls.length = 0;
    }
  });

  it('holds Review until PMM has answered about the name', () => {
    nameCheckState.pending = true;
    try {
      openConfigureStep();
      typeInto(/Replica set name/, 'rs0');

      expect(screen.getByRole('button', { name: 'Review' })).toBeDisabled();
      expect(screen.getByTestId('om-review-blockers')).toHaveTextContent(
        'Checking whether this replica set name is free.'
      );
    } finally {
      nameCheckState.pending = false;
    }
  });

  it('blocks Review, and says why beside the field, when PMM cannot check the name', () => {
    nameCheckState.error = 'PMM Extensions is unreachable';
    try {
      openConfigureStep();
      typeInto(/Replica set name/, 'rs0');

      expect(screen.getByRole('button', { name: 'Review' })).toBeDisabled();
      expect(screen.getByTestId('om-review-blockers')).toHaveTextContent(
        'PMM could not check whether this replica set name is free: PMM Extensions is unreachable'
      );
      fireEvent.click(screen.getByRole('button', { name: 'Check again' }));
      expect(nameCheckRefetch).toHaveBeenCalledTimes(1);
    } finally {
      nameCheckState.error = null;
    }
  });

  it('asks about a name only once it is well formed', () => {
    openConfigureStep();
    typeInto(/Replica set name/, 'rs 0');

    expect(nameCheckState.calls.at(-1)?.enabled).toBe(false);
    nameCheckState.calls.length = 0;
  });

  it('offers only the supported versions, with 7.0 selected', () => {
    openConfigureStep();

    const version = screen.getByRole('combobox', { name: /MongoDB version/ });
    expect(version).toHaveTextContent('7.0');
    fireEvent.mouseDown(version);
    expect(
      screen.getAllByRole('option').map((option) => option.textContent)
    ).toEqual(['7.0', '8.0']);
  });

  it('offers only the versions every selected node supports, and sends that one', async () => {
    // Rocky Linux 10 gets 8.0 only.
    hostsState.data = [{ ...host(1), supported_mongodb_versions: ['8.0'] }];
    openConfigureStep();

    const version = screen.getByRole('combobox', {
      name: /MongoDB version/,
    });
    expect(version).toHaveTextContent('8.0');
    fireEvent.mouseDown(version);
    expect(
      screen.getAllByRole('option').map((option) => option.textContent)
    ).toEqual(['8.0']);
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape' });

    typeInto(/Replica set name/, 'rs0');
    fireEvent.click(review());
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));

    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    expect(triggerCalls[0].mongodbVersion).toBe('8.0');
  });

  it('sends the user back from Review, installing nothing, when a refetch drops the chosen version', () => {
    const page = () => (
      <MemoryRouter initialEntries={['/operations/nodes/install?nodes=node-1']}>
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
        </Routes>
      </MemoryRouter>
    );
    const { rerender } = render(page());
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
    typeInto(/Replica set name/, 'rs0');
    fireEvent.click(review());
    expect(
      screen.getByRole('button', { name: 'Install MongoDB' })
    ).toBeInTheDocument();

    hostsState.data = [{ ...host(1), supported_mongodb_versions: ['8.0'] }];
    rerender(page());

    expect(
      screen.queryByRole('button', { name: 'Install MongoDB' })
    ).not.toBeInTheDocument();
    expect(review()).toBeDisabled();
    expect(
      screen.getByText(
        'Choose a MongoDB version that every selected node supports.'
      )
    ).toBeInTheDocument();
    expect(triggerCalls).toHaveLength(0);
  });

  it('flags an invalid replica set name while typing, and clears it once fixed', () => {
    openConfigureStep();

    typeInto(/Replica set name/, 'my set:1');
    expect(screen.getByLabelText(/Replica set name/)).toHaveAttribute(
      'aria-invalid',
      'true'
    );
    expect(
      screen.getAllByText(
        'Replica set name can use only letters, digits, - and _.'
      )
    ).not.toHaveLength(0);
    expect(review()).toBeDisabled();
    // An invalid name is never offered as the cluster label.
    expect(
      screen.getByText('Optional. Defaults to the replica set name.')
    ).toBeInTheDocument();

    typeInto(/Replica set name/, 'rs0');
    expect(screen.getByLabelText(/Replica set name/)).toHaveAttribute(
      'aria-invalid',
      'false'
    );
    expect(review()).toBeEnabled();
  });

  it('says why Review is disabled on an untouched form, without flagging the name', () => {
    openConfigureStep();

    expect(review()).toBeDisabled();
    expect(review()).toHaveAttribute('aria-describedby', 'om-review-blockers');
    expect(screen.getByTestId('om-review-blockers')).toHaveTextContent(
      'Enter a replica set name.'
    );
    expect(screen.getByLabelText(/Replica set name/)).toHaveAttribute(
      'aria-invalid',
      'false'
    );
  });

  it('lists every reason Review is disabled, not only the first', () => {
    openConfigureStep();
    typeInto(/Replica set name/, 'rs0');
    typeInto(/Data path/, '');
    typeInto(/^Port/, '0');

    const items = within(screen.getByTestId('om-review-blockers'))
      .getAllByRole('listitem')
      .map((item) => item.textContent);
    expect(items).toEqual([
      'Enter a data path.',
      'Port must be a whole number from 1 to 65535.',
    ]);
  });

  it('hides the reasons once the form is valid', () => {
    openConfigureStep();
    typeInto(/Replica set name/, 'rs0');

    expect(screen.queryByTestId('om-review-blockers')).toBeNull();
    expect(review()).not.toHaveAttribute('aria-describedby');
  });

  it('says a blank cluster defaults to the replica set name, and sends it', async () => {
    openConfigureStep();
    typeInto(/Replica set name/, 'rs-orders');

    expect(
      screen.getByText(
        'Optional. Defaults to the replica set name (rs-orders).'
      )
    ).toBeInTheDocument();

    fireEvent.click(review());
    const clusterRow = screen.getByRole('rowheader', { name: 'Cluster' })
      .parentElement as HTMLElement;
    expect(clusterRow).toHaveTextContent('rs-orders');

    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));
    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    expect(triggerCalls[0].cluster).toBe('rs-orders');
  });

  it('sends a cluster the user typed as it is', async () => {
    openConfigureStep();
    typeInto(/Replica set name/, 'rs-orders');
    typeInto(/^Cluster/, 'orders');
    fireEvent.click(review());
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));

    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    expect(triggerCalls[0].cluster).toBe('orders');
  });
});

describe('BootstrapPage database administrator', () => {
  const renderWizard = () =>
    render(
      <MemoryRouter initialEntries={['/operations/nodes/install?nodes=node-1']}>
        <Routes>
          <Route path="/operations/nodes/install" element={<BootstrapPage />} />
          <Route path="/operations/automations" element={<p>Automations</p>} />
        </Routes>
      </MemoryRouter>
    );

  const configure = () => {
    fireEvent.click(screen.getByRole('button', { name: 'Configure' }));
    fireEvent.change(screen.getByLabelText(/Replica set name/), {
      target: { value: 'rs0' },
    });
  };

  afterEach(() => {
    triggerCalls.length = 0;
  });

  it('shows a generated password once and leaves only when the user has saved it', async () => {
    renderWizard();
    configure();
    fireEvent.click(review());
    expect(
      screen.getByRole('rowheader', { name: 'Administrator' }).parentElement
    ).toHaveTextContent('admin');
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));

    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    expect(triggerCalls[0].adminUsername).toBe('admin');
    expect(triggerCalls[0].adminPassword).toBeUndefined();
    const dialog = await screen.findByRole('dialog');
    expect(
      within(dialog).getByDisplayValue('generated-secret')
    ).toBeInTheDocument();
    expect(screen.queryByText('Automations')).toBeNull();

    fireEvent.click(
      within(dialog).getByRole('button', { name: 'I have saved it' })
    );
    expect(await screen.findByText('Automations')).toBeInTheDocument();
  });

  it('keeps the generated password on screen when a background refresh fails', async () => {
    const { rerender } = renderWizard();
    configure();
    fireEvent.click(review());
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));
    await screen.findByRole('dialog');

    hostsState.isError = true;
    try {
      rerender(
        <MemoryRouter
          initialEntries={['/operations/nodes/install?nodes=node-1']}
        >
          <Routes>
            <Route
              path="/operations/nodes/install"
              element={<BootstrapPage />}
            />
          </Routes>
        </MemoryRouter>
      );

      expect(screen.getByText('Could not load the nodes')).toBeInTheDocument();
      expect(
        within(screen.getByRole('dialog')).getByDisplayValue('generated-secret')
      ).toBeInTheDocument();
    } finally {
      hostsState.isError = false;
    }
  });

  it("shows a typed password's own problems on the password field", () => {
    renderWizard();
    configure();
    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: /Administrator password/ })
    );
    fireEvent.click(screen.getByRole('option', { name: 'Enter one' }));

    fireEvent.change(screen.getByLabelText(/^Password/), {
      target: { value: 'short' },
    });

    expect(screen.getByLabelText(/^Password/)).toHaveAccessibleDescription(
      'Administrator password must be at least 8 characters.'
    );
    expect(
      screen.getByLabelText(/^Confirm password/)
    ).not.toHaveAccessibleDescription(
      'Administrator password must be at least 8 characters.'
    );
  });

  it('sends a typed password once both fields match, and shows no dialog', async () => {
    renderWizard();
    configure();
    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: /Administrator password/ })
    );
    fireEvent.click(screen.getByRole('option', { name: 'Enter one' }));
    fireEvent.change(screen.getByLabelText(/^Password/), {
      target: { value: 'chosen-secret' },
    });
    fireEvent.change(screen.getByLabelText(/^Confirm password/), {
      target: { value: 'chosen-secre' },
    });
    expect(review()).toBeDisabled();
    expect(
      screen.getByLabelText(/^Confirm password/)
    ).toHaveAccessibleDescription('The administrator passwords do not match.');

    fireEvent.change(screen.getByLabelText(/^Confirm password/), {
      target: { value: 'chosen-secret' },
    });
    fireEvent.click(review());
    fireEvent.click(screen.getByRole('button', { name: 'Install MongoDB' }));

    await vi.waitFor(() => expect(triggerCalls).toHaveLength(1));
    expect(triggerCalls[0].adminPassword).toBe('chosen-secret');
    expect(await screen.findByText('Automations')).toBeInTheDocument();
    expect(screen.queryByRole('dialog')).toBeNull();
  });
});
