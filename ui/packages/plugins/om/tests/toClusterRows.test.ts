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

import { describe, expect, it } from 'vitest';
import { toClusterRows, toEnvironmentSections } from '../src/topology';
import type {
  OmCluster,
  OmService,
  OmServiceStatus,
  OmTopologyResponse,
} from '../src/types';

let nextClusterID = 0;

// The id is opaque and server-derived in production; these tests only need it unique
// per cluster, since nothing here asserts on its value.
const cluster = (overrides: Partial<OmCluster>): OmCluster => ({
  name: null,
  services: [],
  id: `cluster-${nextClusterID++}`,
  type: 'CLUSTER_TYPE_REPLICA_SET',
  ...overrides,
});

const service = (overrides: Partial<OmService>): OmService => ({
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

const topology = (
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

describe('toClusterRows', () => {
  it('is empty before the document arrives', () => {
    expect(toClusterRows(undefined)).toEqual([]);
  });

  it('rolls each cluster up and keeps its environment', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({
              name: 'rs0',
              services: [
                service({ service_name: 'a', state: 'PRIMARY' }),
                service({
                  service_name: 'b',
                  state: 'SECONDARY',
                  status: 'SERVICE_STATUS_DOWN',
                }),
                service({ service_name: 'c', state: 'SECONDARY' }),
              ],
            }),
          ],
        },
      ])
    );

    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      env_name: 'sandbox',
      cluster_name: 'rs0',
      total_services: 3,
      up_services: 2,
      down_services: 1,
      by_process_role: { PROCESS_ROLE_MONGOD: 3 },
      by_state: { PRIMARY: 1, SECONDARY: 2 },
      versions: ['7.0.39-21'],
    });
  });

  it('flattens every environment into one list of clusters', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({ name: 'rs0', services: [service({})] }),
            cluster({ name: 'rs1', services: [service({})] }),
          ],
        },
        { env_name: null, clusters: [cluster({ name: null, services: [] })] },
      ])
    );

    expect(rows.map((row) => [row.env_name, row.cluster_name])).toEqual([
      ['sandbox', 'rs0'],
      ['sandbox', 'rs1'],
      [null, null],
    ]);
    expect(rows[2].total_services).toBe(0);
  });

  // The two ends are deliberate: lag is a problem at its worst member, an oplog
  // window is a budget that runs out at its tightest one.
  it('takes the worst lag and the tightest oplog window', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({
              name: 'rs0',
              services: [
                service({
                  replication_lag_seconds: 2,
                  oplog_window_seconds: 7200,
                }),
                service({
                  replication_lag_seconds: 47,
                  oplog_window_seconds: 3600,
                }),
              ],
            }),
          ],
        },
      ])
    );

    expect(rows[0].max_replication_lag_seconds).toBe(47);
    expect(rows[0].min_oplog_window_seconds).toBe(3600);
  });

  // A router has no oplog, so it must not drag the cluster's window down to zero.
  it('skips services that report neither, and keeps null when nobody does', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({
              name: 'sharded',
              services: [
                service({ process_role: 'PROCESS_ROLE_MONGOS', state: null }),
                service({
                  process_role: 'PROCESS_ROLE_SHARDSVR',
                  replication_lag_seconds: 0,
                  oplog_window_seconds: 86400,
                }),
              ],
            }),
            cluster({
              name: 'routers-only',
              services: [service({ process_role: 'PROCESS_ROLE_MONGOS' })],
            }),
          ],
        },
      ])
    );

    expect(rows[0].max_replication_lag_seconds).toBe(0);
    expect(rows[0].min_oplog_window_seconds).toBe(86400);
    expect(rows[0].by_process_role).toEqual({
      PROCESS_ROLE_MONGOS: 1,
      PROCESS_ROLE_SHARDSVR: 1,
    });
    expect(rows[1].max_replication_lag_seconds).toBeNull();
    expect(rows[1].min_oplog_window_seconds).toBeNull();
  });

  it('reports every distinct running version, sorted', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({
              name: 'rs0',
              services: [
                service({ version: '8.0.4-1' }),
                service({ version: '7.0.39-21' }),
                service({ version: '7.0.39-21' }),
                service({ version: null }),
              ],
            }),
          ],
        },
      ])
    );

    expect(rows[0].versions).toEqual(['7.0.39-21', '8.0.4-1']);
  });

  // The unfolded row renders these, so the roll-up has to keep them rather than
  // count them and throw them away.
  it('keeps the cluster its services were rolled up from', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({
              name: 'rs0',
              services: [
                service({ service_name: 'a' }),
                service({ service_name: 'b' }),
              ],
            }),
          ],
        },
      ])
    );

    expect(rows[0].services.map((s) => s.service_name)).toEqual(['a', 'b']);
  });
});

describe('toEnvironmentSections', () => {
  it('is empty before the document arrives', () => {
    expect(toEnvironmentSections(undefined)).toEqual([]);
  });

  it('gives each environment its own clusters and its own counts', () => {
    const sections = toEnvironmentSections(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            cluster({
              name: 'rs0',
              services: [
                service({}),
                service({ status: 'SERVICE_STATUS_DOWN' }),
              ],
            }),
            cluster({ name: 'rs1', services: [service({})] }),
          ],
        },
        {
          env_name: 'production',
          clusters: [
            cluster({
              name: 'rs2',
              services: [service({ status: 'SERVICE_STATUS_DOWN' })],
            }),
          ],
        },
      ])
    );

    expect(sections).toHaveLength(2);
    expect(sections[0]).toMatchObject({
      env_name: 'sandbox',
      total_services: 3,
      up_services: 2,
      down_services: 1,
    });
    expect(sections[0].clusters.map((cluster) => cluster.cluster_name)).toEqual(
      ['rs0', 'rs1']
    );
    expect(sections[1]).toMatchObject({
      env_name: 'production',
      total_services: 1,
      up_services: 0,
      down_services: 1,
    });
  });

  // An environment nobody labelled is still an environment; it must get a section
  // rather than be folded into a neighbour.
  it('keeps an unnamed environment as its own section', () => {
    const sections = toEnvironmentSections(
      topology([
        { env_name: null, clusters: [cluster({ name: null, services: [] })] },
      ])
    );

    expect(sections).toHaveLength(1);
    expect(sections[0].env_name).toBeNull();
    expect(sections[0].total_services).toBe(0);
  });
});
