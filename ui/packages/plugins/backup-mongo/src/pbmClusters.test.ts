import { describe, expect, it } from 'vitest';

import { toPbmClusters } from './pbmClusters';

/** One series as VictoriaMetrics returns it, trimmed to the labels the shaper reads. */
const series = (
  service: string,
  role: string,
  value: string,
  replicaSet = 'replicaset-cluster'
) => ({
  metric: {
    service_id: `id-${service}`,
    service_name: service,
    replica_set: replicaSet,
    host: `${service}:27017`,
    role,
    self: '1',
  },
  value: [1790800000, value] as [number, string],
});

describe('toPbmClusters', () => {
  it('groups members by replica set', () => {
    const clusters = toPbmClusters([
      series('node00', 'P', '0'),
      series('node01', 'S', '0'),
      series('single00', 'P', '0', 'replicaset-single'),
    ]);

    expect(clusters.map((cluster) => cluster.name)).toEqual([
      'replicaset-cluster',
      'replicaset-single',
    ]);
    expect(clusters[0].members).toHaveLength(2);
  });

  it('puts the primary first so an election reshuffles one row, not the list', () => {
    const clusters = toPbmClusters([
      series('node02', 'S', '0'),
      series('node00', 'P', '0'),
      series('node01', 'S', '0'),
    ]);

    expect(clusters[0].members.map((member) => member.serviceName)).toEqual([
      'node00',
      'node01',
      'node02',
    ]);
  });

  it('treats any non-zero status as unhealthy', () => {
    // 2 is a lost agent, established by stopping one and watching the metric. 1 has
    // never been observed, so the rule is "not 0" rather than an enumeration.
    const clusters = toPbmClusters([
      series('node00', 'P', '0'),
      series('node01', 'S', '2'),
    ]);

    expect(clusters[0].allHealthy).toBe(false);
    expect(
      clusters[0].members.find((m) => m.serviceName === 'node01')?.healthy
    ).toBe(false);
  });

  it('reports a fully healthy cluster as such', () => {
    const clusters = toPbmClusters([series('node00', 'P', '0')]);
    expect(clusters[0].allHealthy).toBe(true);
  });

  it('skips a series that cannot be placed or keyed', () => {
    const noSet = {
      metric: { service_id: 'x', role: 'S' },
      value: [0, '0'] as [number, string],
    };
    const noId = {
      metric: { replica_set: 'rs0', role: 'S' },
      value: [0, '0'] as [number, string],
    };
    expect(toPbmClusters([noSet, noId])).toEqual([]);
  });

  it('falls back to replication_set when replica_set is absent', () => {
    const clusters = toPbmClusters([
      {
        metric: {
          service_id: 'id-a',
          service_name: 'a',
          replication_set: 'rs-legacy',
          role: 'P',
        },
        value: [0, '0'] as [number, string],
      },
    ]);
    expect(clusters[0].name).toBe('rs-legacy');
  });

  it('returns nothing for an estate with no PBM metrics', () => {
    expect(toPbmClusters([])).toEqual([]);
  });
});
