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
 * POM's estate, served by pmm-managed at `/v1/pom/inventory`.
 *
 * A different source from `hooks.ts`, on the same client and the same origin. The
 * topology document is PMM's own derivation, rebuilt per request in about a tenth of a
 * second, and it never touches a host. The estate is what SEP's probe found by running
 * a payload on the hosts themselves.
 *
 * That difference is why the two are not merged into one hook file and, more
 * importantly, why the refresh here is nothing like the sync there. `POST /discovery/runs`
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
import { request } from './hooks';
import type {
  PomInventoryHost,
  PomInventoryRunAccepted,
  PomInventoryService,
  PomProbeRun,
} from './types';

const hostsKey = ['pom', 'inventory', 'hosts'] as const;
const servicesKey = ['pom', 'inventory', 'services'] as const;
const runsKey = ['pom', 'inventory', 'runs'] as const;

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
export interface PomHostFilters {
  /** Whether the host has at least one registered MongoDB service. */
  has_service?: boolean;
  /** Whether it is currently failing its probe. */
  failing?: boolean;
  /** Restrict to one executor host. */
  executor?: string;
}

/**
 * Build the query string, omitting anything unset.
 *
 * An unset filter must not be sent as `false`: `has_service=false` is the question
 * "which hosts have no database", and sending it by accident would hide every host
 * that has one.
 */
function toQuery(filters: PomHostFilters): string {
  const params = new URLSearchParams();
  if (filters.has_service !== undefined) {
    params.set('has_service', String(filters.has_service));
  }
  if (filters.failing !== undefined) {
    params.set('failing', String(filters.failing));
  }
  if (filters.executor) {
    params.set('executor', filters.executor);
  }
  const query = params.toString();
  return query ? `?${query}` : '';
}

/**
 * Every host POM keeps a row for, with the services on it.
 *
 * One request: `GET /hosts` nests each host's services, so the row expansion needs no
 * second fetch and there is no `/hosts/{id}/services` collection to call.
 */
export function usePomInventoryHosts(filters: PomHostFilters = {}) {
  const query = toQuery(filters);
  return useQuery<PomInventoryHost[]>({
    queryKey: [...hostsKey, query],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { hosts } = await request<{ hosts: PomInventoryHost[] }>(
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
 * Every service POM has probed, flat.
 *
 * Fetched separately from the hosts rather than flattened out of them: the Services
 * page joins these against PMM's snapshot and never needs the host nesting, and asking
 * for the hosts to get at their services would carry every host's document along with
 * it.
 */
export function usePomInventoryServices() {
  return useQuery<PomInventoryService[]>({
    queryKey: servicesKey,
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { services } = await request<{ services: PomInventoryService[] }>(
        '/inventory/services'
      );
      return services ?? [];
    },
    refetchInterval: ESTATE_POLL_MS,
  });
}

/** True while a refresh has not reached a terminal status. */
export function isRefreshActive(status: string | undefined): boolean {
  return status === 'running';
}

/**
 * Refresh history, newest first.
 *
 * Polls fast while the newest refresh is in flight and slowly otherwise, rather than
 * stopping: the schedule starts sweeps nobody here asked for, and a history that only
 * updated on reload could not show them.
 */
export function usePomInventoryRuns(limit = 25) {
  return useQuery<PomProbeRun[]>({
    queryKey: [...runsKey, limit],
    queryFn: async () => {
      const { runs } = await request<{ runs: PomProbeRun[] }>(
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
 * Ask for a refresh of named hosts, or of the whole estate.
 *
 * One hook for both, because `node_ids` is a list and one host is a list of one. That
 * is what the plural on the API buys: the Hosts row action and a full sweep are the
 * same call with a different argument, so there is no second endpoint and no
 * translation step between the button and the app.
 *
 * A 409 means some other run already holds one of these hosts, which is an expected
 * answer rather than a failure - the schedule runs every ten minutes by default, so a
 * per-row refresh will meet it sooner or later. The caller reads `PomApiError.status`
 * and says so.
 */
export function useRefreshInventory() {
  const queryClient = useQueryClient();
  return useMutation<PomInventoryRunAccepted, Error, string[] | undefined>({
    mutationFn: (nodeIds) =>
      request<PomInventoryRunAccepted>('/inventory/runs', {
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
