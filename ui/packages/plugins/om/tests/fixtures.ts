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
