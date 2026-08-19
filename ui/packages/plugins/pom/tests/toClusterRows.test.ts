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
import { toClusterRows, toEnvironmentSections } from '../src/hooks';
import type {
  PomService,
  PomServiceStatus,
  PomTopologyResponse,
} from '../src/types';

const service = (overrides: Partial<PomService>): PomService => ({
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
  status: 'UP' as PomServiceStatus,
  cpu_usage_percent: 1,
  connections_free_percent: 99,
  process_role: 'mongod',
  replication_lag_seconds: null,
  oplog_window_seconds: null,
  installed_version: null,
  config_path: null,
  argv: null,
  ...overrides,
});

const topology = (
  environments: PomTopologyResponse['environments']
): PomTopologyResponse => ({
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
    services_total: 0,
    services_up: 0,
    services_down: 0,
    by_process_role: {},
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
            {
              name: 'rs0',
              services: [
                service({ service_name: 'a', state: 'PRIMARY' }),
                service({
                  service_name: 'b',
                  state: 'SECONDARY',
                  status: 'DOWN',
                }),
                service({ service_name: 'c', state: 'SECONDARY' }),
              ],
            },
          ],
        },
      ])
    );

    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      env_name: 'sandbox',
      cluster_name: 'rs0',
      services_total: 3,
      services_up: 2,
      services_down: 1,
      by_process_role: { mongod: 3 },
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
            { name: 'rs0', services: [service({})] },
            { name: 'rs1', services: [service({})] },
          ],
        },
        { env_name: null, clusters: [{ name: null, services: [] }] },
      ])
    );

    expect(rows.map((row) => [row.env_name, row.cluster_name])).toEqual([
      ['sandbox', 'rs0'],
      ['sandbox', 'rs1'],
      [null, null],
    ]);
    expect(rows[2].services_total).toBe(0);
  });

  // The two ends are deliberate: lag is a problem at its worst member, an oplog
  // window is a budget that runs out at its tightest one.
  it('takes the worst lag and the tightest oplog window', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            {
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
            },
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
            {
              name: 'sharded',
              services: [
                service({ process_role: 'mongos', state: null }),
                service({
                  process_role: 'shardsvr',
                  replication_lag_seconds: 0,
                  oplog_window_seconds: 86400,
                }),
              ],
            },
            {
              name: 'routers-only',
              services: [service({ process_role: 'mongos' })],
            },
          ],
        },
      ])
    );

    expect(rows[0].max_replication_lag_seconds).toBe(0);
    expect(rows[0].min_oplog_window_seconds).toBe(86400);
    expect(rows[0].by_process_role).toEqual({ mongos: 1, shardsvr: 1 });
    expect(rows[1].max_replication_lag_seconds).toBeNull();
    expect(rows[1].min_oplog_window_seconds).toBeNull();
  });

  it('reports every distinct running version, sorted', () => {
    const rows = toClusterRows(
      topology([
        {
          env_name: 'sandbox',
          clusters: [
            {
              name: 'rs0',
              services: [
                service({ version: '8.0.4-1' }),
                service({ version: '7.0.39-21' }),
                service({ version: '7.0.39-21' }),
                service({ version: null }),
              ],
            },
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
            {
              name: 'rs0',
              services: [
                service({ service_name: 'a' }),
                service({ service_name: 'b' }),
              ],
            },
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
            {
              name: 'rs0',
              services: [service({}), service({ status: 'DOWN' })],
            },
            { name: 'rs1', services: [service({})] },
          ],
        },
        {
          env_name: 'production',
          clusters: [{ name: 'rs2', services: [service({ status: 'DOWN' })] }],
        },
      ])
    );

    expect(sections).toHaveLength(2);
    expect(sections[0]).toMatchObject({
      env_name: 'sandbox',
      services_total: 3,
      services_up: 2,
      services_down: 1,
    });
    expect(sections[0].clusters.map((cluster) => cluster.cluster_name)).toEqual(
      ['rs0', 'rs1']
    );
    expect(sections[1]).toMatchObject({
      env_name: 'production',
      services_total: 1,
      services_up: 0,
      services_down: 1,
    });
  });

  // An environment nobody labelled is still an environment; it must get a section
  // rather than be folded into a neighbour.
  it('keeps an unnamed environment as its own section', () => {
    const sections = toEnvironmentSections(
      topology([{ env_name: null, clusters: [{ name: null, services: [] }] }])
    );

    expect(sections).toHaveLength(1);
    expect(sections[0].env_name).toBeNull();
    expect(sections[0].services_total).toBe(0);
  });
});
