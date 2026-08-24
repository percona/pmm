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
 * OM's read API, served by **pmm-managed** at `/v1/om`.
 *
 * Every OM page reads PMM's own origin, and the plugin depends on `@sep/api` nowhere:
 * pmm-managed derives the document from PMM's inventory and VictoriaMetrics, and
 * gRPC-Gateway is configured with `UseProtoNames` + `EmitUnpopulated`, so the JSON is
 * snake_case with explicit nulls, exactly as `types.ts` describes it.
 *
 * Two consequences of serving it from here, both simplifications:
 *
 * - **No bearer.** `/v1` is PMM's own origin and authorises on the Grafana session
 *   cookie, so there is no token to mint and no `SepAuthGate` to wait for. `fetch` with
 *   `credentials: 'same-origin'` is the whole auth story.
 * - **The trigger is synchronous.** pmm-managed has no remote fan-out to wait on, so it
 *   rebuilds the document in the request and answers with a terminal status rather than
 *   a `202` to poll. The polling below is kept because it costs nothing and keeps the
 *   page honest if the endpoint ever goes back to being asynchronous.
 */

import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import type {
  OmCluster,
  OmClusterRow,
  OmEnvironmentSection,
  OmProcessRole,
  OmServiceRow,
  OmTopologyResponse,
  OmTopologyRun,
  OmTopologyRunAccepted,
  OmTopologyRunStatus,
} from './types';

const OM_BASE = '/v1/om';

/** Rows fetched for the run history. The API caps `limit` at 100. */
export const OM_TOPOLOGY_RUNS_LIMIT = 25;

/** Poll interval while a collection run is in flight (ms). */
const RUN_POLL_MS = 3000;
// Idle cadence for the document and the run history. pmm-managed collects on its own
// schedule, so both change without anyone here asking; without a poll the page only
// ever shows what it happened to load with.
const SNAPSHOT_POLL_MS = 30000;

const topologyKey = ['om', 'topology'] as const;
const runsKey = ['om', 'runs'] as const;

/**
 * One request against pmm-managed.
 *
 * Deliberately `fetch` rather than an axios instance: PMM's own axios client runs
 * `axios-case-converter` and would camelCase the response out from under `types.ts`,
 * and an axios instance would camelCase it. A bare same-origin fetch is both smaller
 * and the only one that leaves the wire shape alone.
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
 * Exported so the inventory hooks share this client rather than growing their own.
 *
 * That is the whole point of the proxy: before it, the estate was read from SEP
 * directly with a bearer minted from the PMM session, which meant a second HTTP client
 * and a page that failed closed when SEP was unwell. One origin, one client, and
 * ``SEP is unreachable`` becomes an error inside a page that still renders.
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
    // The status has to survive onto the error: a 409 from the trigger is an expected
    // outcome the button renders differently, not a failure.
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
 * Typed on the union rather than on `string`, because the compiler is the only thing
 * that would have caught this comparison going stale when the wire values changed.
 */
export function isRunActive(status: OmTopologyRunStatus | undefined): boolean {
  return status === 'RUN_STATUS_RUNNING';
}

/**
 * The whole estate from the latest terminal snapshot.
 *
 * One request, one document: the API assembles the tree per run, so there is no
 * per-cluster fetch and nothing to paginate. Keeping the previous page in place stops
 * the table collapsing to a spinner while a refetch is in flight.
 */
export function useOmTopology() {
  return useQuery<OmTopologyResponse>({
    queryKey: topologyKey,
    placeholderData: keepPreviousData,
    queryFn: () => request<OmTopologyResponse>('/topology'),
    // pmm-managed rebuilds the document on its own timer, so a page left open goes
    // stale against an estate that has moved on. Nothing else refetches it: the
    // trigger's invalidation only covers a sync the reader started themselves.
    refetchInterval: SNAPSHOT_POLL_MS,
  });
}

/**
 * Flatten the tree into one row per service, carrying its grouping keys.
 *
 * The table sorts and filters across the whole estate, which a nested render cannot
 * do; the nesting survives as the environment and cluster columns.
 */
export function toServiceRows(
  topology: OmTopologyResponse | undefined
): OmServiceRow[] {
  if (!topology) {
    return [];
  }
  return topology.environments.flatMap((environment) =>
    environment.clusters.flatMap((cluster) =>
      cluster.services.map((service) => ({
        ...service,
        env_name: environment.env_name,
        cluster_name: cluster.name,
      }))
    )
  );
}

/**
 * Roll one cluster up into a row, keeping its services for the unfolded view.
 *
 * The two duration fields take opposite ends on purpose. Lag is a problem at its
 * worst member, so the cluster's number is the maximum; an oplog window is a budget
 * that runs out at its tightest member, so the cluster's number is the minimum.
 * Services that report neither -- routers, standalones, anything unobserved -- are
 * skipped rather than counted as zero, and a cluster where nobody reports keeps null.
 */
function rollUpCluster(
  cluster: OmCluster,
  envName: string | null | undefined
): OmClusterRow {
  const byProcessRole: Partial<Record<OmProcessRole, number>> = {};
  const byState: Record<string, number> = {};
  const versions = new Set<string>();
  let maxLag: number | null = null;
  let minWindow: number | null = null;

  for (const service of cluster.services) {
    byProcessRole[service.process_role] =
      (byProcessRole[service.process_role] ?? 0) + 1;
    if (service.state) {
      byState[service.state] = (byState[service.state] ?? 0) + 1;
    }
    if (service.version) {
      versions.add(service.version);
    }
    if (service.replication_lag_seconds != null) {
      maxLag =
        maxLag == null
          ? service.replication_lag_seconds
          : Math.max(maxLag, service.replication_lag_seconds);
    }
    if (service.oplog_window_seconds != null) {
      minWindow =
        minWindow == null
          ? service.oplog_window_seconds
          : Math.min(minWindow, service.oplog_window_seconds);
    }
  }

  return {
    env_name: envName,
    cluster_name: cluster.name,
    id: cluster.id,
    services: cluster.services,
    total_services: cluster.services.length,
    up_services: cluster.services.filter(
      (service) => service.status === 'SERVICE_STATUS_UP'
    ).length,
    down_services: cluster.services.filter(
      (service) => service.status === 'SERVICE_STATUS_DOWN'
    ).length,
    by_process_role: byProcessRole,
    by_state: byState,
    versions: [...versions].sort(),
    max_replication_lag_seconds: maxLag,
    min_oplog_window_seconds: minWindow,
  };
}

/**
 * Roll the tree up to one row per cluster, keeping its environment.
 *
 * Derived rather than fetched: the API serves the estate as one document, so the
 * overview and the topology table are two readings of the same snapshot. A separate
 * summary endpoint would let them drift apart between requests for no gain.
 */
export function toClusterRows(
  topology: OmTopologyResponse | undefined
): OmClusterRow[] {
  if (!topology) {
    return [];
  }
  return topology.environments.flatMap((environment) =>
    environment.clusters.map((cluster) =>
      rollUpCluster(cluster, environment.env_name)
    )
  );
}

/**
 * The same roll-up kept under its environment, one section per table.
 *
 * The per-environment counts are summed from the cluster rows rather than read off
 * `summary`, which is fleet-wide and has no per-environment breakdown to read.
 */
export function toEnvironmentSections(
  topology: OmTopologyResponse | undefined
): OmEnvironmentSection[] {
  if (!topology) {
    return [];
  }
  return topology.environments.map((environment) => {
    const clusters = environment.clusters.map((cluster) =>
      rollUpCluster(cluster, environment.env_name)
    );
    return {
      env_name: environment.env_name,
      clusters,
      total_services: clusters.reduce(
        (total, cluster) => total + cluster.total_services,
        0
      ),
      up_services: clusters.reduce(
        (total, cluster) => total + cluster.up_services,
        0
      ),
      down_services: clusters.reduce(
        (total, cluster) => total + cluster.down_services,
        0
      ),
    };
  });
}

/**
 * Run history, newest first.
 *
 * Polls while the newest run is still going, which is also what makes the
 * topology refresh after a triggered run lands.
 */
export function useOmTopologyRuns(limit: number = OM_TOPOLOGY_RUNS_LIMIT) {
  return useQuery<OmTopologyRun[]>({
    queryKey: [...runsKey, limit],
    queryFn: async () => {
      const { runs } = await request<{ runs: OmTopologyRun[] }>(
        `/topology/runs?limit=${limit}`
      );
      // gRPC-Gateway always answers with an object, so the list arrives wrapped.
      return runs ?? [];
    },
    // Fast while a run is in flight, slow otherwise rather than off: runs are also
    // started by pmm-managed's own timer and by other readers, and a history that only
    // updates on reload cannot show them.
    refetchInterval: (query) =>
      isRunActive(query.state.data?.[0]?.status)
        ? RUN_POLL_MS
        : SNAPSHOT_POLL_MS,
  });
}

/**
 * Rebuild the topology document now.
 *
 * pmm-managed does this in the request — inventory is a local database read and
 * VictoriaMetrics is on localhost — so the response already carries a terminal status
 * rather than a `202` to poll.
 */
export function useTriggerOmTopologyRun() {
  const queryClient = useQueryClient();
  return useMutation<OmTopologyRunAccepted, Error, void>({
    mutationFn: () =>
      request<OmTopologyRunAccepted>('/topology/runs:collect', {
        method: 'POST',
        body: '{}',
      }),
    // onSettled, not onSuccess. A 409 means pmm-managed's own timer holds the lock, so
    // a run is in flight and the document is about to change -- the one case where a
    // refetch is most obviously wanted, and the one the success-only path skipped. The
    // request is cheap and the answer is idempotent, so refetching after a failure that
    // is not a conflict costs nothing either.
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: runsKey });
      queryClient.invalidateQueries({ queryKey: topologyKey });
    },
  });
}

/** Refresh everything the snapshot backs, once a run reaches a terminal status. */
export function useInvalidateOmTopologySnapshot() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: topologyKey });
    queryClient.invalidateQueries({ queryKey: runsKey });
  };
}
