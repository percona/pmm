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
  OmClusterRow,
  OmEnvironmentSection,
  OmProcessRole,
  OmServiceRow,
  OmTopologyResponse,
} from './types';

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
