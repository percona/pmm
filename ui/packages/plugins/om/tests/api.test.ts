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
import { bootstrapRunDisplayStatus, isBootstrapRunActive } from '../src/api';
import type { OmBootstrapStep, OmGetBootstrapRunResponse } from '../src/types';

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
    cancel_requested: false,
    ...overrides,
  };
}

describe('isBootstrapRunActive', () => {
  it('is false with no run yet', () => {
    expect(isBootstrapRunActive(undefined)).toBe(false);
  });

  it('is true while SEP itself still reports the run running', () => {
    expect(isBootstrapRunActive(run({ status: 'running' }))).toBe(true);
  });

  it('is false once failed or rolled back, regardless of any step', () => {
    expect(isBootstrapRunActive(run({ status: 'failed' }))).toBe(false);
    expect(isBootstrapRunActive(run({ status: 'rolled_back' }))).toBe(false);
  });

  // The regression this exists for: SEP marks a run succeeded the moment every
  // step it dispatched has succeeded, before PMM's own inventory sweep has
  // necessarily noticed the registered service. Stopping on SEP's status alone
  // left confirm_monitoring stuck showing "Running" forever once nothing was
  // polling to see it flip to "Succeeded".
  it('stays true once succeeded while a host confirm_monitoring is still running', () => {
    const active = run({
      status: 'succeeded',
      hosts: [
        {
          host: 'n1',
          steps: [],
          rollback_steps: [],
          finalize_steps: [
            step('enable_auth', 'succeeded'),
            step('confirm_monitoring', 'running'),
          ],
        },
      ],
    });

    expect(isBootstrapRunActive(active)).toBe(true);
  });

  it('is false once succeeded and every host confirm_monitoring has succeeded', () => {
    const done = run({
      status: 'succeeded',
      hosts: [
        {
          host: 'n1',
          steps: [],
          rollback_steps: [],
          finalize_steps: [step('confirm_monitoring', 'succeeded')],
        },
        {
          host: 'n2',
          steps: [],
          rollback_steps: [],
          finalize_steps: [step('confirm_monitoring', 'succeeded')],
        },
      ],
    });

    expect(isBootstrapRunActive(done)).toBe(false);
  });
});

describe('bootstrapRunDisplayStatus', () => {
  it('passes through a non-succeeded status unchanged', () => {
    expect(bootstrapRunDisplayStatus(run({ status: 'running' }))).toBe(
      'running'
    );
    expect(bootstrapRunDisplayStatus(run({ status: 'failed' }))).toBe('failed');
    expect(bootstrapRunDisplayStatus(run({ status: 'rolled_back' }))).toBe(
      'rolled_back'
    );
  });

  // The regression this exists for: the Automations page and the wizard both
  // showed a green "Succeeded" badge the moment SEP's own status flipped,
  // while confirm_monitoring was still visibly "Running" a step below it --
  // the same premature-done gap isBootstrapRunActive fixes for polling, but
  // for the badge a reader actually sees.
  it('reads as running while succeeded but a host confirm_monitoring is still running', () => {
    const stillConfirming = run({
      status: 'succeeded',
      hosts: [
        {
          host: 'n1',
          steps: [],
          rollback_steps: [],
          finalize_steps: [
            step('enable_auth', 'succeeded'),
            step('confirm_monitoring', 'running'),
          ],
        },
      ],
    });

    expect(bootstrapRunDisplayStatus(stillConfirming)).toBe('running');
  });

  it('reads as succeeded once every host confirm_monitoring has succeeded', () => {
    const done = run({
      status: 'succeeded',
      hosts: [
        {
          host: 'n1',
          steps: [],
          rollback_steps: [],
          finalize_steps: [step('confirm_monitoring', 'succeeded')],
        },
      ],
    });

    expect(bootstrapRunDisplayStatus(done)).toBe('succeeded');
  });
});
