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

/**
 * The one HTTP client both halves of OM use, and the wire facts they share.
 *
 * `/v1/om` serves two very different things - pmm-managed's own topology document and
 * SEP's estate, proxied - but they arrive over the same origin with the same auth and
 * the same error envelope, so the transport belongs in one place rather than in
 * whichever hook file grew it first.
 *
 * Two consequences of pmm-managed serving both, and both are simplifications:
 *
 * - **No bearer.** `/v1` is PMM's own origin and authorises on the Grafana session
 *   cookie, so there is no token to mint and no `SepAuthGate` to wait for. `fetch` with
 *   `credentials: 'same-origin'` is the whole auth story. Before the proxy, the estate
 *   was read from SEP directly with a bearer minted from the PMM session, which meant a
 *   second HTTP client and a page that failed closed when SEP was unwell.
 * - **snake_case survives.** gRPC-Gateway is configured with `UseProtoNames` and
 *   `EmitUnpopulated`, and `types.ts` describes exactly that shape.
 */

import type {
  OmBootstrapHost,
  OmGetBootstrapRunResponse,
  OmTopologyRunStatus,
} from './types';

const OM_BASE = '/v1/om';

/**
 * A failed request, carrying the status the caller has to branch on.
 *
 * The status has to survive onto the error: a 409 from either trigger is an expected
 * outcome a button renders differently, not a failure.
 */
export class OmApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'OmApiError';
    this.status = status;
  }
}

/**
 * One request against pmm-managed.
 *
 * Deliberately `fetch` rather than an axios instance: PMM's own axios client runs
 * `axios-case-converter` and would camelCase the response out from under `types.ts`. A
 * bare same-origin fetch is both smaller and the only one that leaves the wire shape
 * alone.
 */
export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${OM_BASE}${path}`, {
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!response.ok) {
    // The gateway reports failures as {code, message, details}; the message is the only
    // part worth showing, and its absence should not mask the status.
    const body = (await response.json().catch(() => null)) as {
      message?: string;
    } | null;
    throw new OmApiError(
      response.status,
      body?.message ?? `Request failed with ${response.status}`
    );
  }
  return (await response.json()) as T;
}

/**
 * True while a run has not reached a terminal status.
 *
 * Covers both kinds. A collection pass and an inventory refresh are very different
 * operations, but they report the same `RunStatus` enum, so asking "is it still going"
 * is one question and this is the one place that answers it. The inventory side used to
 * carry a byte-identical `isRefreshActive`, which meant two functions to keep in step
 * with one wire contract.
 *
 * Typed on the union rather than on `string`, because the compiler is the only thing
 * that would have caught this comparison going stale when the wire values changed.
 */
export function isRunActive(status: OmTopologyRunStatus | undefined): boolean {
  return status === 'RUN_STATUS_RUNNING';
}

/**
 * True while a bootstrap run is still worth polling: SEP's own status has not
 * reached a terminal value, or it has but a host's confirm_monitoring step
 * (PMM's own, appended to finalize_steps -- see its own proto comment) has not
 * caught up yet.
 *
 * SEP's status alone understates "done": om_bootstrap marks a run succeeded the
 * moment every step it dispatched has succeeded, before PMM's inventory sweep
 * has necessarily noticed the newly-registered service. Stopping on SEP's status
 * alone left confirm_monitoring showing "Running" forever once nothing was
 * polling to see it flip to "Succeeded" a few seconds later.
 */
export function isBootstrapRunActive(
  run: Pick<OmGetBootstrapRunResponse, 'status' | 'hosts'> | undefined
): boolean {
  if (!run) {
    return false;
  }
  if (run.status === 'running') {
    return true;
  }
  if (run.status !== 'succeeded') {
    return false;
  }
  return run.hosts.some((host) =>
    host.finalize_steps.some(
      (step) => step.name === 'confirm_monitoring' && step.status === 'running'
    )
  );
}

/**
 * True once any of a host's rollback steps has been dispatched.
 *
 * Every rollback step is planned `pending` up front alongside the host's
 * forward steps (om_bootstrap's own HostBootstrapState doc comment) and stays
 * that way for the entire life of a run that never needed rollback - so "not
 * still all pending" is "rollback actually started," the same test
 * PMM-15347's stepper itself uses (bootstrap_decision.go's hostRollbackStarted).
 */
export function isHostRollingBack(host: OmBootstrapHost): boolean {
  return host.rollback_steps.some((step) => step.status !== 'pending');
}
