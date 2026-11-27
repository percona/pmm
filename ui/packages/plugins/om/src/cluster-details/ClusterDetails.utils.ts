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

import { serviceDashboardPath } from '../components/ServiceLink';
import { isArbiter, isRouter } from '../topology';
import type { OmClusterRow, OmService } from '../types';
import { CLUSTER_DASHBOARD, ROLE_RANK } from './ClusterDetails.constants';
import { Messages } from './ClusterDetails.messages';
import type { MemberGroup, MemberRole } from './ClusterDetails.types';

/** A replica set's name; null for any other type, or when its members disagree. */
export const singleReplicaSet = (cluster: OmClusterRow): string | null => {
  if (cluster.type !== 'CLUSTER_TYPE_REPLICA_SET') {
    return null;
  }
  const names = new Set(
    cluster.services.map((service) => service.replication_set).filter(Boolean)
  );
  return names.size === 1 ? ([...names][0] ?? null) : null;
};

/** The shard a member belongs to; null for a router or a config server. */
const shardName = (service: OmService) =>
  isRouter(service) || service.process_role === 'PROCESS_ROLE_CONFIGSVR'
    ? null
    : service.replication_set || null;

export const shardCount = (cluster: OmClusterRow) =>
  new Set(cluster.services.map(shardName).filter(Boolean)).size;

/** The members' seed list; credentials and TLS options are left to the reader. */
export const buildConnectionString = (cluster: OmClusterRow): string | null => {
  const seeds =
    cluster.type === 'CLUSTER_TYPE_SHARDED'
      ? cluster.services.filter(isRouter)
      : cluster.services.filter((service) => !isArbiter(service));
  const hosts = [
    ...new Set(seeds.map((service) => service.endpoint).filter(Boolean)),
  ];
  if (!hosts.length) {
    return null;
  }
  const replicaSet = singleReplicaSet(cluster);
  const query = replicaSet
    ? `?replicaSet=${encodeURIComponent(replicaSet)}`
    : '';
  return `mongodb://${hosts.join(',')}/${query}`;
};

/** The cluster's PMM dashboard; a standalone has only its instance's. */
export const clusterDashboardPath = (cluster: OmClusterRow): string => {
  if (cluster.type === 'CLUSTER_TYPE_STANDALONE' && cluster.services[0]) {
    return serviceDashboardPath(cluster.services[0].service_name);
  }
  const dashboard =
    cluster.type === 'CLUSTER_TYPE_SHARDED'
      ? CLUSTER_DASHBOARD.sharded
      : CLUSTER_DASHBOARD.replicaSet;
  const params = new URLSearchParams();
  if (cluster.env_name) {
    params.set('var-environment', cluster.env_name);
  }
  if (cluster.cluster_name) {
    params.set('var-cluster', cluster.cluster_name);
  }
  const replicaSet = singleReplicaSet(cluster);
  if (replicaSet) {
    params.set('var-rs_nm', replicaSet);
  }
  const query = params.toString();
  return query ? `${dashboard}?${query}` : dashboard;
};

export const memberRole = (service: OmService): MemberRole | null => {
  if (isRouter(service)) {
    return 'router';
  }
  if (isArbiter(service)) {
    return 'arbiter';
  }
  switch (service.state) {
    case 'PRIMARY':
      return 'primary';
    case 'SECONDARY':
      return 'secondary';
    default:
      return null;
  }
};

/** A primary is what the others lag behind, so it has no lag of its own. */
export const memberLag = (service: OmService) =>
  memberRole(service) === 'primary'
    ? null
    : (service.replication_lag_seconds ?? null);

/** The host the member runs on, as the design names it, else where it listens. */
export const memberHost = (service: OmService) =>
  service.host ||
  service.endpoint?.replace(/:\d+$/, '') ||
  service.service_name;

/** A sharded cluster's member in its group: routers, then config servers, then each shard. */
export const memberGroup = (service: OmService): MemberGroup => {
  const { groups } = Messages.members;
  if (isRouter(service)) {
    return { key: '0', label: groups.routers };
  }
  if (service.process_role === 'PROCESS_ROLE_CONFIGSVR') {
    return {
      key: `1:${service.replication_set ?? ''}`,
      label: groups.configServers,
    };
  }
  const shard = shardName(service);
  return shard
    ? { key: `2:${shard}`, label: shard }
    : { key: '3', label: groups.ungrouped };
};

export const roleRank = (service: OmService) => {
  const role = memberRole(service);
  return role ? ROLE_RANK[role] : Object.keys(ROLE_RANK).length;
};

/** Primary first, then secondaries, arbiters, routers and the rest, each by host. */
export const compareMembers = (a: OmService, b: OmService) =>
  roleRank(a) - roleRank(b) || memberHost(a).localeCompare(memberHost(b));
