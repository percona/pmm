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
 * OM's estate, served by pmm-managed at `/v1/om/inventory`.
 *
 * A different source from `hooks.ts`, on the same client and the same origin. The
 * topology document is PMM's own derivation, rebuilt per request in about a tenth of a
 * second, and it never touches a host. The estate is what SEP's probe found by running
 * a payload on the hosts themselves.
 *
 * That difference is why the two are not merged into one hook file and, more
 * importantly, why the refresh here is nothing like the sync there. `POST /topology/runs`
 * recomputes a document from data PMM already holds and answers with a terminal status.
 * `POST /inventory/runs` dispatches a Nomad job per host and takes tens of seconds, so
 * it answers `running` and has to be polled. Two buttons that different must not look
 * alike, and keeping their hooks apart is the cheapest way to keep them from being
 * written alike.
 */

import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { request } from './hooks';
import type {
  OmInventoryHost,
  OmInventoryRun,
  OmInventoryRunAccepted,
  OmInventoryRunDetail,
  OmInventoryService,
  OmInventorySetting,
  OmTopologyRunStatus,
} from './types';

const hostsKey = ['om', 'inventory', 'hosts'] as const;
const servicesKey = ['om', 'inventory', 'services'] as const;
const runsKey = ['om', 'inventory', 'runs'] as const;
const configKey = ['om', 'inventory', 'config'] as const;

/** Poll cadence while a refresh is in flight (ms). */
const REFRESH_POLL_MS = 3000;

/**
 * Idle cadence for the estate (ms).
 *
 * Slower than the snapshot's 30s on purpose: rows only change when a sweep runs, and
 * the schedule's default is ten minutes. Polling faster would ask a question whose
 * answer cannot have changed.
 */
const ESTATE_POLL_MS = 60000;

/** Filters `GET /hosts` accepts. Unset means unfiltered, which is not the same as false. */
export interface OmHostFilters {
  /** Whether the host has at least one registered MongoDB service. */
  has_service?: boolean;
  /** Whether it is currently failing its probe. */
  failing?: boolean;
  /**
   * Whether an executor is matched for it at all.
   *
   * Not a client name: the app filters on presence, so `false` is "which machines can
   * nothing be dispatched to" rather than "which are served by nobody in particular".
   */
  executor?: boolean;
}

/**
 * Build the query string, omitting anything unset.
 *
 * An unset filter must not be sent as `false`: `has_service=false` is the question
 * "which hosts have no database", and sending it by accident would hide every host
 * that has one.
 */
function toQuery(filters: OmHostFilters): string {
  const params = new URLSearchParams();
  if (filters.has_service !== undefined) {
    params.set('has_service', String(filters.has_service));
  }
  if (filters.failing !== undefined) {
    params.set('failing', String(filters.failing));
  }
  if (filters.executor !== undefined) {
    params.set('executor', String(filters.executor));
  }
  const query = params.toString();
  return query ? `?${query}` : '';
}

/**
 * Every host OM keeps a row for, with the services on it.
 *
 * One request: `GET /hosts` nests each host's services, so the row expansion needs no
 * second fetch and there is no `/hosts/{id}/services` collection to call.
 */
export function useOmInventoryHosts(filters: OmHostFilters = {}) {
  const query = toQuery(filters);
  return useQuery<OmInventoryHost[]>({
    queryKey: [...hostsKey, query],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { hosts } = await request<{ hosts: OmInventoryHost[] }>(
        `/inventory/hosts${query}`
      );
      // The gateway always answers with an object, so the list arrives wrapped, and an
      // empty estate arrives as an absent key rather than an empty array.
      return hosts ?? [];
    },
    refetchInterval: ESTATE_POLL_MS,
  });
}

/**
 * Every service OM has probed, flat.
 *
 * Fetched separately from the hosts rather than flattened out of them: the Services
 * page joins these against PMM's snapshot and never needs the host nesting, and asking
 * for the hosts to get at their services would carry every host's document along with
 * it.
 */
export function useOmInventoryServices() {
  return useQuery<OmInventoryService[]>({
    queryKey: servicesKey,
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { services } = await request<{ services: OmInventoryService[] }>(
        '/inventory/services'
      );
      return services ?? [];
    },
    refetchInterval: ESTATE_POLL_MS,
  });
}

/**
 * True while a refresh has not reached a terminal status.
 *
 * Typed on the union rather than on `string`, because the compiler is the only thing
 * that would have caught this comparison going stale when the wire values changed.
 */
export function isRefreshActive(
  status: OmTopologyRunStatus | undefined
): boolean {
  return status === 'RUN_STATUS_RUNNING';
}

/**
 * Refresh history, newest first.
 *
 * Polls fast while the newest refresh is in flight and slowly otherwise, rather than
 * stopping: the schedule starts sweeps nobody here asked for, and a history that only
 * updated on reload could not show them.
 */
export function useOmInventoryRuns(limit = 25) {
  return useQuery<OmInventoryRun[]>({
    queryKey: [...runsKey, limit],
    queryFn: async () => {
      const { runs } = await request<{ runs: OmInventoryRun[] }>(
        `/inventory/runs?limit=${limit}`
      );
      return runs ?? [];
    },
    refetchInterval: (query) =>
      isRefreshActive(query.state.data?.[0]?.status)
        ? REFRESH_POLL_MS
        : ESTATE_POLL_MS,
  });
}

/**
 * Invalidate the estate when a refresh stops running.
 *
 * `useRefreshInventory`'s own `onSettled` cannot do this. It fires when the app
 * *accepts* the refresh, which is tens of seconds before any host has been probed, so
 * the rows it would refetch are the ones from before. This watches the newest run's
 * status and invalidates on the running -> terminal edge, which is when the estate
 * actually changed.
 *
 * Without it, hosts and services stay stale for up to ESTATE_POLL_MS after a refresh
 * the user asked for and is watching - a minute, on the page whose job is to show that
 * something happened.
 */
export function useInvalidateEstateOnRefreshEnd(
  status: OmTopologyRunStatus | undefined
) {
  const queryClient = useQueryClient();
  const wasActive = useRef(false);

  useEffect(() => {
    const active = isRefreshActive(status);
    if (wasActive.current && !active) {
      queryClient.invalidateQueries({ queryKey: hostsKey });
      queryClient.invalidateQueries({ queryKey: servicesKey });
    }
    wasActive.current = active;
  }, [status, queryClient]);
}

/**
 * Ask for a refresh of named hosts, or of the whole estate.
 *
 * One hook for both, because `node_ids` is a list and one host is a list of one. That
 * is what the plural on the API buys: the Hosts row action and a full sweep are the
 * same call with a different argument, so there is no second endpoint and no
 * translation step between the button and the app.
 *
 * A 409 means some other run already holds one of these hosts, which is an expected
 * answer rather than a failure - the schedule runs every ten minutes by default, so a
 * per-row refresh will meet it sooner or later. The caller reads `OmApiError.status`
 * and says so.
 */
export function useRefreshInventory() {
  const queryClient = useQueryClient();
  return useMutation<OmInventoryRunAccepted, Error, string[] | undefined>({
    mutationFn: (nodeIds) =>
      request<OmInventoryRunAccepted>('/inventory/runs:trigger', {
        method: 'POST',
        body: JSON.stringify(nodeIds?.length ? { node_ids: nodeIds } : {}),
      }),
    // onSettled rather than onSuccess: a 409 means a sweep is in flight and the estate
    // is about to change, which is exactly when a refetch is most wanted.
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: runsKey });
    },
  });
}

/**
 * Forget a host row and the services on it.
 *
 * Not suppression. An entity PMM still knows about returns on the next sweep; what
 * this clears is a row for something PMM no longer has - most often a duplicate left
 * behind when `pmm-agent setup --force` re-registered a node under a new id. Any UI
 * that offers it has to say so, or a reader will use it as a mute exactly once.
 */
export function useForgetHost() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (nodeId) =>
      request<void>(`/inventory/hosts/${encodeURIComponent(nodeId)}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: hostsKey });
      queryClient.invalidateQueries({ queryKey: servicesKey });
    },
  });
}

/** Forget one service row, on the same terms as {@link useForgetHost}. */
export function useForgetService() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (serviceId) =>
      request<void>(`/inventory/services/${encodeURIComponent(serviceId)}`, {
        method: 'DELETE',
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: hostsKey });
      queryClient.invalidateQueries({ queryKey: servicesKey });
    },
  });
}

/**
 * One refresh with the rows behind its counters.
 *
 * Fetched per run rather than read out of the history: only the detail endpoint
 * carries the entity list, because a real estate has a row per service and a
 * twenty-five-run history would carry the lot to render a panel that shows one at a
 * time. Fetched only while the panel is open, for the same reason.
 */
export function useOmInventoryRun(runId: string | undefined) {
  return useQuery<OmInventoryRunDetail>({
    queryKey: [...runsKey, 'detail', runId],
    enabled: Boolean(runId),
    queryFn: () => request<OmInventoryRunDetail>(`/inventory/runs/${runId}`),
    // A run still going gains entities as its dispatches land, so the open panel
    // follows it; a finished one never changes again.
    refetchInterval: (query) =>
      isRefreshActive(query.state.data?.run.status) ? REFRESH_POLL_MS : false,
  });
}

/**
 * The app's configuration, every field with where its value came from.
 *
 * Every field, not only the overridden ones: "why is it sweeping every ten minutes" is
 * answered by the origin rather than by the number, and a form that showed only
 * overrides could not say whether a value is the deployment's or nobody's.
 */
export function useOmInventoryConfig() {
  return useQuery<OmInventorySetting[]>({
    queryKey: configKey,
    queryFn: async () => {
      const { settings } = await request<{ settings: OmInventorySetting[] }>(
        '/inventory/config'
      );
      return settings ?? [];
    },
  });
}

/**
 * Change configuration fields.
 *
 * The response is the whole configuration as it now stands, not the keys that were
 * submitted: pmm-managed reads it back after the write, so a form renders what actually
 * took effect rather than what it asked for. That matters because beat runs as a forked
 * side-car process, and because overriding a nested parent moves what its children
 * resolve to.
 *
 * The invalidation below is therefore belt-and-braces rather than the mechanism it used
 * to be. It stays because the schedule and the run history move together, and because a
 * cache keyed on this query should not depend on a caller remembering to read the
 * mutation's result.
 *
 * The batch is atomic on the app's side: one bad key rejects all of it with a per-key
 * 422 and writes nothing, so a caller never has to work out how far a partial apply
 * got.
 */
export function useUpdateOmInventoryConfig() {
  const queryClient = useQueryClient();
  return useMutation<
    { settings: OmInventorySetting[] },
    Error,
    Record<string, unknown>
  >({
    mutationFn: (values) =>
      request<{ settings: OmInventorySetting[] }>('/inventory/config', {
        method: 'PUT',
        body: JSON.stringify(values),
      }),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: configKey });
      // The schedule drives when the next refresh runs, so the history's idea of
      // "due" changes with it.
      queryClient.invalidateQueries({ queryKey: runsKey });
    },
  });
}

/** Put one field back to whatever the deployment configured. */
export function useResetOmInventoryConfig() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: (key) =>
      request<void>(`/inventory/config/overrides/${encodeURIComponent(key)}`, {
        method: 'DELETE',
      }),
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: configKey });
    },
  });
}
