import { parseVector } from 'hubs/api/victoriaMetrics';
import { RECORDED_SERVICES, RECORDED_VECTORS } from './fixtures/recorded';
import { healthySnapshot } from './fixtures/healthy';
import { MONGO_QUERY_IDS, parseMongoMetrics } from './mongodb.promql';
import { toServiceInputs } from './mongodb.sources';
import { groupClusters, setHealthOf } from './mongodb.topology';
import {
  emptyMetrics,
  mergeMetrics,
  NOW,
  replicaSet,
  service,
} from './mongodb.testUtils';

describe('groupClusters', () => {
  it('builds a sharded cluster from routers, config servers and shards', () => {
    const [orders] = groupClusters(
      healthySnapshot(NOW, false).services ?? [],
      healthySnapshot(NOW, false).metrics
    ).filter((cluster) => cluster.name === 'orders');

    expect(orders.kind).toBe('sharded');
    expect(orders.routers.map((member) => member.service.serviceName)).toEqual([
      'orders-mongos',
      'orders-mongos-2',
    ]);
    expect(orders.config?.name).toBe('orders-cfg');
    expect(orders.sets.map((set) => set.name)).toEqual([
      'orders-shard1',
      'orders-shard2',
    ]);
  });

  it('builds a replica set and a standalone', () => {
    const clusters = groupClusters(
      healthySnapshot(NOW, false).services ?? [],
      healthySnapshot(NOW, false).metrics
    );

    expect(clusters.find((cluster) => cluster.name === 'billing')?.kind).toBe(
      'replicaSet'
    );
    expect(clusters.find((cluster) => cluster.name === 'reports')?.kind).toBe(
      'standalone'
    );
  });

  it('groups by replication set when a service has no cluster label', () => {
    const { services, metrics } = replicaSet('rs0', { cluster: '' });
    const clusters = groupClusters(
      services.map(({ cluster: _cluster, ...rest }) => rest),
      metrics
    );

    expect(clusters).toHaveLength(1);
    expect(clusters[0]).toMatchObject({ name: 'rs0', kind: 'replicaSet' });
    expect(clusters[0].label).toBeUndefined();
  });

  it('makes an unlabelled service its own standalone', () => {
    const clusters = groupClusters(
      [service('lonely-1'), service('lonely-2')],
      emptyMetrics()
    );

    expect(clusters.map((cluster) => [cluster.name, cluster.kind])).toEqual([
      ['lonely-1', 'standalone'],
      ['lonely-2', 'standalone'],
    ]);
  });

  it('keeps clusters of the same name apart by environment', () => {
    const prod = replicaSet('app', { environment: 'prod' });
    const staging = replicaSet('app', { environment: 'staging' });
    const clusters = groupClusters(
      [
        ...prod.services,
        ...staging.services.map((svc) => ({
          ...svc,
          serviceId: `${svc.serviceId}-s`,
          serviceName: `${svc.serviceName}-s`,
        })),
      ],
      mergeMetrics(prod.metrics)
    );

    expect(clusters.map((cluster) => cluster.key)).toEqual([
      'prod/app',
      'staging/app',
    ]);
  });

  it('keeps a config server set whose members are all down as config servers', () => {
    const snapshot = healthySnapshot(NOW, false);
    const metrics = snapshot.metrics ?? emptyMetrics();

    ['orders-cfg-m1', 'orders-cfg-m2', 'orders-cfg-m3'].forEach((name) => {
      const id = `svc-${name}`;

      metrics.up[id] = { value: 0, clusterRole: 'mongod' };
      delete metrics.self[id];
    });
    const [orders] = groupClusters(snapshot.services ?? [], metrics).filter(
      (cluster) => cluster.name === 'orders'
    );

    expect(orders.config?.name).toBe('orders-cfg');
    expect(orders.sets.map((set) => set.name)).toEqual([
      'orders-shard1',
      'orders-shard2',
    ]);
    expect(orders.config && setHealthOf(orders.config)).toBe('down');
  });

  it('puts a silent member without a set into the routers of a sharded cluster', () => {
    const shard = replicaSet('s1', { cluster: 'c', clRole: 'shardsvr' });
    const shard2 = replicaSet('s2', { cluster: 'c', clRole: 'shardsvr' });
    const router = service('c-mongos', { cluster: 'c' });
    const [cluster] = groupClusters(
      [...shard.services, ...shard2.services, router],
      mergeMetrics(shard.metrics, shard2.metrics)
    );

    expect(cluster.kind).toBe('sharded');
    expect(cluster.routers.map((member) => member.verdict)).toEqual(['noData']);
  });

  it('reads a recorded estate: one router, one config member, two shards, a replica set and a standalone', () => {
    const metrics = parseMongoMetrics(
      Object.fromEntries(
        MONGO_QUERY_IDS.map((id) => [
          id,
          parseVector({
            status: 'success',
            data: { resultType: 'vector', result: RECORDED_VECTORS[id] },
          }),
        ])
      ) as Parameters<typeof parseMongoMetrics>[0]
    );
    const clusters = groupClusters(toServiceInputs(RECORDED_SERVICES), metrics);
    const orders = clusters.find((cluster) => cluster.name === 'orders');

    expect(clusters.map((cluster) => [cluster.name, cluster.kind])).toEqual([
      ['billing', 'replicaSet'],
      ['orders', 'sharded'],
      ['reports', 'standalone'],
    ]);
    expect(orders?.routers.map((m) => m.service.serviceName)).toEqual([
      'orders-mongos',
    ]);
    expect(orders?.config?.members.map((m) => m.service.serviceName)).toEqual([
      'orders-cfg-m1',
    ]);
    expect(orders?.sets.map((set) => set.members.length)).toEqual([3, 3]);
  });
});
