import type { MongoSnapshot } from '../mongodb.types';
import {
  ago,
  BILLING_PITR,
  HOUR_MS,
  MINUTE_MS,
  ORDERS_SCHEDULE,
  serviceId,
} from './estate';
import { healthySnapshot, USUAL_QUERIES } from './healthy';

const down = (
  snapshot: MongoSnapshot,
  name: string,
  now: number,
  sinceMs: number
) => {
  const id = serviceId(name);
  const metrics = snapshot.metrics;

  if (metrics) {
    metrics.up[id] = { ...metrics.up[id], value: 0 };
    metrics.lastUpAt[id] = Math.floor((now - sinceMs) / 1000);
    delete metrics.self[id];
    delete metrics.lag[id];
    delete metrics.oplogWindow[id];
    delete metrics.oplogFill[id];
  }
  snapshot.services = (snapshot.services ?? []).map((service) =>
    service.serviceId === id ? { ...service, inventoryStatus: 'down' } : service
  );
};

/** One of every attention kind on the shared estate, as design section 6.3 lists them. */
export const wrongSnapshot = (now: number, ops: boolean): MongoSnapshot => {
  const snapshot = healthySnapshot(now, ops);
  const metrics = snapshot.metrics;

  down(snapshot, 'billing-m1', now, 18 * MINUTE_MS);
  down(snapshot, 'billing-m2', now, 18 * MINUTE_MS);
  // Without a primary the remaining secondary reports no lag.
  delete metrics?.lag[serviceId('billing-m3')];
  down(snapshot, 'orders-shard2-m2', now, 12 * MINUTE_MS);

  if (metrics) {
    metrics.lag[serviceId('orders-shard1-m3')] = { now: 40, sustained: 38 };
    metrics.oplogWindow[serviceId('orders-shard2-m1')] = 3 * 3600;
    metrics.oplogFill[serviceId('orders-shard2-m1')] = 0.96;
    metrics.oplogWindow[serviceId('orders-shard2-m3')] = 3 * 3600;
    metrics.oplogFill[serviceId('orders-shard2-m3')] = 0.96;

    const reports = serviceId('reports');

    delete metrics.up[reports];
    metrics.lastUpAt[reports] = Math.floor((now - 9 * MINUTE_MS) / 1000);
  }
  snapshot.services = (snapshot.services ?? []).map((service) =>
    service.serviceName === 'reports'
      ? { ...service, inventoryStatus: 'unknown' }
      : service
  );

  if (snapshot.backups) {
    snapshot.backups.artifacts = [
      ...snapshot.backups.artifacts.filter(
        (artifact) =>
          artifact.artifactId !== 'art-orders-snapshot' &&
          artifact.artifactId !== 'art-billing-pitr'
      ),
      {
        artifactId: 'art-orders-old',
        name: `${ORDERS_SCHEDULE} 0`,
        serviceId: serviceId('orders-cfg-m1'),
        serviceName: 'orders-cfg-m1',
        mode: 'snapshot',
        status: 'success',
        createdAt: ago(now, 32 * HOUR_MS),
      },
      {
        artifactId: 'art-orders-failed',
        name: `${ORDERS_SCHEDULE} 1`,
        serviceId: serviceId('orders-cfg-m1'),
        serviceName: 'orders-cfg-m1',
        mode: 'snapshot',
        status: 'error',
        createdAt: ago(now, 8 * HOUR_MS),
      },
      {
        artifactId: 'art-billing-pitr-failed',
        name: BILLING_PITR,
        serviceId: serviceId('billing-m1'),
        serviceName: 'billing-m1',
        mode: 'pitr',
        status: 'error',
        createdAt: ago(now, 3 * HOUR_MS),
      },
    ];
  }

  snapshot.alerts = [
    {
      name: 'MongoDB down',
      templateName: 'pmm_mongodb_down',
      severity: 'critical',
      serviceName: 'orders-shard2-m2',
      cluster: 'orders',
      environment: 'prod',
      activeAt: ago(now, 11 * MINUTE_MS),
      summary: 'MongoDB down (orders-shard2-m2)',
    },
    {
      name: 'MongoDB restarted',
      templateName: 'pmm_mongodb_restarted',
      severity: 'warning',
      serviceName: 'orders-shard1-m1',
      cluster: 'orders',
      environment: 'prod',
      activeAt: ago(now, 30 * MINUTE_MS),
      summary: 'MongoDB restarted (orders-shard1-m1)',
    },
  ];

  snapshot.qan = {
    current: USUAL_QUERIES.map((query) =>
      query.queryId === 'q-orders-find'
        ? { ...query, avgQueryTime: 0.84 }
        : query
    ),
    baseline: USUAL_QUERIES,
  };

  snapshot.realtime = {
    sessionServiceIds: [serviceId('orders-shard1-m1')],
    queries: [
      {
        serviceId: serviceId('orders-shard1-m1'),
        serviceName: 'orders-shard1-m1',
        queryText:
          'db.orders.aggregate([{ $match: { status: "open" } }, { $group: { _id: "$customer", total: { $sum: "$amount" } } }])',
        operation: 'aggregate',
        namespace: 'shop.orders',
        durationMs: 14 * MINUTE_MS,
      },
    ],
  };

  if (snapshot.operations) {
    snapshot.operations = {
      ...snapshot.operations,
      // Operations counts the silent standalone as down.
      summary: { ...snapshot.operations.summary, up: 11 },
      failedRuns: [
        {
          id: 'run-install-pmm-client',
          kind: 'install',
          target: 'pmm-client',
          finishedAt: ago(now, 20 * MINUTE_MS),
        },
        {
          id: 'run-scan',
          kind: 'scan',
          target: '',
          finishedAt: ago(now, 25 * MINUTE_MS),
          failingNodes: 2,
        },
      ],
    };
  }

  return snapshot;
};
