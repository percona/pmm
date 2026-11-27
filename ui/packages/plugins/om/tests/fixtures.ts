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

import type {
  OmCluster,
  OmService,
  OmServiceStatus,
  OmTopologyResponse,
} from '../src/types';

let nextClusterID = 0;

export const cluster = (overrides: Partial<OmCluster>): OmCluster => ({
  name: null,
  services: [],
  id: `cluster-${nextClusterID++}`,
  type: 'CLUSTER_TYPE_REPLICA_SET',
  ...overrides,
});

export const service = (overrides: Partial<OmService>): OmService => ({
  service_name: 'svc',
  host: 'node00',
  endpoint: 'node00:27017',
  service_id: '30',
  service_type: 'mongodb',
  version: '7.0.39-21',
  vendor: 'percona',
  edition: 'Community',
  replication_set: 'rs0',
  state: 'PRIMARY',
  status: 'SERVICE_STATUS_UP' as OmServiceStatus,
  cpu_usage_percent: 1,
  connections_free_percent: 99,
  process_role: 'PROCESS_ROLE_MONGOD',
  replication_lag_seconds: null,
  oplog_window_seconds: null,
  installed_version: null,
  config_path: null,
  argv: null,
  ...overrides,
});

export const topology = (
  environments: OmTopologyResponse['environments']
): OmTopologyResponse => ({
  snapshot: {
    generated_at: '2026-08-12T09:00:00Z',
    observed_at: '2026-08-12T09:00:00Z',
    stale: false,
    schema_version: 1,
    run_id: 'run',
  },
  origin_node: null,
  source_queries: [],
  summary: {
    environments: environments.length,
    clusters: 0,
    total_services: 0,
    up_services: 0,
    down_services: 0,
    process_role_counts: {},
  },
  environments,
});

/** A three-member replica set, primary first, with one secondary lagging. */
export const replicaSetCluster = (overrides: Partial<OmCluster> = {}) =>
  cluster({
    name: 'reporting',
    type: 'CLUSTER_TYPE_REPLICA_SET',
    services: [
      service({
        service_name: 'rpt-1',
        service_id: 'rpt-1-id',
        host: 'rpt-1.eu',
        endpoint: 'rpt-1.eu:27017',
        replication_set: 'rs-reporting',
        state: 'PRIMARY',
        oplog_window_seconds: 3300,
        replication_lag_seconds: 0,
      }),
      service({
        service_name: 'rpt-3',
        service_id: 'rpt-3-id',
        host: 'rpt-3.eu',
        endpoint: 'rpt-3.eu:27017',
        replication_set: 'rs-reporting',
        state: 'SECONDARY',
        oplog_window_seconds: 7200,
        replication_lag_seconds: 45,
      }),
      service({
        service_name: 'rpt-2',
        service_id: 'rpt-2-id',
        host: 'rpt-2.eu',
        endpoint: 'rpt-2.eu:27017',
        replication_set: 'rs-reporting',
        state: 'SECONDARY',
        oplog_window_seconds: 7200,
        replication_lag_seconds: 0.4,
      }),
    ],
    ...overrides,
  });

/** Two routers, a config server replica set and two shards. */
export const shardedCluster = (overrides: Partial<OmCluster> = {}) =>
  cluster({
    name: 'orders',
    type: 'CLUSTER_TYPE_SHARDED',
    services: [
      service({
        service_name: 'shard-b-1',
        host: 'host-b1',
        endpoint: 'host-b1:27018',
        replication_set: 'rs-b',
        process_role: 'PROCESS_ROLE_SHARDSVR',
      }),
      service({
        service_name: 'mongos-1',
        host: 'mongos-1',
        endpoint: 'mongos-1:27017',
        replication_set: null,
        state: null,
        process_role: 'PROCESS_ROLE_MONGOS',
      }),
      service({
        service_name: 'cfg-1',
        host: 'cfg-1',
        endpoint: 'cfg-1:27019',
        replication_set: 'configRS',
        process_role: 'PROCESS_ROLE_CONFIGSVR',
      }),
      service({
        service_name: 'shard-a-1',
        host: 'host-a1',
        endpoint: 'host-a1:27018',
        replication_set: 'rs-a',
        process_role: 'PROCESS_ROLE_SHARDSVR',
      }),
      service({
        service_name: 'mongos-2',
        host: 'mongos-2',
        endpoint: 'mongos-2:27017',
        replication_set: null,
        state: null,
        process_role: 'PROCESS_ROLE_MONGOS',
      }),
    ],
    ...overrides,
  });

/**
 * One environment holding a healthy cluster named first and a degraded one named
 * second, so name order and health order disagree.
 */
export const mixedEstate = () =>
  topology([
    {
      env_name: 'production',
      clusters: [
        cluster({
          name: 'alpha',
          services: [service({ service_name: 'alpha-1' })],
        }),
        cluster({
          name: 'orders',
          services: [
            service({ service_name: 'orders-1', state: 'PRIMARY' }),
            service({
              service_name: 'orders-2',
              state: 'SECONDARY',
              status: 'SERVICE_STATUS_DOWN',
            }),
            service({ service_name: 'orders-3', state: 'SECONDARY' }),
            service({
              service_name: 'orders-router',
              state: null,
              process_role: 'PROCESS_ROLE_MONGOS',
            }),
          ],
        }),
      ],
    },
  ]);
