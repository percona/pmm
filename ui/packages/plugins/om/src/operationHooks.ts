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
 * The managed-cluster operation lifecycle, served by pmm-managed at `/v1/om/operations`,
 * and the uninstall trigger + readiness report the confirm flow drives.
 *
 * A different shape from the bootstrap run in `inventoryHooks.ts`: an operation is not a
 * run of steps a page watches to completion but a state machine an operator drives -
 * trigger to `checking`, poll to `awaiting_confirmation`, confirm to `running`. The
 * transport (`request`) and the snake_case wire are shared with the rest of OM.
 */

import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { request } from './api';
import type {
  OmGetOperationResponse,
  OmOperation,
  OmOperationStatus,
  OmTriggerUninstallResponse,
  OmUninstallReport,
} from './types';

const OPERATIONS_KEY = ['om', 'operations'] as const;
const HOSTS_KEY = ['om', 'inventory', 'hosts'] as const;
const SERVICES_KEY = ['om', 'inventory', 'services'] as const;

/** Poll cadence while an operation is progressing (ms). */
const OPERATION_POLL_MS = 3000;

/** Idle cadence for the operation list (ms): slow enough to notice a trigger from elsewhere. */
const OPERATION_IDLE_MS = 30000;

/**
 * Statuses in which an operation holds its cluster: another operation cannot start, so a
 * trigger stays visible but disabled. `awaiting_confirmation` is included - a dialog left
 * open still owns the cluster.
 */
const BUSY_STATUSES: readonly OmOperationStatus[] = [
  'checking',
  'awaiting_confirmation',
  'queued',
  'running',
  'cancelling',
];

/**
 * Statuses in which an operation changes on its own, so a watcher keeps polling.
 * `awaiting_confirmation` is not one: it waits on the operator, not the server.
 */
const PROGRESSING_STATUSES: readonly OmOperationStatus[] = [
  'checking',
  'queued',
  'running',
  'cancelling',
];

/**
 * Statuses in which an operation holds its cluster but has not begun tearing it down, so its
 * dialog can be reopened to confirm or cancel it rather than leaving the action a dead end.
 */
const PENDING_STATUSES: readonly OmOperationStatus[] = [
  'checking',
  'awaiting_confirmation',
  'queued',
];

/** Whether an operation still holds its cluster, for gating a second trigger. */
export function isOperationBusy(op: Pick<OmOperation, 'status'>): boolean {
  return BUSY_STATUSES.includes(op.status);
}

/** Whether an operation holds its cluster but can still be reopened to confirm or cancel. */
export function isOperationPending(op: Pick<OmOperation, 'status'>): boolean {
  return PENDING_STATUSES.includes(op.status);
}

/** Whether an operation is still moving on its own, for a poll cadence. */
export function isOperationProgressing(
  op: Pick<OmOperation, 'status'>
): boolean {
  return PROGRESSING_STATUSES.includes(op.status);
}

/**
 * Every managed-cluster operation, newest first, from `GET /v1/om/operations`.
 *
 * Read whole rather than per cluster so one query gates every row's trigger: a host
 * carries services from more than one cluster, and asking per cluster would be a request
 * per menu item. Polls fast while any operation is progressing and slowly otherwise, so a
 * trigger started on another page appears here too.
 */
export function useOmOperations() {
  return useQuery<OmOperation[]>({
    queryKey: OPERATIONS_KEY,
    queryFn: async () => {
      const { operations } = await request<{ operations?: OmOperation[] }>(
        '/operations'
      );
      return operations ?? [];
    },
    refetchInterval: (query) =>
      (query.state.data ?? []).some(isOperationProgressing)
        ? OPERATION_POLL_MS
        : OPERATION_IDLE_MS,
    // See useOmInventoryHosts: a backgrounded tab otherwise pauses polling with nothing
    // to unstick it, freezing a cluster on a "removing" badge after the teardown landed.
    refetchIntervalInBackground: true,
    placeholderData: keepPreviousData,
  });
}

/**
 * One operation and its dispatched steps, from `GET /v1/om/operations/{id}`.
 *
 * Polls while the operation progresses and stops once it settles or awaits the operator -
 * an `awaiting_confirmation` operation does not change until a confirm or cancel moves it,
 * and the confirm invalidates this query itself.
 */
export function useOmOperation(operationId: string | null) {
  return useQuery<OmGetOperationResponse>({
    queryKey: [...OPERATIONS_KEY, operationId],
    enabled: Boolean(operationId),
    queryFn: () =>
      request<OmGetOperationResponse>(
        `/operations/${encodeURIComponent(operationId ?? '')}`
      ),
    refetchInterval: (query) => {
      const op = query.state.data?.operation;
      return op && isOperationProgressing(op) ? OPERATION_POLL_MS : false;
    },
    refetchIntervalInBackground: true,
  });
}

/**
 * An uninstall operation's readiness report, from
 * `GET /v1/om/operations/{id}/uninstallReport`.
 *
 * Polls itself to a verdict: while any member is still `checking` the readiness scan is in
 * flight, so it refetches; once every member has a verdict (ready, a refusal, or drift) it
 * stops, since the report then only changes on a confirm or a reseal, which invalidate it.
 */
export function useUninstallReport(
  operationId: string | null,
  enabled = true,
  operationVersion?: number
) {
  return useQuery<OmUninstallReport>({
    // Keyed on the operation's version too: a transition can finish after the members have
    // already read as ready, and polling on members alone would never fetch its final report.
    queryKey: [
      ...OPERATIONS_KEY,
      operationId,
      'uninstall-report',
      operationVersion,
    ],
    enabled: enabled && Boolean(operationId),
    queryFn: async () => {
      const { report } = await request<{ report: OmUninstallReport }>(
        `/operations/${encodeURIComponent(operationId ?? '')}/uninstallReport`
      );
      return report;
    },
    refetchInterval: (query) =>
      (query.state.data?.members ?? []).some(
        (member) => member.readiness === 'checking'
      )
        ? OPERATION_POLL_MS
        : false,
    refetchIntervalInBackground: true,
    // The query key carries the operation version, so a version bump re-keys it; keep the last
    // report on screen while it refetches instead of blanking to the checking state.
    placeholderData: keepPreviousData,
  });
}

/**
 * Start uninstalling a managed cluster, from `POST /v1/om/clusters/{id}:uninstall`.
 *
 * Returns the checking operation's id; the dialog then polls {@link useOmOperation} and
 * {@link useUninstallReport}. Idempotent per cluster server-side - a second trigger for a
 * cluster that already has an open uninstall returns the same operation id.
 */
export function useTriggerUninstall() {
  const queryClient = useQueryClient();
  return useMutation<OmTriggerUninstallResponse, Error, string>({
    mutationFn: (clusterId) =>
      request<OmTriggerUninstallResponse>(
        `/clusters/${encodeURIComponent(clusterId)}:uninstall`,
        { method: 'POST' }
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: OPERATIONS_KEY });
    },
  });
}

/** An operation action fenced by the version the caller read, with an optional confirm phrase. */
interface OperationAction {
  operationId: string;
  version: number;
  confirmPhrase?: string;
}

/**
 * Invalidate every query a transition touches: the operation, its report, the list that
 * gates triggers, and the estate, which a teardown changes as members are removed.
 */
function useInvalidateAfterTransition() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: OPERATIONS_KEY });
    queryClient.invalidateQueries({ queryKey: HOSTS_KEY });
    queryClient.invalidateQueries({ queryKey: SERVICES_KEY });
  };
}

/**
 * Confirm an operation awaiting confirmation, from `POST /v1/om/operations/{id}:confirm`.
 *
 * Fenced by `version`: a rescan or any change since the review has bumped it and the
 * confirm moves nothing (409). The sealed confirm phrase is validated server-side.
 */
export function useConfirmOperation() {
  const invalidate = useInvalidateAfterTransition();
  return useMutation<{ operation: OmOperation }, Error, OperationAction>({
    mutationFn: ({ operationId, version, confirmPhrase }) =>
      request<{ operation: OmOperation }>(
        `/operations/${encodeURIComponent(operationId)}:confirm`,
        {
          method: 'POST',
          body: JSON.stringify({
            version,
            ...(confirmPhrase === undefined
              ? {}
              : { confirm_phrase: confirmPhrase }),
          }),
        }
      ),
    onSuccess: invalidate,
  });
}

/**
 * Cancel an operation, from `POST /v1/om/operations/{id}:cancel`.
 *
 * Dismissing an open trigger before it runs cancels the `checking`/`awaiting_confirmation`
 * operation so it stops holding the cluster; a running one of a cancellable kind goes to
 * cancelling instead. Fenced by `version`.
 */
export function useCancelOperation() {
  const invalidate = useInvalidateAfterTransition();
  return useMutation<{ operation: OmOperation }, Error, OperationAction>({
    mutationFn: ({ operationId, version }) =>
      request<{ operation: OmOperation }>(
        `/operations/${encodeURIComponent(operationId)}:cancel`,
        { method: 'POST', body: JSON.stringify({ version }) }
      ),
    onSuccess: invalidate,
  });
}

/** Resume a failed, resumable operation, from `POST /v1/om/operations/{id}:resume`. */
export function useResumeOperation() {
  const invalidate = useInvalidateAfterTransition();
  return useMutation<{ operation: OmOperation }, Error, OperationAction>({
    mutationFn: ({ operationId, version }) =>
      request<{ operation: OmOperation }>(
        `/operations/${encodeURIComponent(operationId)}:resume`,
        { method: 'POST', body: JSON.stringify({ version }) }
      ),
    onSuccess: invalidate,
  });
}

/**
 * Abandon a failed operation, from `POST /v1/om/operations/{id}:abandon`.
 *
 * Resolves one that will not be resumed: releases its locks, records its residual hosts and
 * leaves it abandoned. The confirm phrase is validated server-side, the same as a confirm.
 */
export function useAbandonOperation() {
  const invalidate = useInvalidateAfterTransition();
  return useMutation<{ operation: OmOperation }, Error, OperationAction>({
    mutationFn: ({ operationId, version, confirmPhrase }) =>
      request<{ operation: OmOperation }>(
        `/operations/${encodeURIComponent(operationId)}:abandon`,
        {
          method: 'POST',
          body: JSON.stringify({
            version,
            ...(confirmPhrase === undefined
              ? {}
              : { confirm_phrase: confirmPhrase }),
          }),
        }
      ),
    onSuccess: invalidate,
  });
}

/**
 * Adopt the live plan of a drift-blocked uninstall, from
 * `POST /v1/om/operations/{id}:confirmLivePlan`.
 *
 * Re-seals the plan to the hosts' live package sets, bumps the plan revision and returns the
 * operation to `checking` so readiness re-runs. Fenced by `version`.
 */
export function useConfirmUninstallLivePlan() {
  const invalidate = useInvalidateAfterTransition();
  return useMutation<{ operation: OmOperation }, Error, OperationAction>({
    mutationFn: ({ operationId, version }) =>
      request<{ operation: OmOperation }>(
        `/operations/${encodeURIComponent(operationId)}:confirmLivePlan`,
        { method: 'POST', body: JSON.stringify({ version }) }
      ),
    onSuccess: invalidate,
  });
}
