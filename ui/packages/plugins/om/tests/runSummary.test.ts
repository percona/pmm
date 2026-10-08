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

import { describe, expect, it } from 'vitest';
import { bootstrapStepLabel } from '../src/constants';
import { formatRunElapsed } from '../src/format';
import { runHasFailure, runSummaryLine } from '../src/runSummary';
import type {
  OmBootstrapHost,
  OmBootstrapStep,
  OmGetBootstrapRunResponse,
} from '../src/types';

const STARTED = '2026-01-01T00:00:00Z';
const NOW = Date.parse(STARTED) + 192_000;
const name = (id: string) => `node-${id}`;

function step(
  stepName: string,
  status: OmBootstrapStep['status']
): OmBootstrapStep {
  return { name: stepName, status, attempt_count: 1 };
}

function host(
  id: string,
  steps: OmBootstrapStep[],
  overrides: Partial<OmBootstrapHost> = {}
): OmBootstrapHost {
  return {
    host: id,
    steps,
    rollback_steps: [
      step('stop_service', 'pending'),
      step('remove_data', 'pending'),
    ],
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
    run_steps: [step('rs_initiate', 'pending')],
    replica_set_name: 'rs0',
    mongodb_version: '7.0.8',
    started_at: STARTED,
    cancel_requested: false,
    ...overrides,
  };
}

const installing = (status: OmBootstrapStep['status']) => [
  step('pre_check', 'succeeded'),
  step('install_package', status),
];

describe('bootstrapStepLabel', () => {
  it('labels a step PMM Extensions is known to send', () => {
    expect(bootstrapStepLabel('install_package')).toBe('Installing packages');
  });

  it('falls back to the raw key for a step it does not know', () => {
    expect(bootstrapStepLabel('tune_kernel')).toBe('tune_kernel');
  });

  it('does not mistake an Object prototype key for a label', () => {
    expect(bootstrapStepLabel('constructor')).toBe('constructor');
  });
});

describe('formatRunElapsed', () => {
  it('counts up to now while a run has no finish time', () => {
    expect(formatRunElapsed(STARTED, null, NOW)).toBe('3m 12s');
  });

  it('stops at the finish time once there is one', () => {
    expect(formatRunElapsed(STARTED, '2026-01-01T00:01:00Z', NOW)).toBe('1m');
  });

  it('reads a clock behind the server as just started', () => {
    expect(formatRunElapsed(STARTED, null, Date.parse(STARTED) - 5000)).toBe(
      '0s'
    );
  });
});

describe('runSummaryLine', () => {
  it('counts every forward step and names the current one, on the nodes running it', () => {
    const line = runSummaryLine(
      run({
        hosts: [
          host('a', installing('running')),
          host('b', installing('running')),
          host('c', installing('succeeded')),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Step 2 of 4: Installing packages on 2 nodes, running for 3m 12s'
    );
  });

  it('counts the nodes left when none is running the step yet', () => {
    const line = runSummaryLine(
      run({
        hosts: [
          host('a', installing('pending')),
          host('b', installing('succeeded')),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Step 2 of 4: Installing packages on 1 node, running for 3m 12s'
    );
  });

  it('says nothing about nodes for a run-level step', () => {
    const line = runSummaryLine(
      run({
        hosts: [host('a', installing('succeeded'))],
        run_steps: [step('rs_initiate', 'running')],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Step 3 of 4: Initiating the replica set, running for 3m 12s'
    );
  });

  it('says the run is starting while nothing has been dispatched', () => {
    const line = runSummaryLine(
      run({ hosts: [host('a', [step('pre_check', 'pending')])] }),
      name,
      NOW
    );

    expect(line).toBe('Starting, running for 3m 12s');
  });

  it('says which step failed and where, by node name', () => {
    const line = runSummaryLine(
      run({
        status: 'failed',
        finished_at: '2026-01-01T00:02:00Z',
        hosts: [
          host('a', installing('succeeded')),
          host('b', installing('failed')),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Failed at step 2 of 4: Installing packages on node-b, after 2m'
    );
  });

  it('follows the rollback while it is in flight', () => {
    const line = runSummaryLine(
      run({
        hosts: [
          host('a', installing('failed'), {
            rollback_steps: [
              step('stop_service', 'succeeded'),
              step('remove_data', 'running'),
            ],
          }),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Rolling back, step 2 of 2: Removing data directories on 1 node, running for 3m 12s'
    );
  });

  it('reports a failed teardown step rather than a rollback still going', () => {
    const line = runSummaryLine(
      run({
        cancel_requested: true,
        hosts: [
          host('a', installing('pending'), {
            rollback_steps: [
              step('stop_service', 'succeeded'),
              step('remove_data', 'failed'),
            ],
          }),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Rollback failed at step 2 of 2: Removing data directories on node-a, running for 3m 12s'
    );
  });

  it('reports a failed teardown step over the forward failure that started it', () => {
    const line = runSummaryLine(
      run({
        status: 'failed',
        finished_at: '2026-01-01T00:02:00Z',
        hosts: [
          host('a', installing('failed'), {
            rollback_steps: [
              step('stop_service', 'failed'),
              step('remove_data', 'pending'),
            ],
          }),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Rollback failed at step 1 of 2: Stopping mongod on node-a, after 2m'
    );
  });

  it('says where an aborted run stopped', () => {
    const line = runSummaryLine(
      run({
        status: 'rolled_back',
        finished_at: '2026-01-01T00:01:00Z',
        cancel_requested: true,
        hosts: [host('a', installing('pending'))],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Stopped at step 2 of 4: Installing packages on 1 node, after 1m'
    );
  });

  it('keeps counting while monitoring is confirmed after PMM Extensions finished', () => {
    const line = runSummaryLine(
      run({
        status: 'succeeded',
        finished_at: '2026-01-01T00:01:00Z',
        run_steps: [step('rs_initiate', 'succeeded')],
        hosts: [
          host('a', installing('succeeded'), {
            finalize_steps: [
              step('enable_auth', 'succeeded'),
              step('confirm_monitoring', 'running'),
            ],
          }),
        ],
      }),
      name,
      NOW
    );

    expect(line).toBe(
      'Step 5 of 5: Confirming PMM monitoring on 1 node, running for 3m 12s'
    );
  });

  it('has no line for a run that succeeded', () => {
    expect(
      runSummaryLine(
        run({
          status: 'succeeded',
          hosts: [host('a', installing('succeeded'))],
        }),
        name,
        NOW
      )
    ).toBeNull();
  });
});

describe('runHasFailure', () => {
  it('is true once any step has failed, before the run status catches up', () => {
    expect(
      runHasFailure(run({ hosts: [host('a', installing('failed'))] }))
    ).toBe(true);
  });

  it('is true for a failed teardown step with no forward step failed', () => {
    expect(
      runHasFailure(
        run({
          cancel_requested: true,
          hosts: [
            host('a', installing('pending'), {
              rollback_steps: [step('stop_service', 'failed')],
            }),
          ],
        })
      )
    ).toBe(true);
  });

  it('is false for a run going normally', () => {
    expect(
      runHasFailure(run({ hosts: [host('a', installing('running'))] }))
    ).toBe(false);
  });
});
