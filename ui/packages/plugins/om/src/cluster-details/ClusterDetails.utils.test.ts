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
import { toClusterRows } from '../topology';
import type { OmCluster } from '../types';
import {
  cluster,
  replicaSetCluster,
  service,
  shardedCluster,
  topology,
} from '../../tests/fixtures';
import {
  buildConnectionString,
  clusterDashboardPath,
  compareMembers,
  memberGroup,
  memberHost,
  memberLag,
  memberRole,
  shardCount,
} from './ClusterDetails.utils';

const row = (source: OmCluster, envName: string | null = 'production') =>
  toClusterRows(topology([{ env_name: envName, clusters: [source] }]))[0];

describe('buildConnectionString', () => {
  it('lists a replica set’s members and names the set', () => {
    expect(buildConnectionString(row(replicaSetCluster()))).toBe(
      'mongodb://rpt-1.eu:27017,rpt-3.eu:27017,rpt-2.eu:27017/?replicaSet=rs-reporting'
    );
  });

  it('connects to a sharded cluster through its routers only', () => {
    expect(buildConnectionString(row(shardedCluster()))).toBe(
      'mongodb://mongos-1:27017,mongos-2:27017/'
    );
  });

  it('leaves arbiters and members without an address out', () => {
    const value = buildConnectionString(
      row(
        cluster({
          services: [
            service({ endpoint: 'a:27017' }),
            service({ endpoint: null }),
            service({
              endpoint: 'arb:27017',
              process_role: 'PROCESS_ROLE_ARBITER',
            }),
          ],
        })
      )
    );
    expect(value).toBe('mongodb://a:27017/?replicaSet=rs0');
  });

  it('is absent when no member has an address', () => {
    expect(
      buildConnectionString(
        row(cluster({ services: [service({ endpoint: null })] }))
      )
    ).toBeNull();
  });

  it('names no set for a standalone', () => {
    expect(
      buildConnectionString(
        row(
          cluster({
            type: 'CLUSTER_TYPE_STANDALONE',
            services: [service({ endpoint: 'solo:27017' })],
          })
        )
      )
    ).toBe('mongodb://solo:27017/');
  });
});

describe('clusterDashboardPath', () => {
  it('opens the replica set summary on this cluster', () => {
    expect(clusterDashboardPath(row(replicaSetCluster()))).toBe(
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=production&var-cluster=reporting&var-rs_nm=rs-reporting'
    );
  });

  it('opens the sharded cluster summary for a sharded cluster', () => {
    expect(clusterDashboardPath(row(shardedCluster(), null))).toBe(
      '/graph/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary?var-cluster=orders'
    );
  });

  it('opens the instance summary for a standalone', () => {
    expect(
      clusterDashboardPath(
        row(
          cluster({
            type: 'CLUSTER_TYPE_STANDALONE',
            services: [service({ service_name: 'solo' })],
          })
        )
      )
    ).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-service_name=solo'
    );
  });
});

describe('members', () => {
  it('names each role', () => {
    expect(
      [
        service({ state: 'PRIMARY' }),
        service({ state: 'SECONDARY' }),
        service({ state: 'ARBITER' }),
        service({ state: null, process_role: 'PROCESS_ROLE_MONGOS' }),
        service({ state: 'STARTUP2' }),
      ].map(memberRole)
    ).toEqual(['primary', 'secondary', 'arbiter', 'router', null]);
  });

  it('reads the host, then the endpoint, then the service name', () => {
    expect(memberHost(service({ host: 'h1' }))).toBe('h1');
    expect(memberHost(service({ host: null, endpoint: 'h2:27017' }))).toBe(
      'h2'
    );
    expect(
      memberHost(service({ host: null, endpoint: null, service_name: 's' }))
    ).toBe('s');
  });

  it('orders the primary first, then secondaries by host', () => {
    expect(
      [...replicaSetCluster().services].sort(compareMembers).map(memberHost)
    ).toEqual(['rpt-1.eu', 'rpt-2.eu', 'rpt-3.eu']);
  });

  it('groups routers, then config servers, then each shard', () => {
    const groups = new Map(
      shardedCluster().services.map((member) => {
        const group = memberGroup(member);
        return [group.key, group.label];
      })
    );
    expect([...groups].sort()).toEqual([
      ['0', 'Routers'],
      ['1:configRS', 'Config servers'],
      ['2:rs-a', 'rs-a'],
      ['2:rs-b', 'rs-b'],
    ]);
  });

  it('gives a primary no lag of its own', () => {
    expect(
      memberLag(service({ state: 'PRIMARY', replication_lag_seconds: 3 }))
    ).toBeNull();
    expect(
      memberLag(service({ state: 'SECONDARY', replication_lag_seconds: 3 }))
    ).toBe(3);
  });

  it('counts shards without the config servers', () => {
    expect(shardCount(row(shardedCluster()))).toBe(2);
  });
});
