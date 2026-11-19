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
  bootstrapRunDisplayStatus,
  isConfirmingMonitoring,
  isHostRollingBack,
} from './api';
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
 * How many dispatches pmm-managed gives a step before its failure is permanent:
 * `bootstrapMaxAttempts` in managed/services/om/bootstrap_decision.go. A failed step
 * with an attempt left is about to be dispatched again, so it is not a failure yet.
 */
const BOOTSTRAP_MAX_ATTEMPTS = 2;

/**
 * Which failed steps of a run are failures for good, as opposed to about to be
 * retried, for the forward steps and the rollback steps.
 *
 * A step is out of attempts at BOOTSTRAP_MAX_ATTEMPTS, and every step is once the run
 * has ended. A forward step is also final once the run is rolling back: the stepper
 * dispatches no forward step from then on. That covers the one step not retried at
 * all, pre_check, whose `retryable: false` is not on the wire - its failure starts
 * the rollback on the stepper's next tick.
 */
function finalFailures(run: OmGetBootstrapRunResponse) {
  const ended = bootstrapRunDisplayStatus(run) !== 'running';
  const isFinal = (over: boolean) => (step: OmBootstrapStep) =>
    step.status === 'failed' &&
    (over || step.attempt_count >= BOOTSTRAP_MAX_ATTEMPTS);
  return {
    forward: isFinal(ended || run.hosts.some(isHostRollingBack)),
    rollback: isFinal(ended),
  };
}

/** The nodes a phase's steps matching `pick` are on, by name. */
function failedNodes(
  phase: RunPhase,
  pick: (step: OmBootstrapStep) => boolean,
  nodeName: (nodeId: string) => string
): string {
  return phase.steps
    .filter(({ step }) => pick(step))
    .map(({ node }) => nodeName(node))
    .join(', ');
}

/**
 * " failed on node-a, retrying", for the first phase with a step about to be
 * dispatched again, along with that phase's index. Null when there is none.
 */
function retrying(
  phases: RunPhase[],
  isFinal: (step: OmBootstrapStep) => boolean,
  nodeName: (nodeId: string) => string
) {
  const isRetried = (step: OmBootstrapStep) =>
    step.status === 'failed' && !isFinal(step);
  const index = phases.findIndex((phase) =>
    phase.steps.some(({ step }) => isRetried(step))
  );
  if (index === -1) {
    return null;
  }
  const phase = phases[index];
  const where = phase.perNode
    ? ` on ${failedNodes(phase, isRetried, nodeName)}`
    : '';
  return {
    index,
    what: `${bootstrapStepLabel(phase.name)} failed${where}, retrying`,
  };
}

/**
 * Whether anything in this run failed for good or it ended without succeeding.
 *
 * Decides whether the step matrix opens by itself: a reader of a failed run needs the
 * failing cell, and should not have to find a toggle to get to it. A failure about
 * to be retried does not count, or a transient one would open the matrix and leave
 * it open after the retry succeeded.
 */
export function runHasFailure(run: OmGetBootstrapRunResponse): boolean {
  const status = bootstrapRunDisplayStatus(run);
  if (status === 'failed' || status === 'rolled_back') {
    return true;
  }
  const isFinal = finalFailures(run);
  return (
    [
      ...run.run_steps,
      ...run.hosts.flatMap((host) => [...host.steps, ...host.finalize_steps]),
    ].some(isFinal.forward) ||
    // A teardown step out of retries can leave the run waiting on an operator
    // with no forward step failed - an aborted run, for one.
    run.hosts.some((host) => host.rollback_steps.some(isFinal.rollback))
  );
}

/**
 * A run's progress as one line: which step of how many, what it is doing, and for how
 * long - "Step 4 of 11: Installing packages on 3 nodes, running for 3m 12s".
 *
 * Built from the steps alone. Per-step timestamps do not reach the UI (PMM-15667), so
 * the elapsed time is the run's own, and there is no estimate of what is left.
 *
 * `nodeName` resolves a node id to the name a reader knows it by, for the one case
 * that names a node: a step that failed or is about to be retried. A `now` of null leaves the elapsed time
 * off, which is always a suffix of the full line. Returns null for a run that
 * succeeded, which ends on its completion card instead.
 */
export function runSummaryLine(
  run: OmGetBootstrapRunResponse,
  nodeName: (nodeId: string) => string,
  now: number | null = Date.now()
): string | null {
  const status = bootstrapRunDisplayStatus(run);
  if (status === 'succeeded') {
    return null;
  }
  // finished_at is set as soon as PMM Extensions is done, while confirm_monitoring
  // can keep the run going, and nothing on the wire says when that ends. The clock
  // stops at finished_at so it never runs backwards once confirmation lands, and
  // the confirming window shows no time rather than a stopped "running for".
  const confirming = isConfirmingMonitoring(run);
  const elapsed =
    now === null || confirming
      ? ''
      : formatRunElapsed(run.started_at, run.finished_at, now);

  const isFinal = finalFailures(run);
  const rollbackPhases = perNodePhases(run, 'rollback_steps');
  // Ahead of every other reading, a forward failure included: a teardown that failed
  // can leave a node half-installed, which outranks why the rollback started (the
  // run's own error still says that). Without it, position() counts the failed step
  // as not done yet and the line reads as a rollback still going.
  const rollbackFailedAt = rollbackPhases.findIndex((phase) =>
    phase.steps.some(({ step }) => isFinal.rollback(step))
  );
  if (rollbackFailedAt !== -1) {
    const phase = rollbackPhases[rollbackFailedAt];
    const timing = elapsed
      ? `, ${status === 'running' ? 'running for' : 'after'} ${elapsed}`
      : '';
    return `Rollback failed at step ${rollbackFailedAt + 1} of ${
      rollbackPhases.length
    }: ${bootstrapStepLabel(phase.name)} on ${failedNodes(
      phase,
      isFinal.rollback,
      nodeName
    )}${timing}`;
  }

  if (status === 'running' && run.hosts.some(isHostRollingBack)) {
    const phases = rollbackPhases;
    const retry = retrying(phases, isFinal.rollback, nodeName);
    const { index, phase } = retry
      ? { index: retry.index, phase: undefined }
      : position(phases);
    const what =
      retry?.what ??
      (phase
        ? `${bootstrapStepLabel(phase.name)}${nodeScope(phase)}`
        : 'Finishing');
    return `Rolling back, step ${index + 1} of ${phases.length}: ${what}${
      elapsed ? `, running for ${elapsed}` : ''
    }`;
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
    phase.steps.some(({ step }) => isFinal.forward(step))
  );
  if (failedAt !== -1) {
    const phase = phases[failedAt];
    const where = phase.perNode
      ? ` on ${failedNodes(phase, isFinal.forward, nodeName)}`
      : '';
    return `Failed at step ${failedAt + 1} of ${phases.length}: ${bootstrapStepLabel(
      phase.name
    )}${where}${elapsed ? `, after ${elapsed}` : ''}`;
  }

  // Only while running: once the run has ended every failure is final, above.
  const retry = retrying(phases, isFinal.forward, nodeName);
  const { index, phase } = retry
    ? { index: retry.index, phase: undefined }
    : position(phases);
  const step = `step ${index + 1} of ${phases.length}`;
  const what =
    retry?.what ??
    (phase
      ? `${bootstrapStepLabel(phase.name)}${nodeScope(phase)}`
      : 'Finishing');

  if (status !== 'running') {
    // Ended without a failed step: an abort, rolled back from wherever it had got to.
    return `Stopped at ${step}: ${what}${elapsed ? `, after ${elapsed}` : ''}`;
  }
  return `${step[0].toUpperCase()}${step.slice(1)}: ${what}${
    elapsed ? `, running for ${elapsed}` : ''
  }`;
}

/** About how long a step's reason may be before it is cut at a word. */
const EXCERPT_CHARS = 200;

/** A failed step's detail split for reading: what broke, the rest, and its logs. */
interface StepFailureReason {
  /** The last line of the detail, where mongod, apt and dnf say what broke. */
  excerpt: string;
  /** The whole detail without its task history suffix, newlines kept. */
  detail: string;
  taskHistoryId: string | null;
}

/**
 * Split a step's detail: PMM Extensions writes a reason, then the end of the step's
 * output, then " (task history N)", and only that suffix has a fixed form.
 */
export function stepFailureReason(text: string): StepFailureReason {
  const suffix = /\s*\(task history (\d+)\)\s*$/.exec(text);
  const detail = (suffix ? text.slice(0, suffix.index) : text).trim();
  const lines = detail
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
  const last = lines[lines.length - 1] ?? detail;
  return {
    excerpt:
      last.length <= EXCERPT_CHARS
        ? last
        : `…${cutToWord(last.slice(-EXCERPT_CHARS))}`,
    detail,
    taskHistoryId: suffix?.[1] ?? null,
  };
}

/** `text` from its first word boundary on, unless it has none. */
function cutToWord(text: string): string {
  const space = text.search(/\s/);
  return space === -1 ? text : text.slice(space + 1);
}

/** A failed step with its reason, and whether it is about to be retried. */
export interface StepFailure {
  step: string;
  /** The node's name, or empty for a run-level step. */
  node: string;
  attempt: number;
  final: boolean;
  rollback: boolean;
  reason: StepFailureReason;
}

/**
 * The failed steps of a run that say why, forward then rollback, leaving out monitoring
 * confirmation, which runs once the install has succeeded.
 */
export function runFailures(
  run: OmGetBootstrapRunResponse,
  nodeName: (nodeId: string) => string
): StepFailure[] {
  const isFinal = finalFailures(run);
  const collect = (
    phases: RunPhase[],
    rollback: boolean,
    final: (step: OmBootstrapStep) => boolean
  ): StepFailure[] =>
    phases.flatMap((phase) =>
      phase.steps
        .filter(({ step }) => step.status === 'failed' && step.detail)
        .map(({ node, step }) => ({
          step: phase.name,
          node: phase.perNode ? nodeName(node) : '',
          attempt: step.attempt_count,
          final: final(step),
          rollback,
          reason: stepFailureReason(step.detail ?? ''),
        }))
    );
  const forward = forwardPhases(run).filter(
    (phase) => phase.name !== 'confirm_monitoring'
  );
  return [
    ...collect(forward, false, isFinal.forward),
    ...collect(perNodePhases(run, 'rollback_steps'), true, isFinal.rollback),
  ];
}

/**
 * Why a run failed or is failing, one line per reason with the nodes it holds on: a
 * failed rollback first, then the install's, or its first failed attempt.
 */
export function runFailureLines(failures: StepFailure[]): string[] {
  const lines = (label: string, picked: StepFailure[]) => {
    const nodes = new Map<string, string[]>();
    for (const { reason, node } of picked) {
      nodes.set(reason.excerpt, [...(nodes.get(reason.excerpt) ?? []), node]);
    }
    return [...nodes].map(([excerpt, on]) => {
      const named = on.filter(Boolean);
      return `${label}: ${excerpt}${named.length ? ` (on ${named.join(', ')})` : ''}`;
    });
  };
  const said = failures.filter((failure) => failure.reason.excerpt);
  const forward = said.filter((failure) => !failure.rollback);
  const finalForward = forward.filter((failure) => failure.final);
  return [
    ...lines(
      'Rollback failed',
      said.filter((failure) => failure.rollback && failure.final)
    ),
    ...(finalForward.length
      ? lines('Why', finalForward)
      : forward.length
        ? lines(`Attempt ${forward[0].attempt} failed`, forward)
        : []),
  ];
}

/**
 * Why a run failed or is failing: the run's own error first when an operator aborted
 * it or no forward step says why, then each step's reason.
 */
export function runReasons(
  run: OmGetBootstrapRunResponse,
  failures: StepFailure[]
): string[] {
  const lines = runFailureLines(failures);
  const forwardSaid = failures.some(
    (failure) => !failure.rollback && failure.reason.excerpt
  );
  return run.error && (run.cancel_requested || !forwardSaid)
    ? [run.error, ...lines]
    : lines;
}
