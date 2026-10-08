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

import { bootstrapRunDisplayStatus, isHostRollingBack } from './api';
import { bootstrapStepLabel } from './constants';
import { formatRunElapsed, pluralize } from './format';
import type { OmBootstrapStep, OmGetBootstrapRunResponse } from './types';

/**
 * One step of a run as a whole: the same-named step on every node it runs on.
 *
 * `perNode` is false for a run-level step, which has one shared outcome rather than
 * one per node - the summary then says nothing about how many nodes it is on.
 */
interface RunPhase {
  name: string;
  perNode: boolean;
  /** The step's record on each node, keyed by node id, or the run's own once. */
  steps: { node: string; step: OmBootstrapStep }[];
}

type StepList = 'steps' | 'finalize_steps' | 'rollback_steps';

function perNodePhases(
  run: OmGetBootstrapRunResponse,
  list: StepList
): RunPhase[] {
  const seed = run.hosts[0];
  if (!seed) {
    return [];
  }
  return seed[list].map(({ name }) => ({
    name,
    perNode: true,
    steps: run.hosts.flatMap((host) => {
      const step = host[list].find((candidate) => candidate.name === name);
      return step ? [{ node: host.host, step }] : [];
    }),
  }));
}

/**
 * Every forward step of a run, in the order it runs: each node's own steps, then the
 * run-level ones, then each node's finalize steps. The same order the matrix shows.
 */
function forwardPhases(run: OmGetBootstrapRunResponse): RunPhase[] {
  return [
    ...perNodePhases(run, 'steps'),
    ...run.run_steps.map((step) => ({
      name: step.name,
      perNode: false,
      steps: [{ node: '', step }],
    })),
    ...perNodePhases(run, 'finalize_steps'),
  ];
}

function isDone(phase: RunPhase): boolean {
  return phase.steps.every(
    ({ step }) => step.status === 'succeeded' || step.status === 'skipped'
  );
}

/** Where a list of phases has got to: the first one not finished on every node. */
function position(phases: RunPhase[]) {
  const index = phases.findIndex((phase) => !isDone(phase));
  return {
    index: index === -1 ? phases.length - 1 : index,
    phase: index === -1 ? undefined : phases[index],
  };
}

/** " on 3 nodes", counting the nodes the step is still going on. */
function nodeScope(phase: RunPhase): string {
  if (!phase.perNode) {
    return '';
  }
  const running = phase.steps.filter(({ step }) => step.status === 'running');
  const remaining = running.length
    ? running
    : phase.steps.filter(
        ({ step }) => step.status !== 'succeeded' && step.status !== 'skipped'
      );
  return ` on ${remaining.length} ${pluralize(remaining.length, 'node')}`;
}

/**
 * Whether anything in this run failed or it ended without succeeding.
 *
 * Decides whether the step matrix opens by itself: a reader of a failed run needs the
 * failing cell, and should not have to find a toggle to get to it.
 */
export function runHasFailure(run: OmGetBootstrapRunResponse): boolean {
  const status = bootstrapRunDisplayStatus(run);
  if (status === 'failed' || status === 'rolled_back') {
    return true;
  }
  return [
    ...run.run_steps,
    ...run.hosts.flatMap((host) => [
      ...host.steps,
      ...host.finalize_steps,
      // A teardown step out of retries can leave the run waiting on an operator
      // with no forward step failed - an aborted run, for one.
      ...host.rollback_steps,
    ]),
  ].some((step) => step.status === 'failed');
}

/**
 * A run's progress as one line: which step of how many, what it is doing, and for how
 * long - "Step 4 of 11: Installing packages on 3 nodes, running for 3m 12s".
 *
 * Built from the steps alone. Per-step timestamps do not reach the UI (PMM-15667), so
 * the elapsed time is the run's own, and there is no estimate of what is left.
 *
 * `nodeName` resolves a node id to the name a reader knows it by, for the one case
 * that names a node: the step that failed. Returns null for a run that succeeded,
 * which ends on its completion card instead.
 */
export function runSummaryLine(
  run: OmGetBootstrapRunResponse,
  nodeName: (nodeId: string) => string,
  now: number = Date.now()
): string | null {
  const status = bootstrapRunDisplayStatus(run);
  if (status === 'succeeded') {
    return null;
  }
  // finished_at is set as soon as PMM Extensions is done, while confirm_monitoring
  // still keeps the run going: the clock runs on until the display status ends.
  const elapsed = formatRunElapsed(
    run.started_at,
    status === 'running' ? null : run.finished_at,
    now
  );

  if (status === 'running' && run.hosts.some(isHostRollingBack)) {
    const phases = perNodePhases(run, 'rollback_steps');
    const { index, phase } = position(phases);
    const label = phase ? bootstrapStepLabel(phase.name) : 'Finishing';
    return `Rolling back, step ${index + 1} of ${phases.length}: ${label}${
      phase ? nodeScope(phase) : ''
    }${elapsed ? `, running for ${elapsed}` : ''}`;
  }

  const phases = forwardPhases(run);
  // Nothing dispatched yet: the stepper picks a new run up on its next tick, and until
  // then "step 1" would read as a run stuck on its first step.
  const nothingDispatched = phases.every((phase) =>
    phase.steps.every(({ step }) => step.status === 'pending')
  );
  if (status === 'running' && nothingDispatched) {
    return elapsed ? `Starting, running for ${elapsed}` : 'Starting';
  }

  const failedAt = phases.findIndex((phase) =>
    phase.steps.some(({ step }) => step.status === 'failed')
  );
  if (failedAt !== -1) {
    const phase = phases[failedAt];
    const failedNodes = phase.steps
      .filter(({ step }) => step.status === 'failed')
      .map(({ node }) => node);
    const where = phase.perNode
      ? ` on ${failedNodes.map(nodeName).join(', ')}`
      : '';
    return `Failed at step ${failedAt + 1} of ${phases.length}: ${bootstrapStepLabel(
      phase.name
    )}${where}${elapsed ? `, after ${elapsed}` : ''}`;
  }

  const { index, phase } = position(phases);
  const step = `step ${index + 1} of ${phases.length}`;
  const what = phase
    ? `${bootstrapStepLabel(phase.name)}${nodeScope(phase)}`
    : 'Finishing';

  if (status !== 'running') {
    // Ended without a failed step: an abort, rolled back from wherever it had got to.
    return `Stopped at ${step}: ${what}${elapsed ? `, after ${elapsed}` : ''}`;
  }
  return `${step[0].toUpperCase()}${step.slice(1)}: ${what}${
    elapsed ? `, running for ${elapsed}` : ''
  }`;
}
