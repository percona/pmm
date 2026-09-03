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
 * A different source from `topologyHooks.ts`, on the same client and the same origin.
 * The topology document is PMM's own derivation, rebuilt per request in about a tenth
 * of a second, and it never touches a host. The estate is what SEP's probe found by
 * running a payload on the hosts themselves.
 *
 * That difference is why the two are not merged into one hook file and, more
 * importantly, why the refresh here is nothing like the sync there. `POST
 * /topology/runs:collect` recomputes a document from data PMM already holds and answers
 * with a terminal status. `POST /inventory/runs:trigger` dispatches a Nomad job per host
 * and takes tens of seconds, so it answers `running` and has to be polled. Two buttons
 * that different must not look alike, and keeping their hooks apart is the cheapest way
 * to keep them from being written alike.
 *
 * The transport both share is in `api.ts`; the pure helpers this estate feeds are in
 * `inventory.ts`.
 */

import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { useEffect, useRef } from 'react';
import { isRunActive, request } from './api';
import { periodSince, type OmRunPeriod } from './inventory';
import type {
  OmHostBootstrapAccepted,
  OmInventoryHost,
  OmInventoryRun,
  OmInventoryRunAccepted,
  OmInventoryRunDetail,
  OmInventoryService,
  OmInventorySetting,
} from './types';

const HOSTS_KEY = ['om', 'inventory', 'hosts'] as const;
const SERVICES_KEY = ['om', 'inventory', 'services'] as const;
const RUNS_KEY = ['om', 'inventory', 'runs'] as const;
const CONFIG_KEY = ['om', 'inventory', 'config'] as const;

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
  const refreshing = useEstateRefreshWatch();
  return useQuery<OmInventoryHost[]>({
    queryKey: [...HOSTS_KEY, query],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { hosts } = await request<{ hosts: OmInventoryHost[] }>(
        `/inventory/hosts${query}`
      );
      // The gateway always answers with an object, so the list arrives wrapped, and an
      // empty estate arrives as an absent key rather than an empty array.
      return hosts ?? [];
    },
    refetchInterval: refreshing ? REFRESH_POLL_MS : ESTATE_POLL_MS,
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
  const refreshing = useEstateRefreshWatch();
  return useQuery<OmInventoryService[]>({
    queryKey: SERVICES_KEY,
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { services } = await request<{ services: OmInventoryService[] }>(
        '/inventory/services'
      );
      return services ?? [];
    },
    refetchInterval: refreshing ? REFRESH_POLL_MS : ESTATE_POLL_MS,
  });
}

/**
 * Filters `GET /runs` accepts. Unset means unfiltered: newest first, default page.
 *
 * `period` names a quick-filter window rather than carrying a computed instant: the
 * query key is the period, and `since` is derived from it fresh on every fetch inside
 * `queryFn`, not once up front. A caller that memoized `periodSince(period)` itself and
 * passed the resulting string here would reproduce exactly the bug this shape exists to
 * avoid -- `since` frozen at click time while the label keeps polling, so "Last 15
 * minutes" quietly becomes "the last 35 minutes" the longer the tab stays open. `since`
 * is a lower bound, so a frozen one does not go stale by excluding new rows; it goes
 * stale by including too many.
 */
export interface OmRunFilters {
  limit?: number;
  period?: OmRunPeriod;
}

function toRunsQuery(limit: number, since: string | undefined): string {
  const params = new URLSearchParams();
  params.set('limit', String(limit));
  if (since) {
    params.set('since', since);
  }
  return `?${params.toString()}`;
}

/**
 * Refresh history, newest first.
 *
 * Polls fast while any refresh is in flight and slowly otherwise, rather than stopping:
 * the schedule starts sweeps nobody here asked for, and a history that only updated on
 * reload could not show them.
 */
export function useOmInventoryRuns(filters: OmRunFilters = {}) {
  const limit = filters.limit ?? 25;
  const period = filters.period ?? 'all';
  return useQuery<OmInventoryRun[]>({
    // The period, not the computed `since` -- see OmRunFilters. A fresh `since` on
    // every poll must not create a new query key each time, or the table would sit on
    // the loading spinner exactly like it did before periodSince was frozen upstream.
    queryKey: [...RUNS_KEY, period, limit],
    queryFn: async () => {
      const query = toRunsQuery(limit, periodSince(period));
      const { runs } = await request<{ runs: OmInventoryRun[] }>(
        `/inventory/runs${query}`
      );
      return runs ?? [];
    },
    // Any active run, not just the newest. Refreshes can be host-scoped, so two can
    // overlap -- and a narrow one started later can finish first, which would leave
    // data[0] terminal while a broader run is still probing.
    refetchInterval: (query) =>
      (query.state.data ?? []).some((run) => isRunActive(run.status))
        ? REFRESH_POLL_MS
        : ESTATE_POLL_MS,
    // Switching period keeps the old page on screen instead of blanking to the
    // spinner: the filters change the query key, and with no placeholder the table
    // would flash empty on every chip click the way HostsPage's and ServicesPage's
    // queries do not.
    placeholderData: keepPreviousData,
  });
}

/**
 * Follow the refresh history from inside an estate query, and report whether one is in
 * flight.
 *
 * Two jobs, both of which the estate queries need and neither of which a page should
 * have to remember.
 *
 * **Invalidating on the edge.** `useRefreshInventory`'s own `onSettled` cannot do it: it
 * fires when the app *accepts* the refresh, tens of seconds before any host has been
 * probed, so the rows it would refetch are the ones from before. The active set going
 * from nonempty to empty is the edge where the estate actually changed.
 *
 * The whole collection, not the newest run: refreshes are host-scoped, so two can
 * overlap, and a narrow one started later can reach a terminal status while a broader
 * one is still probing. Watching only `runs[0]` would invalidate on that -- refetching
 * rows the older run has not finished writing -- and then never fire again for the run
 * that mattered.
 *
 * **Polling at the active cadence.** A refresh writes rows as its dispatches land, so
 * an estate read at the idle minute would show a half-written sweep until the edge
 * fires. Three seconds while one is in flight follows it instead.
 *
 * Called by the estate queries rather than mounted by a page, which is the correction
 * to what this used to be. As a page-level hook it lived on Hosts and Inventory only:
 * start a refresh on either, navigate to Services before it lands, and the component
 * watching the edge unmounted with the page - leaving Services on pre-refresh rows for
 * up to a minute. Inside the queries there is nothing to forget. The runs query is
 * keyed, so a page that reads the history itself shares this one rather than adding a
 * second.
 */
function useEstateRefreshWatch(): boolean {
  const queryClient = useQueryClient();
  const { data: runs } = useOmInventoryRuns();
  const wasActive = useRef(false);

  const anyActive = (runs ?? []).some((run) => isRunActive(run.status));

  useEffect(() => {
    if (wasActive.current && !anyActive) {
      queryClient.invalidateQueries({ queryKey: HOSTS_KEY });
      queryClient.invalidateQueries({ queryKey: SERVICES_KEY });
    }
    wasActive.current = anyActive;
  }, [anyActive, queryClient]);

  return anyActive;
}

/**
 * Whether any refresh is in flight, for a page that has a button to disable.
 *
 * The same question `useEstateRefreshWatch` answers internally, exposed on its own so a
 * page does not have to read the history and fold over it to ask.
 */
export function useIsEstateRefreshing(): boolean {
  const { data: runs } = useOmInventoryRuns();
  return (runs ?? []).some((run) => isRunActive(run.status));
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
 *
 * The two scopes are returned as named actions rather than as one `mutate` taking an
 * optional argument. TanStack's `mutate` is `(variables, options?)` with `variables`
 * positional, so a `string[] | undefined` variable forces every unscoped caller to
 * write the literal `mutate(undefined)` - which reads as an oversight and had to be
 * explained twice. `refreshAll()` and `refreshHosts(ids)` say which sweep is being
 * asked for; the rest of the mutation state comes back unchanged beside them.
 */
export function useRefreshInventory() {
  const queryClient = useQueryClient();
  const mutation = useMutation<OmInventoryRunAccepted, Error, string[]>({
    // A body only when there is a scope to state. `body: "*"` on the method plus
    // grpc-gateway tolerating io.EOF means an unscoped sweep needs no `{}` to say
    // "everything"; the absent body already says it.
    mutationFn: (nodeIds) =>
      request<OmInventoryRunAccepted>('/inventory/runs:trigger', {
        method: 'POST',
        body: nodeIds.length
          ? JSON.stringify({ node_ids: nodeIds })
          : undefined,
      }),
    // onSettled rather than onSuccess: a 409 means a sweep is in flight and the estate
    // is about to change, which is exactly when a refetch is most wanted.
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: RUNS_KEY });
    },
  });

  return {
    ...mutation,
    /** Probe every host in the estate. */
    refreshAll: () => mutation.mutate([]),
    /** Probe only these hosts, by PMM node id. */
    refreshHosts: (nodeIds: string[]) => mutation.mutate(nodeIds),
  };
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
      queryClient.invalidateQueries({ queryKey: HOSTS_KEY });
      queryClient.invalidateQueries({ queryKey: SERVICES_KEY });
    },
  });
}

/**
 * Bootstrap one host: install MongoDB through the Nomad client and initialize
 * it as a single-member replica set.
 *
 * PMM-15347 PoC only. Does not invalidate the hosts query on success -- unlike
 * a refresh or a forget, nothing about the estate's *current* row changes yet;
 * the new service only appears once a later probe finds it.
 */
export function useTriggerHostBootstrap() {
  return useMutation<
    OmHostBootstrapAccepted,
    Error,
    { nodeId: string; replicaSetName: string; mongodbVersion: string }
  >({
    mutationFn: ({ nodeId, replicaSetName, mongodbVersion }) =>
      request<OmHostBootstrapAccepted>(
        `/inventory/hosts/${encodeURIComponent(nodeId)}:bootstrap`,
        {
          method: 'POST',
          body: JSON.stringify({
            replica_set_name: replicaSetName,
            mongodb_version: mongodbVersion,
          }),
        }
      ),
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
      queryClient.invalidateQueries({ queryKey: HOSTS_KEY });
      queryClient.invalidateQueries({ queryKey: SERVICES_KEY });
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
    queryKey: [...RUNS_KEY, 'detail', runId],
    enabled: Boolean(runId),
    queryFn: () => request<OmInventoryRunDetail>(`/inventory/runs/${runId}`),
    // A run still going gains entities as its dispatches land, so the open panel
    // follows it; a finished one never changes again.
    refetchInterval: (query) =>
      isRunActive(query.state.data?.run.status) ? REFRESH_POLL_MS : false,
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
    queryKey: CONFIG_KEY,
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
      queryClient.invalidateQueries({ queryKey: CONFIG_KEY });
      // The schedule drives when the next refresh runs, so the history's idea of
      // "due" changes with it.
      queryClient.invalidateQueries({ queryKey: RUNS_KEY });
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
      queryClient.invalidateQueries({ queryKey: CONFIG_KEY });
    },
  });
}
