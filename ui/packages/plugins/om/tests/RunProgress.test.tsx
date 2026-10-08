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

import type { ReactElement } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { RunProgress } from '../src/components/RunProgress';
import type {
  OmBootstrapHost,
  OmBootstrapStep,
  OmGetBootstrapRunResponse,
} from '../src/types';

// Node names come from the nodes list; the rest of the module stays real, since
// Abort's mutation needs the real useCancelBootstrapRun.
vi.mock('../src/inventoryHooks', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../src/inventoryHooks')>()),
  useOmInventoryHosts: () => ({
    data: [
      { node_id: 'id-1', executor_host: 'n1', name: 'db-1' },
      { node_id: 'id-2', executor_host: 'n2', name: 'db-2' },
      { node_id: 'id-3', executor_host: null, name: 'db-3' },
    ],
  }),
}));

/**
 * `RunProgress` renders an Abort button backed by `useCancelBootstrapRun`
 * (a `useMutation`), which throws without a `QueryClient` in context, and links
 * resolved through `useOmBase`, which needs a router.
 */
function renderWithClient(ui: ReactElement) {
  const queryClient = new QueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={['/om/automations']}>{ui}</MemoryRouter>
    </QueryClientProvider>
  );
}

function step(
  name: string,
  status: OmBootstrapStep['status']
): OmBootstrapStep {
  return { name, status, attempt_count: 1 };
}

function host(overrides: Partial<OmBootstrapHost> = {}): OmBootstrapHost {
  return {
    host: 'n1',
    steps: [step('pre_check', 'succeeded')],
    rollback_steps: [step('stop_service', 'pending')],
    finalize_steps: [step('enable_auth', 'pending')],
    ...overrides,
  };
}

function run(
  overrides: Partial<OmGetBootstrapRunResponse> = {}
): OmGetBootstrapRunResponse {
  return {
    run_id: 'run-abc',
    status: 'running',
    hosts: [],
    run_steps: [],
    replica_set_name: 'rs-orders-prod',
    mongodb_version: '7.0.8',
    started_at: '2026-01-01T00:00:00Z',
    cancel_requested: false,
    ...overrides,
  };
}

async function showDetails() {
  await userEvent.click(
    screen.getByRole('button', { name: 'Show step details' })
  );
}

describe('RunProgress', () => {
  it('leads with the replica set and its status, with the run id as secondary text', () => {
    renderWithClient(<RunProgress run={run()} />);

    expect(
      screen.getByRole('heading', { name: 'rs-orders-prod' })
    ).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByText('Run run-abc')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Copy run ID' })
    ).toBeInTheDocument();
  });

  it('copies the run id', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText },
      configurable: true,
    });
    renderWithClient(<RunProgress run={run()} />);

    await userEvent.click(screen.getByRole('button', { name: 'Copy run ID' }));

    expect(writeText).toHaveBeenCalledWith('run-abc');
  });

  it('shows the environment and cluster the run was triggered with', () => {
    renderWithClient(
      <RunProgress run={run({ environment: 'staging', cluster: 'orders' })} />
    );

    expect(screen.getByText('staging / orders')).toBeInTheDocument();
  });

  it('shows just the one label given, without a stray separator', () => {
    renderWithClient(<RunProgress run={run({ environment: 'staging' })} />);

    expect(screen.getByText('staging')).toBeInTheDocument();
  });

  it('reads as one line with the step count, current step and elapsed time', () => {
    renderWithClient(
      <RunProgress
        run={run({
          started_at: new Date(Date.now() - 192_000).toISOString(),
          hosts: [
            host({
              steps: [
                step('pre_check', 'succeeded'),
                step('install_package', 'running'),
              ],
            }),
          ],
          run_steps: [step('rs_initiate', 'pending')],
        })}
      />
    );

    expect(screen.getByRole('status')).toHaveTextContent(
      /^Step 2 of 4: Installing packages on 1 node, running for 3m 1[2-3]s$/
    );
  });

  // The regression this exists for: PMM Extensions reports the run succeeded before PMM's
  // own confirm_monitoring step has caught up, and the badge used to say
  // "Succeeded" anyway -- see bootstrapRunDisplayStatus's own doc comment.
  it('shows the run itself as still running while a host confirm_monitoring is running', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'succeeded',
          hosts: [
            host({
              finalize_steps: [step('confirm_monitoring', 'running')],
            }),
          ],
        })}
      />
    );

    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.queryByText('Succeeded')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Install summary')).not.toBeInTheDocument();
  });

  // The regression this exists for: a freshly created run has nothing
  // `running` yet -- every step is `pending` until the stepper's next tick --
  // so it looked identical to a run that was stuck, not one that had just started.
  it('says the run is starting while every step is still pending', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'running',
          hosts: [host({ steps: [step('pre_check', 'pending')] })],
        })}
      />
    );

    expect(screen.getByRole('status')).toHaveTextContent(/^Starting/);
  });

  it('keeps the step matrix one click away while the run is going', async () => {
    renderWithClient(
      <RunProgress
        run={run({ hosts: [host({ host: 'n1' }), host({ host: 'n2' })] })}
      />
    );

    expect(screen.queryByRole('table')).not.toBeInTheDocument();
    await showDetails();

    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Hide step details' })
    ).toHaveAttribute('aria-expanded', 'true');
  });

  it('heads the matrix columns with node names, falling back to the id', async () => {
    renderWithClient(
      <RunProgress
        run={run({ hosts: [host({ host: 'n1' }), host({ host: 'n9' })] })}
      />
    );
    await showDetails();

    expect(
      screen.getByRole('columnheader', { name: 'db-1' })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('columnheader', { name: 'n9' })
    ).toBeInTheDocument();
  });

  it('resolves a node id too, for a host keyed by the id it was requested with', async () => {
    renderWithClient(
      <RunProgress run={run({ hosts: [host({ host: 'id-3' })] })} />
    );
    await showDetails();

    expect(
      screen.getByRole('columnheader', { name: 'db-3' })
    ).toBeInTheDocument();
  });

  it('labels a known step and shows an unknown one by its key', async () => {
    renderWithClient(
      <RunProgress
        run={run({
          hosts: [
            host({
              steps: [
                step('pre_check', 'succeeded'),
                step('tune_kernel', 'running'),
              ],
              finalize_steps: [step('enable_auth', 'pending')],
            }),
          ],
        })}
      />
    );
    await showDetails();

    expect(screen.getByText('Checking the node')).toBeInTheDocument();
    expect(screen.queryByText('pre_check')).not.toBeInTheDocument();
    expect(screen.getByText('tune_kernel')).toBeInTheDocument();
    expect(
      screen.getByLabelText(/Enabling authentication: Pending/)
    ).toBeInTheDocument();
  });

  // The regression this exists for: a single cell spanning every host column
  // centers within their combined width, which for an odd host count lands
  // the icon looking like it belongs to whichever column is in the middle --
  // see the same run-level row's own comment in RunProgress.tsx.
  it('repeats the same run-level outcome under every host column, once per row', async () => {
    renderWithClient(
      <RunProgress
        run={run({
          hosts: [host({ host: 'n1' }), host({ host: 'n2' })],
          run_steps: [step('rs_initiate', 'succeeded')],
        })}
      />
    );
    await showDetails();

    expect(screen.getAllByText('Initiating the replica set')).toHaveLength(1);
    expect(
      screen.getAllByLabelText(/Initiating the replica set: Succeeded/)
    ).toHaveLength(2);
  });

  // The regression this exists for: rollback used to replace the forward and
  // finalize rows entirely, so a reader diagnosing a failed run lost the
  // record of what actually happened before the teardown started.
  it('opens the matrix on a failed run, with the failing cell and the rollback below', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'failed',
          finished_at: '2026-01-01T00:05:00Z',
          hosts: [
            host({
              host: 'n2',
              steps: [
                step('pre_check', 'succeeded'),
                step('install_package', 'failed'),
              ],
              rollback_steps: [step('stop_service', 'running')],
              finalize_steps: [step('enable_auth', 'pending')],
            }),
          ],
        })}
      />
    );

    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(
      screen.getByLabelText(/Installing packages: Failed/)
    ).toBeInTheDocument();
    expect(screen.getByText('Enabling authentication')).toBeInTheDocument();
    expect(screen.getByText('Stopping mongod')).toBeInTheDocument();
    expect(screen.getByText('Rollback')).toBeInTheDocument();
    expect(screen.getByText('Rolling back every node.')).toBeInTheDocument();
    expect(screen.getByRole('status')).toHaveTextContent(
      'Failed at step 2 of 3: Installing packages on db-2, after 5m'
    );
    expect(screen.queryByLabelText('Install summary')).not.toBeInTheDocument();
  });

  it('opens the matrix by itself when a watched run fails', () => {
    const running = run({
      hosts: [host({ steps: [step('install_package', 'running')] })],
    });
    const { rerender } = renderWithClient(<RunProgress run={running} />);
    expect(screen.queryByRole('table')).not.toBeInTheDocument();

    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <MemoryRouter>
          <RunProgress
            run={run({
              hosts: [host({ steps: [step('install_package', 'failed')] })],
            })}
          />
        </MemoryRouter>
      </QueryClientProvider>
    );

    expect(screen.getByRole('table')).toBeInTheDocument();
  });

  it('ends a successful run on a card saying what was created, with links to it', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'succeeded',
          finished_at: '2026-01-01T00:04:30Z',
          environment: 'prod',
          cluster: 'orders',
          hosts: [
            host({
              host: 'n1',
              finalize_steps: [step('confirm_monitoring', 'succeeded')],
            }),
            host({
              host: 'n2',
              finalize_steps: [step('confirm_monitoring', 'succeeded')],
            }),
          ],
        })}
      />
    );

    const card = screen.getByLabelText('Install summary');
    expect(card).toHaveTextContent('Replica set rs-orders-prod is installed');
    expect(card).toHaveTextContent('7.0.8');
    expect(card).toHaveTextContent('Environmentprod');
    expect(card).toHaveTextContent('Clusterorders');
    expect(card).toHaveTextContent('4m 30s');
    expect(screen.getByRole('link', { name: 'db-1' })).toHaveAttribute(
      'href',
      '/om/nodes?node=db-1'
    );
    expect(screen.getByRole('link', { name: 'db-2' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'View in Fleet' })).toHaveAttribute(
      'href',
      '/om/?tab=clusters'
    );
    expect(screen.getByRole('link', { name: 'View nodes' })).toHaveAttribute(
      'href',
      '/om/nodes'
    );
    expect(card).toHaveTextContent('Security settings still in place');
    expect(card).toHaveTextContent(/shared keyFile/);
    expect(card).toHaveTextContent(/TLS is off/);
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('offers an Abort button while the run is still running', () => {
    renderWithClient(<RunProgress run={run({ status: 'running' })} />);

    expect(screen.getByRole('button', { name: 'Abort' })).toBeInTheDocument();
  });

  // The regression this guards against: PMM Extensions' own :cancel route 409s once
  // cancellation was already requested (canCancelBootstrapRun's own doc
  // comment) -- offering the button again invites a confusing error rather
  // than reflecting that the request is already in flight.
  it('replaces the Abort button with a status line once cancellation was requested', () => {
    renderWithClient(
      <RunProgress run={run({ status: 'running', cancel_requested: true })} />
    );

    expect(
      screen.queryByRole('button', { name: 'Abort' })
    ).not.toBeInTheDocument();
    expect(
      screen.getByText(
        'Abort requested - rolling back once the current step stops.'
      )
    ).toBeInTheDocument();
  });

  it('offers no Abort button once the run has already reached a terminal status', () => {
    renderWithClient(<RunProgress run={run({ status: 'succeeded' })} />);

    expect(
      screen.queryByRole('button', { name: 'Abort' })
    ).not.toBeInTheDocument();
  });
});
