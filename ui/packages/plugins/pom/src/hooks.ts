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
 * POM's read API, served by **pmm-managed** at `/v1/pom`.
 *
 * Previously this called SEP's `pom_api` at `/api/apps/pom_api` through `@sep/api`,
 * which the plugin no longer depends on at all: every POM page reads PMM's own origin
 * now. Nothing below the transport changed: pmm-managed derives the same document from
 * the same two sources — PMM's inventory and VictoriaMetrics — and gRPC-Gateway is
 * configured with `UseProtoNames` + `EmitUnpopulated`, so the JSON is snake_case with
 * explicit nulls, exactly as `types.ts` already describes it.
 *
 * Two consequences of the move, both simplifications:
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
  PomCluster,
  PomClusterRow,
  PomEnvironmentSection,
  PomProcessRole,
  PomServiceRow,
  PomTopologyResponse,
  PomRun,
  PomRunAccepted,
} from './types';

const POM_BASE = '/v1/pom';

/** Rows fetched for the run history. The API caps `limit` at 100. */
export const POM_RUNS_LIMIT = 25;

/** Poll interval while a discovery run is in flight (ms). */
const RUN_POLL_MS = 3000;
// Idle cadence for the document and the run history. pmm-managed collects on its own
// schedule, so both change without anyone here asking; without a poll the page only
// ever shows what it happened to load with.
const SNAPSHOT_POLL_MS = 30000;

const topologyKey = ['pom', 'topology'] as const;
const runsKey = ['pom', 'runs'] as const;

/**
 * One request against pmm-managed.
 *
 * Deliberately `fetch` rather than an axios instance: PMM's own axios client runs
 * `axios-case-converter` and would camelCase the response out from under `types.ts`,
 * and an axios instance would camelCase it. A bare same-origin fetch is both smaller
 * and the only one that leaves the wire shape alone.
 */
export class PomApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'PomApiError';
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
  const response = await fetch(`${POM_BASE}${path}`, {
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
    throw new PomApiError(
      response.status,
      body?.message ?? `Request failed with ${response.status}`
    );
  }
  return (await response.json()) as T;
}

/** True while a run has not reached a terminal status. */
export function isRunActive(status: string | undefined): boolean {
  return status === 'running';
}

/**
 * The whole estate from the latest terminal snapshot.
 *
 * One request, one document: the API assembles the tree per run, so there is no
 * per-cluster fetch and nothing to paginate. Keeping the previous page in place stops
 * the table collapsing to a spinner while a refetch is in flight.
 */
export function usePomTopology() {
  return useQuery<PomTopologyResponse>({
    queryKey: topologyKey,
    placeholderData: keepPreviousData,
    queryFn: () => request<PomTopologyResponse>('/topology'),
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
  topology: PomTopologyResponse | undefined
): PomServiceRow[] {
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
  cluster: PomCluster,
  envName: string | null
): PomClusterRow {
  const byProcessRole: Partial<Record<PomProcessRole, number>> = {};
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
    services: cluster.services,
    services_total: cluster.services.length,
    services_up: cluster.services.filter((service) => service.status === 'UP')
      .length,
    services_down: cluster.services.filter(
      (service) => service.status === 'DOWN'
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
  topology: PomTopologyResponse | undefined
): PomClusterRow[] {
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
  topology: PomTopologyResponse | undefined
): PomEnvironmentSection[] {
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
      services_total: clusters.reduce(
        (total, cluster) => total + cluster.services_total,
        0
      ),
      services_up: clusters.reduce(
        (total, cluster) => total + cluster.services_up,
        0
      ),
      services_down: clusters.reduce(
        (total, cluster) => total + cluster.services_down,
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
export function usePomRuns(limit: number = POM_RUNS_LIMIT) {
  return useQuery<PomRun[]>({
    queryKey: [...runsKey, limit],
    queryFn: async () => {
      const { runs } = await request<{ runs: PomRun[] }>(
        `/discovery/runs?limit=${limit}`
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
 * One run, with its per-service errors.
 *
 * Read out of the history rather than fetched: a run is small, the list is already in
 * the cache, and pmm-managed keeps the history in memory anyway — a per-run endpoint
 * would be a second way to say the same thing.
 */
export function usePomRun(run_id: string | undefined) {
  const { data, ...rest } = usePomRuns();
  return {
    ...rest,
    data: run_id ? data?.find((run) => run.run_id === run_id) : undefined,
  };
}

/**
 * Rebuild the topology document now.
 *
 * pmm-managed does this in the request — inventory is a local database read and
 * VictoriaMetrics is on localhost — so the response already carries a terminal status
 * rather than a `202` to poll.
 */
export function useTriggerPomRun() {
  const queryClient = useQueryClient();
  return useMutation<PomRunAccepted, Error, void>({
    mutationFn: () =>
      request<PomRunAccepted>('/discovery/runs', {
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
export function useInvalidatePomSnapshot() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: topologyKey });
    queryClient.invalidateQueries({ queryKey: runsKey });
  };
}
