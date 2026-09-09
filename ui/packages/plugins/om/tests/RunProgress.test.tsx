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
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { RunProgress } from '../src/components/RunProgress';
import type {
  OmBootstrapHost,
  OmBootstrapStep,
  OmGetBootstrapRunResponse,
} from '../src/types';

/**
 * `RunProgress` renders an Abort button backed by `useCancelBootstrapRun`
 * (a `useMutation`), which throws without a `QueryClient` in context - every
 * render in this file needs one, even the tests that never touch Abort.
 */
function renderWithClient(ui: ReactElement) {
  const queryClient = new QueryClient();
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
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

describe('RunProgress', () => {
  it('shows the environment and cluster the run was triggered with', () => {
    renderWithClient(
      <RunProgress run={run({ environment: 'staging', cluster: 'orders' })} />
    );

    expect(
      screen.getByText(/rs-orders-prod \(staging \/ orders\)/)
    ).toBeInTheDocument();
  });

  it('shows only the replica set name when neither was given', () => {
    renderWithClient(<RunProgress run={run()} />);

    expect(screen.getByText(/^Run run-abc:/)).toBeInTheDocument();
    expect(screen.queryByText(/\(/)).not.toBeInTheDocument();
  });

  it('shows just the one label given, without a stray separator', () => {
    renderWithClient(<RunProgress run={run({ environment: 'staging' })} />);

    expect(screen.getByText(/rs-orders-prod \(staging\)/)).toBeInTheDocument();
  });

  // The regression this exists for: SEP reports the run succeeded before PMM's
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
  });

  // The regression this exists for: a freshly created run has nothing
  // `running` yet -- every step is `pending` until the stepper's next tick --
  // so the matrix alone looked identical to a run that was stuck, not one
  // that had just started.
  it('shows a starting indicator while every step is still pending', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'running',
          hosts: [host({ steps: [step('pre_check', 'pending')] })],
        })}
      />
    );

    expect(screen.getByText('Starting…')).toBeInTheDocument();
  });

  it('hides the starting indicator once any step has been dispatched', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'running',
          hosts: [host({ steps: [step('pre_check', 'running')] })],
        })}
      />
    );

    expect(screen.queryByText('Starting…')).not.toBeInTheDocument();
  });

  it('renders one column per host, by name', () => {
    renderWithClient(
      <RunProgress
        run={run({ hosts: [host({ host: 'n1' }), host({ host: 'n2' })] })}
      />
    );

    expect(screen.getByText('n1')).toBeInTheDocument();
    expect(screen.getByText('n2')).toBeInTheDocument();
  });

  // The regression this exists for: finalize_steps (e.g. enable_auth) were
  // dispatched and tracked correctly on the backend but never reached the wire
  // until GetBootstrapRunResponse learned the field, and even then a reader
  // saw nothing after the run's own steps unless something rendered them.
  it('shows a finalize step as its own row once dispatched', () => {
    renderWithClient(
      <RunProgress
        run={run({
          hosts: [
            host({
              steps: [step('pre_check', 'succeeded')],
              finalize_steps: [step('enable_auth', 'running')],
            }),
          ],
        })}
      />
    );

    expect(screen.getByText('pre_check')).toBeInTheDocument();
    expect(screen.getByText('enable_auth')).toBeInTheDocument();
    expect(screen.getByLabelText(/enable_auth: Running/)).toBeInTheDocument();
  });

  // The regression this exists for: a single cell spanning every host column
  // centers within their combined width, which for an odd host count lands
  // the icon looking like it belongs to whichever column is in the middle --
  // see the same run-level row's own comment in RunProgress.tsx.
  it('repeats the same run-level outcome under every host column, once per row', () => {
    renderWithClient(
      <RunProgress
        run={run({
          hosts: [host({ host: 'n1' }), host({ host: 'n2' })],
          run_steps: [step('rs_initiate', 'succeeded')],
        })}
      />
    );

    expect(screen.getAllByText('rs_initiate')).toHaveLength(1);
    expect(screen.getAllByLabelText(/rs_initiate: Succeeded/)).toHaveLength(2);
  });

  // The regression this exists for: rollback used to replace the forward and
  // finalize rows entirely, so a reader diagnosing a failed run lost the
  // record of what actually happened before the teardown started.
  it('keeps forward and finalize steps visible once a host starts rolling back, with rollback steps added below', () => {
    renderWithClient(
      <RunProgress
        run={run({
          status: 'failed',
          hosts: [
            host({
              steps: [step('pre_check', 'succeeded')],
              rollback_steps: [step('stop_service', 'running')],
              finalize_steps: [step('enable_auth', 'pending')],
            }),
          ],
        })}
      />
    );

    expect(screen.getByText('pre_check')).toBeInTheDocument();
    expect(screen.getByText('enable_auth')).toBeInTheDocument();
    expect(screen.getByText('stop_service')).toBeInTheDocument();
    expect(screen.getByText('Rollback')).toBeInTheDocument();
    expect(screen.getByText('Rolling back every host.')).toBeInTheDocument();
  });

  it('offers an Abort button while the run is still running', () => {
    renderWithClient(<RunProgress run={run({ status: 'running' })} />);

    expect(screen.getByRole('button', { name: 'Abort' })).toBeInTheDocument();
  });

  // The regression this guards against: SEP's own :cancel route 409s once
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
