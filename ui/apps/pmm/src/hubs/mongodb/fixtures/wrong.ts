import type { MongoSnapshot } from '../mongodb.types';
import {
  ago,
  BILLING_PITR,
  HOUR_MS,
  LATEST,
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
    delete metrics.versions[id];
  }
  snapshot.services = (snapshot.services ?? []).map((service) =>
    service.serviceId === id ? { ...service, inventoryStatus: 'down' } : service
  );
};

/** One of every attention kind on the shared estate: design section 6.3's list, a config server member recovering past its grace period, catalog's upgrade stopped halfway, payments mid-upgrade, billing's version check gone stale, an alert about a member's host, and no rule for lag or the oplog. */
export const wrongSnapshot = (now: number, ops: boolean): MongoSnapshot => {
  const snapshot = healthySnapshot(now, ops);
  const metrics = snapshot.metrics;

  down(snapshot, 'billing-m1', now, 18 * MINUTE_MS);
  down(snapshot, 'billing-m2', now, 18 * MINUTE_MS);
  // Without a primary the remaining secondary reports no lag.
  delete metrics?.lag[serviceId('billing-m3')];
  down(snapshot, 'orders-shard2-m2', now, 12 * MINUTE_MS);

  if (metrics) {
    const recovering = serviceId('orders-cfg-m3');

    metrics.self[recovering] = {
      ...metrics.self[recovering],
      state: 'RECOVERING',
    };
    metrics.stateSince[recovering] = Math.floor((now - 14 * MINUTE_MS) / 1000);
    delete metrics.lag[recovering];

    // The member payments' upgrade restarted a minute ago, still inside its grace period.
    const restarting = serviceId('payments-m2');

    metrics.self[restarting] = {
      ...metrics.self[restarting],
      state: 'RECOVERING',
    };
    metrics.stateSince[restarting] = Math.floor((now - MINUTE_MS) / 1000);
    delete metrics.lag[restarting];
    metrics.versions[restarting] = LATEST['7.0'];
    metrics.versions[serviceId('payments-m3')] = LATEST['7.0'];

    // An upgrade of catalog's secondaries that never reached its primary.
    metrics.versions[serviceId('catalog-m2')] = LATEST['7.0'];
    metrics.versions[serviceId('catalog-m3')] = LATEST['7.0'];
    metrics.lag[serviceId('orders-shard1-m3')] = { now: 40, sustained: 38 };
    metrics.oplogWindow[serviceId('orders-shard2-m1')] = 3 * 3600;
    metrics.oplogFill[serviceId('orders-shard2-m1')] = 0.96;
    metrics.oplogWindow[serviceId('orders-shard2-m3')] = 3 * 3600;
    metrics.oplogFill[serviceId('orders-shard2-m3')] = 0.96;

    const reports = serviceId('reports');

    delete metrics.up[reports];
    delete metrics.versions[reports];
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

  if (snapshot.updates) {
    snapshot.updates['prod/billing'] = {
      checkedAt: ago(now, 3 * 24 * HOUR_MS),
      latest: LATEST['7.0'],
    };
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
    // A host alert carries only the node it is about.
    {
      name: 'Node high CPU load',
      templateName: 'pmm_node_high_cpu_load',
      severity: 'warning',
      nodeName: 'orders-shard2-m1',
      activeAt: ago(now, 7 * MINUTE_MS),
      summary: 'Node high CPU load (orders-shard2-m1)',
    },
  ];
  snapshot.alertRules = {
    templates: [
      'pmm_mongodb_down',
      'pmm_mongodb_restarted',
      'pmm_node_high_cpu_load',
    ],
    custom: 0,
  };

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
      upgrades: [
        {
          clusterKey: 'prod/payments',
          done: 1,
          total: 3,
          startedAt: ago(now, 9 * MINUTE_MS),
          to: '7.0.30-16',
        },
      ],
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
