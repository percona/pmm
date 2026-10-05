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
 * Turning the topology document into rows, as plain functions.
 *
 * The counterpart to `inventory.ts`, and split out of the hooks for the same reason:
 * this is the part with real logic in it - the roll-ups, and the two duration fields
 * that take opposite ends - and it is testable without rendering anything or mocking a
 * fetch. `tests/toClusterRows.test.ts` is what that buys.
 */

import type {
  OmCluster,
  OmClusterHealth,
  OmClusterRow,
  OmEnvironmentSection,
  OmProcessRole,
  OmService,
  OmServiceRow,
  OmServiceStatus,
  OmTopologyResponse,
} from './types';

/** Worst first: the order a reader looking for trouble wants services in. */
const SERVICE_STATUS_RANK: Record<OmServiceStatus, number> = {
  SERVICE_STATUS_DOWN: 0,
  SERVICE_STATUS_UNSPECIFIED: 1,
  SERVICE_STATUS_UP: 2,
};

/** Rank of a status for sorting, worst first. An unrecognised value sorts as unknown. */
export function serviceStatusRank(status: OmServiceStatus): number {
  return (
    SERVICE_STATUS_RANK[status] ??
    SERVICE_STATUS_RANK.SERVICE_STATUS_UNSPECIFIED
  );
}

/** A cluster's members with the down ones first, otherwise in the document's order. */
export function downFirst<T extends Pick<OmService, 'status'>>(
  services: T[]
): T[] {
  // Array.prototype.sort is stable, so members of equal status keep their order.
  return [...services].sort(
    (a, b) => serviceStatusRank(a.status) - serviceStatusRank(b.status)
  );
}

/** Worst first, as {@link SERVICE_STATUS_RANK} is. */
const CLUSTER_HEALTH_RANK: Record<OmClusterHealth, number> = {
  down: 0,
  degraded: 1,
  unknown: 2,
  healthy: 3,
};

/** Rank of a cluster's health for sorting, worst first. */
export function clusterHealthRank(health: OmClusterHealth): number {
  return CLUSTER_HEALTH_RANK[health] ?? CLUSTER_HEALTH_RANK.unknown;
}

/**
 * Name a cluster's state from how many of its members are up and down.
 *
 * Down needs every member down; one up member still serves, which is what a DBA
 * means by degraded. Healthy needs every member up, so a member whose status was not
 * reported holds the cluster at unknown rather than letting it read as fine.
 */
export function clusterHealth(
  total: number,
  up: number,
  down: number
): OmClusterHealth {
  if (total === 0) {
    return 'unknown';
  }
  if (down === total) {
    return 'down';
  }
  if (down > 0) {
    return 'degraded';
  }
  return up === total ? 'healthy' : 'unknown';
}

/**
 * Flatten the tree into one row per service, carrying its grouping keys.
 *
 * The table sorts and filters across the whole fleet, which a nested render cannot do;
 * the nesting survives as the environment and cluster columns.
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
    // `!= null`, not `!== null`. These are proto3 `optional` scalars, which protojson
    // omits entirely rather than nulling - even under EmitUnpopulated, because they sit
    // in a synthetic oneof. So the absent case arrives as undefined, and a `!== null`
    // guard here would admit it into Math.max and report NaN for the whole cluster.
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

  const total = cluster.services.length;
  const up = cluster.services.filter(
    (service) => service.status === 'SERVICE_STATUS_UP'
  ).length;
  const down = cluster.services.filter(
    (service) => service.status === 'SERVICE_STATUS_DOWN'
  ).length;

  return {
    env_name: envName,
    cluster_name: cluster.name,
    id: cluster.id,
    services: cluster.services,
    total_services: total,
    up_services: up,
    down_services: down,
    health: clusterHealth(total, up, down),
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
 * Derived rather than fetched: the API serves the fleet as one document, so the
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
