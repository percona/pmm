import { describe, expect, it } from 'vitest';

import { pickExecutor, toPbmClusters } from './pbmClusters';

const agent = (
  service: string,
  role: string,
  replicaSet: string,
  cluster: string,
  value = '0'
) => ({
  metric: {
    service_id: `id-${service}`,
    service_name: service,
    replica_set: replicaSet,
    cluster,
    host: `${service}:27017`,
    role,
    self: '1',
  },
  value: [1790800000, value] as [number, string],
});

const configsvr = (service: string) => ({
  metric: { service_name: service },
  value: [1790800000, '1'] as [number, string],
});

/** The sandbox's real sharded topology, as PMM reports it. */
const SHARDED = [
  agent('cfg00', 'P', 'sharded-cluster-cfg', 'sharded-cluster'),
  agent('cfg01', 'S', 'sharded-cluster-cfg', 'sharded-cluster'),
  agent('cfg02', 'S', 'sharded-cluster-cfg', 'sharded-cluster'),
  agent('shard00svr0', 'P', 'sharded-cluster-shard00', 'sharded-cluster'),
  agent('shard00svr1', 'S', 'sharded-cluster-shard00', 'sharded-cluster'),
  agent('shard01svr0', 'P', 'sharded-cluster-shard01', 'sharded-cluster'),
  agent('shard01svr1', 'S', 'sharded-cluster-shard01', 'sharded-cluster'),
];
const SHARDED_CONFIGS = [
  configsvr('cfg00'),
  configsvr('cfg01'),
  configsvr('cfg02'),
];

describe('toPbmClusters', () => {
  it('treats a sharded cluster as one deployment, not one per shard', () => {
    // The whole point of keying on `cluster` rather than `replica_set`: PBM's
    // configuration is per deployment, so a sharded cluster is one thing to
    // configure even though it spans three replica sets.
    const clusters = toPbmClusters(SHARDED, SHARDED_CONFIGS);
    expect(clusters).toHaveLength(1);
    expect(clusters[0].name).toBe('sharded-cluster');
    expect(clusters[0].sharded).toBe(true);
    expect(clusters[0].replicaSets).toEqual([
      'sharded-cluster-cfg',
      'sharded-cluster-shard00',
      'sharded-cluster-shard01',
    ]);
  });

  it('keeps separate clusters separate', () => {
    const clusters = toPbmClusters(
      [
        ...SHARDED,
        agent('node00', 'P', 'replicaset-cluster', 'replicaset-cluster'),
        agent('single00', 'P', 'replicaset-single', 'replicaset-single'),
      ],
      SHARDED_CONFIGS
    );
    expect(clusters.map((c) => c.name)).toEqual([
      'replicaset-cluster',
      'replicaset-single',
      'sharded-cluster',
    ]);
    expect(clusters[0].sharded).toBe(false);
  });

  it('marks config-server members from the metric, not from the name', () => {
    const clusters = toPbmClusters(SHARDED, SHARDED_CONFIGS);
    const cfg = clusters[0].members
      .filter((m) => m.configServer)
      .map((m) => m.serviceName);
    expect(cfg).toEqual(['cfg00', 'cfg01', 'cfg02']);
  });

  it('falls back to the replica set when no cluster label is present', () => {
    const clusters = toPbmClusters([agent('a', 'P', 'rs0', '')]);
    expect(clusters[0].name).toBe('rs0');
  });

  it('treats any non-zero agent status as unhealthy', () => {
    const clusters = toPbmClusters([
      agent('node00', 'P', 'rs0', 'c', '0'),
      agent('node01', 'S', 'rs0', 'c', '2'),
    ]);
    expect(clusters[0].allHealthy).toBe(false);
  });

  it('skips a series that cannot be placed or keyed', () => {
    expect(
      toPbmClusters([
        { metric: { role: 'S' }, value: [0, '0'] as [number, string] },
      ])
    ).toEqual([]);
  });
});

describe('pickExecutor', () => {
  it('prefers a config-server secondary on a sharded cluster', () => {
    // PBM's metadata lives on the config replica set, so that is the closest thing
    // to where the coordination happens.
    const [cluster] = toPbmClusters(SHARDED, SHARDED_CONFIGS);
    expect(pickExecutor(cluster)?.serviceName).toBe('cfg01');
  });

  it('falls back to a shard secondary when no config secondary is healthy', () => {
    const degraded = SHARDED.map((s) =>
      s.metric.service_name.startsWith('cfg') && s.metric.role === 'S'
        ? { ...s, value: [s.value[0], '2'] as [number, string] }
        : s
    );
    const [cluster] = toPbmClusters(degraded, SHARDED_CONFIGS);
    expect(pickExecutor(cluster)?.serviceName).toBe('shard00svr1');
  });

  it('never picks the primary when a secondary is available', () => {
    const [cluster] = toPbmClusters([
      agent('node00', 'P', 'rs0', 'c'),
      agent('node01', 'S', 'rs0', 'c'),
      agent('node02', 'S', 'rs0', 'c'),
    ]);
    expect(pickExecutor(cluster)?.role).toBe('S');
  });

  it('is deterministic when several secondaries tie', () => {
    const [cluster] = toPbmClusters([
      agent('node02', 'S', 'rs0', 'c'),
      agent('node01', 'S', 'rs0', 'c'),
      agent('node00', 'P', 'rs0', 'c'),
    ]);
    // Stable by name, so an election reshuffles one row rather than the choice.
    expect(pickExecutor(cluster)?.serviceName).toBe('node01');
  });

  it('accepts the primary when it is the only live member', () => {
    // A single-node set is a cluster of one; refusing it would make the common
    // development topology unusable.
    const [cluster] = toPbmClusters([
      agent('single00', 'P', 'rs0', 'replicaset-single'),
    ]);
    expect(pickExecutor(cluster)?.serviceName).toBe('single00');
  });

  it('returns nothing when no member has a live agent', () => {
    const [cluster] = toPbmClusters([
      agent('node00', 'P', 'rs0', 'c', '2'),
      agent('node01', 'S', 'rs0', 'c', '2'),
    ]);
    expect(pickExecutor(cluster)).toBeUndefined();
  });
});
