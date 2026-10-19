import type { HubTreeNode } from 'hubs/template/hub.types';
import { parseVector } from 'hubs/api/victoriaMetrics';
import { RECORDED_SERVICES, RECORDED_VECTORS } from './fixtures/recorded';
import { healthySnapshot } from './fixtures/healthy';
import { wrongSnapshot } from './fixtures/wrong';
import { MONGO_QUERY_IDS, parseMongoMetrics } from './mongodb.promql';
import { toServiceInputs } from './mongodb.sources';
import {
  buildClusterTree,
  expandedProblems,
  groupClusters,
} from './mongodb.topology';
import type { MongoSnapshot } from './mongodb.types';
import {
  emptyMetrics,
  mergeMetrics,
  NOW,
  replicaSet,
  service,
} from './mongodb.testUtils';

const tree = (snapshot: MongoSnapshot, operations = false) =>
  buildClusterTree(groupClusters(snapshot.services ?? [], snapshot.metrics), {
    capabilities: { operations },
    omClusterIds: snapshot.operations?.clusterIds,
  });

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

  it('reads the live stack’s shape: one router, one config member, two shards, a replica set and a standalone', () => {
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

    expect(names(nodes)).toEqual(['billing', 'orders', 'reports']);
    expect(nodes.every((node) => node.health === 'healthy')).toBe(true);
    expect(expandedProblems(nodes)).toEqual({});
  });

  it('keeps a sharded cluster’s shape: routers, config servers, then shards worst first', () => {
    const orders = find(tree(wrongSnapshot(NOW, false)), 'orders');

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
    const shard2 = find(tree(wrongSnapshot(NOW, false)), 'orders-shard2');

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

  it('shows a member’s health only when it is not up, with since', () => {
    const nodes = tree(wrongSnapshot(NOW, false));
    const downMember = find(nodes, 'orders-shard2-m2');

    expect(find(nodes, 'orders-shard2-m1')?.showHealth).toBe(false);
    expect(downMember).toMatchObject({
      health: 'down',
      showHealth: true,
      since: new Date(NOW - 12 * 60_000).toISOString(),
    });
  });

  it('marks billing Down with no primary, 1 of 3 members up', () => {
    const billing = find(tree(wrongSnapshot(NOW, false)), 'billing');

    expect(billing).toMatchObject({
      kind: 'cluster',
      type: 'Replica set',
      health: 'down',
      members: { up: 1, total: 3, noData: 0 },
      healthDetail: "No primary: it can't take writes",
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
    const reports = find(tree(wrongSnapshot(NOW, false)), 'reports');

    expect(reports).toMatchObject({
      type: 'Standalone',
      health: 'noData',
      members: { up: 0, total: 1, noData: 1 },
      since: new Date(NOW - 9 * 60_000).toISOString(),
    });
    expect(reports?.children).toBeUndefined();
  });

  it('carries the worst secondary lag and the tightest shard window up to the cluster', () => {
    const orders = find(tree(wrongSnapshot(NOW, false)), 'orders');

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
    const nodes = tree(wrongSnapshot(NOW, false));

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
      rule: 'Warning above 30s sustained for 2m; this member is 40s behind',
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

  it('expands only along non-healthy paths', () => {
    const expanded = expandedProblems(tree(wrongSnapshot(NOW, false)));

    expect(expanded).toEqual({
      'cluster:prod/billing': true,
      'cluster:prod/orders': true,
      'set:prod/orders/orders-shard1': true,
      'set:prod/orders/orders-shard2': true,
    });
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
    const nodes = tree(wrongSnapshot(NOW, false));
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
      preview: 'developer',
    });
    expect(
      find(on, 'orders-shard1')?.doors.menu.map((door) => door.label)
    ).not.toContain('Manage in Operations');
  });
});
