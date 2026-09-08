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

import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { HostProgress, RunProgress } from '../src/components/RunProgress';
import type {
  OmBootstrapHost,
  OmBootstrapStep,
  OmGetBootstrapRunResponse,
} from '../src/types';

function step(
  name: string,
  status: OmBootstrapStep['status']
): OmBootstrapStep {
  return { name, status, attempt_count: 1 };
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
    ...overrides,
  };
}

describe('HostProgress', () => {
  // The regression this exists for: finalize_steps (e.g. enable_auth) were
  // dispatched and tracked correctly on the backend but never reached the wire
  // until GetBootstrapRunResponse learned the field, and even then a reader
  // showed nothing after the run's own steps unless something rendered them.
  it('shows finalize steps alongside a host s own steps once dispatched', () => {
    const host: OmBootstrapHost = {
      host: 'n1',
      steps: [step('pre_check', 'succeeded')],
      rollback_steps: [step('stop_service', 'pending')],
      finalize_steps: [step('enable_auth', 'running')],
    };

    render(<HostProgress host={host} />);

    expect(screen.getByText(/pre_check: Succeeded/)).toBeInTheDocument();
    expect(screen.getByText(/enable_auth: Running/)).toBeInTheDocument();
  });

  it('shows only the rollback steps once a host starts rolling back, not its finalize steps', () => {
    const host: OmBootstrapHost = {
      host: 'n1',
      steps: [step('pre_check', 'succeeded')],
      rollback_steps: [step('stop_service', 'running')],
      finalize_steps: [step('enable_auth', 'pending')],
    };

    render(<HostProgress host={host} />);

    expect(screen.getByText(/stop_service: Running/)).toBeInTheDocument();
    expect(screen.queryByText(/pre_check/)).not.toBeInTheDocument();
    expect(screen.queryByText(/enable_auth/)).not.toBeInTheDocument();
  });
});

describe('RunProgress', () => {
  it('shows the environment and cluster the run was triggered with', () => {
    render(
      <RunProgress run={run({ environment: 'staging', cluster: 'orders' })} />
    );

    expect(
      screen.getByText(/rs-orders-prod \(staging \/ orders\)/)
    ).toBeInTheDocument();
  });

  it('shows only the replica set name when neither was given', () => {
    render(<RunProgress run={run()} />);

    expect(screen.getByText(/^Run run-abc:/)).toBeInTheDocument();
    expect(screen.queryByText(/\(/)).not.toBeInTheDocument();
  });

  it('shows just the one label given, without a stray separator', () => {
    render(<RunProgress run={run({ environment: 'staging' })} />);

    expect(screen.getByText(/rs-orders-prod \(staging\)/)).toBeInTheDocument();
  });
});
