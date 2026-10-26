import type { HubTreeNode } from 'hubs/template/hub.types';
import { parseVector } from 'hubs/api/victoriaMetrics';
import { RECORDED_SERVICES, RECORDED_VECTORS } from './fixtures/recorded';
import { healthySnapshot } from './fixtures/healthy';
import { unhealthySnapshot } from './fixtures/unhealthy';
import { MONGO_QUERY_IDS, parseMongoMetrics } from './mongodb.promql';
import { toServiceInputs } from './mongodb.sources';
import {
  buildClusterTree,
  type ClusterModel,
  groupClusters,
  updateOf,
} from './mongodb.topology';
import type { MongoSnapshot } from './mongodb.types';
import {
  emptyMetrics,
  mergeMetrics,
  NOW,
  replicaSet,
  service,
} from './mongodb.testUtils';

const tree = (
  snapshot: MongoSnapshot,
  operations = false,
  withModels: (clusters: ClusterModel[]) => ClusterModel[] = (clusters) =>
    clusters
) =>
  buildClusterTree(
    withModels(groupClusters(snapshot.services ?? [], snapshot.metrics, NOW)),
    {
      capabilities: { operations },
      omClusterIds: snapshot.operations?.clusterIds,
    }
  );

const find = (nodes: HubTreeNode[], name: string): HubTreeNode | undefined => {
  for (const node of nodes) {
    if (node.name === name) {
      return node;
    }
    const child = find(node.children ?? [], name);

    if (child) {
      return child;
    }
  }

  return undefined;
};

const names = (nodes?: HubTreeNode[]) => (nodes ?? []).map((node) => node.name);

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

  it('opens an unlabelled cluster’s dashboards without a cluster variable no series has', () => {
    const { services, metrics } = replicaSet('rs0', { cluster: '' });
    const [node] = buildClusterTree(
      groupClusters(
        services.map(({ cluster: _cluster, ...rest }) => rest),
        metrics
      ),
      { capabilities: { operations: false } }
    );
    const doors = [
      node.doors.name.to,
      ...node.doors.menu.map((door) => door.to),
      ...(node.children ?? []).flatMap((child) => [
        child.doors.name.to,
        ...child.doors.menu.map((door) => door.to),
      ]),
    ];

    expect(node.doors.name.to).toContain('var-rs_nm=rs0');
    doors.forEach((to) => expect(to).not.toContain('var-cluster'));
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
    expect(
      names(find(tree({ ...snapshot, metrics }), 'orders')?.children)
    ).toEqual([
      'Routers (mongos)',
      'orders-cfg',
      'orders-shard1',
      'orders-shard2',
    ]);
    expect(find(tree({ ...snapshot, metrics }), 'orders-cfg')).toMatchObject({
      type: 'Config servers',
      health: 'down',
    });
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

describe('buildClusterTree', () => {
  it('lists clusters worst first and folds a healthy estate', () => {
    const nodes = tree(healthySnapshot(NOW, false));

    expect(names(nodes)).toEqual([
      'billing',
      'catalog',
      'orders',
      'payments',
      'reports',
    ]);
    expect(nodes.every((node) => node.health === 'healthy')).toBe(true);
  });

  it('ranks No data next to Down, then Degraded, Upgrading and Healthy', () => {
    const nodes = tree(unhealthySnapshot(NOW, true), true, (clusters) =>
      clusters.map((cluster) =>
        cluster.name === 'payments'
          ? {
              ...cluster,
              upgrade: {
                clusterKey: cluster.key,
                done: 1,
                total: 3,
                startedAt: new Date(NOW - 9 * 60_000).toISOString(),
              },
            }
          : cluster
      )
    );

    expect(nodes.map((node) => [node.name, node.health])).toEqual([
      ['billing', 'down'],
      ['reports', 'noData'],
      ['catalog', 'degraded'],
      ['orders', 'degraded'],
      ['payments', 'upgrading'],
    ]);
  });

  it('keeps a sharded cluster’s shape: routers, config servers, then shards worst first', () => {
    const orders = find(tree(unhealthySnapshot(NOW, false)), 'orders');

    expect(orders).toMatchObject({
      type: 'Sharded cluster',
      environment: 'prod',
      health: 'degraded',
      members: { up: 10, total: 11, noData: 0 },
    });
    expect(names(orders?.children)).toEqual([
      'Routers (mongos)',
      'orders-cfg',
      'orders-shard1',
      'orders-shard2',
    ]);
    expect(orders?.children?.map((node) => node.type)).toEqual([
      'Routers',
      'Config servers',
      'Shard',
      'Shard',
    ]);
  });

  it('keeps members in topology order: primary, secondaries, then unreported', () => {
    const shard2 = find(tree(unhealthySnapshot(NOW, false)), 'orders-shard2');

    expect(names(shard2?.children)).toEqual([
      'orders-shard2-m1',
      'orders-shard2-m3',
      'orders-shard2-m2',
    ]);
    expect(shard2?.children?.map((node) => node.type)).toEqual([
      'Primary',
      'Secondary',
      'Member',
    ]);
  });

  it('shows a member’s health only when it has a problem of its own, with since', () => {
    const nodes = tree(unhealthySnapshot(NOW, false));
    const downMember = find(nodes, 'orders-shard2-m2');

    expect(find(nodes, 'orders-shard1-m1')?.showHealth).toBe(false);
    expect(find(nodes, 'orders-shard2-m1')).toMatchObject({
      health: 'degraded',
      showHealth: true,
    });
    expect(downMember).toMatchObject({
      health: 'down',
      showHealth: true,
      since: new Date(NOW - 12 * 60_000).toISOString(),
    });
  });

  it('shows an up member that is not serving as Degraded with its state', () => {
    const snapshot = healthySnapshot(NOW, false);
    const id = 'svc-billing-m3';

    if (snapshot.metrics) {
      snapshot.metrics.self[id] = {
        ...snapshot.metrics.self[id],
        state: 'RECOVERING',
      };
      delete snapshot.metrics.lag[id];
    }
    const nodes = tree(snapshot);

    expect(find(nodes, 'billing-m3')).toMatchObject({
      type: 'Recovering',
      health: 'degraded',
      showHealth: true,
    });
    expect(find(nodes, 'billing')).toMatchObject({
      health: 'degraded',
      members: { up: 3, total: 3, noData: 0 },
    });
  });

  it('marks billing Down with no primary, 1 of 3 members up', () => {
    const billing = find(tree(unhealthySnapshot(NOW, false)), 'billing');

    expect(billing).toMatchObject({
      kind: 'cluster',
      type: 'Replica set',
      health: 'down',
      members: { up: 1, total: 3, noData: 0 },
      conditions: [
        {
          text: 'No primary · 2 of 3 down',
          rule: "Down when members answer but none is primary, so the set can't take writes",
        },
      ],
    });
    expect(billing?.measures.lag.kind).toBe('notApplicable');
    expect(billing?.measures.oplogWindow.kind).toBe('notApplicable');
    expect(names(billing?.children)).toEqual([
      'billing-m3',
      'billing-m1',
      'billing-m2',
    ]);
  });

  it('marks reports No data, never up, as a standalone leaf', () => {
    const reports = find(tree(unhealthySnapshot(NOW, false)), 'reports');

    expect(reports).toMatchObject({
      type: 'Standalone',
      health: 'noData',
      members: { up: 0, total: 1, noData: 1 },
      since: new Date(NOW - 9 * 60_000).toISOString(),
    });
    expect(reports?.children).toBeUndefined();
  });

  it('says why each row is not healthy, naming only what the row is not', () => {
    const nodes = tree(unhealthySnapshot(NOW, false));
    const texts = (name: string) =>
      find(nodes, name)?.conditions?.map((condition) => condition.text);

    expect(texts('orders')).toEqual([
      'orders-shard2-m2 down',
      'orders-cfg-m3 recovering for 14m',
      'orders-shard2-m1 oplog window 3h, under 24h',
      'orders-shard1-m3 40s behind',
    ]);
    expect(texts('orders-cfg')).toEqual(['orders-cfg-m3 recovering for 14m']);
    expect(texts('orders-cfg-m3')).toEqual(['Recovering for 14m']);
    expect(texts('orders-shard1-m3')).toEqual(['40s behind']);
    expect(texts('orders-shard2-m1')).toEqual(['Oplog window 3h, under 24h']);
    expect(texts('catalog')).toEqual(['Mixed versions']);
    expect(find(nodes, 'catalog')?.conditions?.[0].rule).toBe(
      'Degraded when members of one set run different versions: 7.0.24-13 on catalog-m1; 7.0.30-16 on catalog-m2, catalog-m3'
    );
    // A down member's or standalone's chip says it all.
    expect(texts('orders-shard2-m2')).toBeUndefined();
    expect(texts('reports')).toBeUndefined();
    expect(find(nodes, 'orders')?.conditions?.[3].rule).toBe(
      'Degraded when a secondary is 10s or more behind, sustained for 2m'
    );
  });

  it('names the set of a set-wide condition on the sharded cluster’s row', () => {
    const snapshot = healthySnapshot(NOW, false);
    const { metrics } = snapshot;

    if (metrics) {
      ['svc-orders-shard2-m1', 'svc-orders-shard2-m2'].forEach((id) => {
        metrics.up[id] = { value: 0, clusterRole: 'mongod' };
      });
    }
    const nodes = tree(snapshot);

    expect(find(nodes, 'orders')?.conditions?.[0].text).toBe(
      'orders-shard2 has no primary · 2 of 3 down'
    );
    expect(find(nodes, 'orders-shard2')?.conditions?.[0].text).toBe(
      'No primary · 2 of 3 down'
    );
  });

  it('carries the worst secondary lag and the tightest shard window up to the cluster', () => {
    const orders = find(tree(unhealthySnapshot(NOW, false)), 'orders');

    expect(orders?.measures.lag).toMatchObject({
      kind: 'value',
      seconds: 40,
      level: 'warning',
    });
    expect(orders?.measures.oplogWindow).toMatchObject({
      kind: 'value',
      seconds: 3 * 3600,
      level: 'warning',
    });
  });

  it('flags only the primary’s short window, and a lagging secondary', () => {
    const nodes = tree(unhealthySnapshot(NOW, false));

    expect(find(nodes, 'orders-shard2-m1')?.measures.oplogWindow).toMatchObject(
      {
        level: 'warning',
      }
    );
    expect(find(nodes, 'orders-shard2-m3')?.measures.oplogWindow).toMatchObject(
      {
        level: 'ok',
      }
    );
    expect(find(nodes, 'orders-shard1-m3')?.measures.lag).toMatchObject({
      seconds: 40,
      level: 'warning',
      rule: 'Warning at 10s sustained for 2m; this member is 40s behind',
    });
    expect(find(nodes, 'orders-shard1-m1')?.measures.lag).toEqual({
      kind: 'notApplicable',
      reason: 'A primary has no replication lag',
    });
  });

  it('gives routers and standalones no lag or oplog', () => {
    const nodes = tree(healthySnapshot(NOW, false));

    expect(find(nodes, 'orders-mongos')?.measures.lag.kind).toBe(
      'notApplicable'
    );
    expect(find(nodes, 'reports')?.measures.oplogWindow.kind).toBe(
      'notApplicable'
    );
  });

  it('marks every set of an upgrading cluster Upgrading since the upgrade began, and keeps it folded', () => {
    const startedAt = new Date(NOW - 9 * 60_000).toISOString();
    const nodes = tree(healthySnapshot(NOW, false), false, (clusters) =>
      clusters.map((cluster) =>
        cluster.name === 'orders'
          ? {
              ...cluster,
              upgrade: {
                clusterKey: cluster.key,
                done: 4,
                total: 11,
                startedAt,
              },
            }
          : cluster
      )
    );
    const orders = find(nodes, 'orders');

    expect(orders).toMatchObject({ health: 'upgrading', since: startedAt });
    expect(orders?.children?.map((node) => node.health)).toEqual([
      'upgrading',
      'upgrading',
      'upgrading',
      'upgrading',
    ]);
  });

  it('opens each level on its dashboard with its own environment and cluster', () => {
    const nodes = tree(healthySnapshot(NOW, false));

    expect(find(nodes, 'orders')?.doors.name.to).toBe(
      '/graph/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary?var-environment=prod&var-cluster=orders'
    );
    expect(find(nodes, 'Routers (mongos)')?.doors.name.to).toBe(
      '/graph/d/mongodb-router-summary/mongodb-router-summary?var-environment=prod&var-cluster=orders'
    );
    expect(find(nodes, 'orders-shard1')?.doors.name.to).toBe(
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=orders&var-rs_nm=orders-shard1'
    );
    expect(find(nodes, 'billing')?.doors.members).toEqual({
      label: '3/3',
      to: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=billing&var-rs_nm=billing',
      context: 'members of billing',
    });
    expect(find(nodes, 'orders-shard1-m2')?.doors.name.to).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-environment=prod&var-cluster=orders&var-service_name=orders-shard1-m2'
    );
    expect(find(nodes, 'reports')?.doors.name.to).toBe(
      '/graph/d/mongodb-instance-summary/mongodb-instance-summary?var-environment=staging&var-cluster=reports&var-service_name=reports'
    );
  });

  it('opens lag on its set over 3 hours and the oplog on Oplog details', () => {
    const nodes = tree(unhealthySnapshot(NOW, false));
    const lag = find(nodes, 'orders-shard1-m3')?.measures.lag;
    const oplog = find(nodes, 'orders-shard2')?.measures.oplogWindow;

    expect(lag?.kind === 'value' && lag.door?.to).toBe(
      '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary?var-environment=prod&var-cluster=orders&var-rs_nm=orders-shard1&from=now-3h&to=now'
    );
    expect(oplog?.kind === 'value' && oplog.door?.to).toBe(
      '/graph/d/mongodb-oplog-details/mongodb-oplog-details?var-environment=prod&var-cluster=orders&var-replset=orders-shard2&var-service_name=orders-shard2-m1'
    );
  });

  it('adds Manage in Operations to cluster rows only with Operations on', () => {
    const off = tree(healthySnapshot(NOW, false));
    const on = tree(healthySnapshot(NOW, true), true);
    const manage = find(on, 'orders')?.doors.menu.find(
      (door) => door.label === 'Manage in Operations'
    );

    expect(find(off, 'orders')?.doors.menu.map((door) => door.label)).toEqual([
      'Open summary',
      'Query analytics for this cluster',
      'Backup status',
    ]);
    expect(manage).toEqual({
      label: 'Manage in Operations',
      to: '/operations?tab=clusters&cluster=om-orders',
    });
    expect(
      find(on, 'orders-shard1')?.doors.menu.map((door) => door.label)
    ).not.toContain('Manage in Operations');
  });
});

describe('versions', () => {
  const withUpdates = (snapshot: MongoSnapshot) =>
    buildClusterTree(
      groupClusters(snapshot.services ?? [], snapshot.metrics, NOW),
      {
        capabilities: { operations: false },
        updates: snapshot.updates,
        now: NOW,
      }
    );

  it('shows each member its own version and each group the lowest one its reachable members run', () => {
    const nodes = withUpdates(unhealthySnapshot(NOW, false));

    expect(find(nodes, 'catalog-m1')?.version).toEqual({
      effective: '7.0.24-13',
    });
    expect(find(nodes, 'catalog')?.version).toMatchObject({
      effective: '7.0.24-13',
      drift: [
        { version: '7.0.24-13', members: ['catalog-m1'] },
        { version: '7.0.30-16', members: ['catalog-m2', 'catalog-m3'] },
      ],
    });
    // billing-m1 and billing-m2 are down, so only billing-m3 counts.
    expect(find(nodes, 'billing')?.version?.effective).toBe('7.0.30-16');
    expect(find(nodes, 'reports')?.version).toBeUndefined();
  });

  it('marks a sharded cluster with the drift of its sets, and its version as the lowest of all', () => {
    const snapshot = healthySnapshot(NOW, false);

    if (snapshot.metrics) {
      snapshot.metrics.versions['svc-orders-shard2-m3'] = '8.0.12-4';
    }
    const orders = find(withUpdates(snapshot), 'orders');

    expect(orders?.version).toMatchObject({
      effective: '8.0.12-4',
      drift: [
        { version: '8.0.12-4', members: ['orders-shard2-m3'] },
        {
          version: '8.0.32-14',
          members: ['orders-shard2-m1', 'orders-shard2-m2'],
        },
      ],
      update: { state: 'available', latest: '8.0.32-14' },
    });
    expect(orders?.health).toBe('degraded');
  });

  it('puts what the version check says on cluster rows only', () => {
    const nodes = withUpdates(healthySnapshot(NOW, false));

    expect(find(nodes, 'payments')?.version?.update).toEqual({
      state: 'available',
      latest: '7.0.30-16',
      checkedAt: new Date(NOW - 2 * 3600_000).toISOString(),
    });
    expect(find(nodes, 'orders')?.version?.update?.state).toBe('upToDate');
    expect(find(nodes, 'orders-shard1')?.version?.update).toBeUndefined();
    expect(
      tree(healthySnapshot(NOW, false)).every((node) => !node.version?.update)
    ).toBe(true);
  });
});

describe('updateOf', () => {
  const checkedAt = new Date(NOW - 3600_000).toISOString();

  it('compares the effective version with the newest patch of its series', () => {
    expect(
      updateOf('7.0.24-13', { checkedAt, latest: '7.0.30-16' }, NOW)
    ).toEqual({ state: 'available', latest: '7.0.30-16', checkedAt });
    expect(
      updateOf('7.0.30-16', { checkedAt, latest: '7.0.30-16' }, NOW)
    ).toEqual({ state: 'upToDate', checkedAt });
  });

  it('never reads a failed, old or missing check as up to date', () => {
    expect(updateOf('7.0.30-16', { checkedAt }, NOW)).toEqual({
      state: 'unknown',
      reason: 'failed',
      checkedAt,
    });

    const old = new Date(NOW - 49 * 3600_000).toISOString();

    expect(
      updateOf('7.0.30-16', { checkedAt: old, latest: '7.0.30-16' }, NOW)
    ).toEqual({ state: 'unknown', reason: 'stale', checkedAt: old });
    expect(updateOf('7.0.30-16', undefined, NOW)).toEqual({
      state: 'unknown',
      reason: 'stale',
    });
  });
});
