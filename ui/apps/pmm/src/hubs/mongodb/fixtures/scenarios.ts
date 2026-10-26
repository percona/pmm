import type { MongoAlertInput, MongoSnapshot } from '../mongodb.types';
import { incompleteAttention, unhealthyAttention } from './attention';
import { ago, HOUR_MS, MINUTE_MS, serviceId } from './estate';
import {
  failBackup,
  fallBehind,
  FLEET,
  fleetMembers,
  fleetSnapshot,
  holdInState,
  leaveOnVersion,
  PITR_CLUSTERS,
  shortenOplog,
  silence,
  stopPitr,
  takeDown,
  unprotect,
} from './fleet';
import { USUAL_QUERIES } from './healthy';

// The three status banner scenarios on the 60-cluster fleet. Each sets the raw data the banner and the Clusters table derive from, and its Needs attention rows as written in attention.ts.

/** All 60 clusters healthy, no alert firing, the newest backup 25 minutes old. */
export const fleetHealthySnapshot = (now: number, ops: boolean) =>
  fleetSnapshot(now, ops);

const runningQuery = (
  member: string,
  minutes: number,
  operation: string,
  collection: string
) => ({
  serviceId: serviceId(member),
  serviceName: member,
  queryText:
    operation === 'aggregate'
      ? `db.${collection}.aggregate([{ $match: { status: "open" } }, { $group: { _id: "$customer", total: { $sum: "$amount" } } }])`
      : `db.${collection}.find({ createdAt: { $lt: ISODate("2026-01-01") } })`,
  operation,
  namespace: `shop.${collection}`,
  durationMs: minutes * MINUTE_MS,
});

/** 12 of 60 clusters degraded with no alert rule yet, and 19 clusters without backups. */
export const fleetIncompleteSnapshot = (
  now: number,
  ops: boolean
): MongoSnapshot => {
  const snapshot = fleetSnapshot(now, ops);

  [
    'orders-us-m2',
    'cart-eu-m3',
    'events-us-m2',
    'users-us-m3',
    'payments-apac-m2',
  ].forEach((member) => fallBehind(snapshot, member, 45));
  shortenOplog(snapshot, 'catalog-eu', 5);
  shortenOplog(snapshot, 'orders-us', 9);
  shortenOplog(snapshot, 'billing-us', 14);
  shortenOplog(snapshot, 'ledger-us', 20);
  ['analytics-dev', 'sandbox-dev'].forEach((cluster) =>
    fleetMembers(cluster).forEach((member) =>
      silence(snapshot, member, now, 9 * MINUTE_MS)
    )
  );
  holdInState(snapshot, 'search-us-m2', 'RECOVERING', now, 14 * MINUTE_MS);
  leaveOnVersion(snapshot, 'inventory-apac-m1');

  // Without rules nothing fires.
  snapshot.alerts = [];
  snapshot.alertRules = { templates: [], custom: 0 };

  [
    ...FLEET.filter((cluster) => cluster.environment !== 'prod').map(
      (cluster) => cluster.name
    ),
    'checkout-apac',
    'accounts-apac',
  ].forEach((cluster) => unprotect(snapshot, cluster));

  snapshot.realtime = {
    sessionServiceIds: [serviceId('orders-us-m1')],
    queries: [
      runningQuery('orders-us-m1', 22, 'aggregate', 'orders'),
      runningQuery('orders-us-m1', 3, 'find', 'orders'),
    ],
  };
  snapshot.attention = incompleteAttention(ops);

  return snapshot;
};

const alert = (
  now: number,
  templateName: string,
  severity: string,
  serviceName: string,
  minutes: number
): MongoAlertInput => {
  const cluster = serviceName.replace(/-(shard\d+-)?m\d+$/, '');

  return {
    name:
      templateName === 'pmm_mongodb_down'
        ? 'MongoDB down'
        : 'MongoDB replication lag',
    templateName,
    severity,
    serviceName,
    cluster,
    environment: 'prod',
    activeAt: ago(now, minutes * MINUTE_MS),
    summary: `${templateName === 'pmm_mongodb_down' ? 'MongoDB down' : 'MongoDB replication lag'} (${serviceName})`,
  };
};

const LAGGING = [
  'orders-us-m2',
  'cart-us-m3',
  'users-us-m2',
  'events-us-m3',
  'payments-us-m2',
  'search-us-m3',
  'catalog-apac-m2',
];

/** A regional EU incident: 23 of 60 clusters need attention, 17 alerts fire, 3 nightly backups failed. */
export const fleetUnhealthySnapshot = (
  now: number,
  ops: boolean
): MongoSnapshot => {
  const snapshot = fleetSnapshot(now, ops);
  const down = (members: string[], minutes: number) =>
    members.forEach((member) =>
      takeDown(snapshot, member, now, minutes * MINUTE_MS)
    );

  // Without a primary, a set's remaining secondaries report no lag.
  down(['billing-eu-m1', 'billing-eu-m2'], 18);
  down(['ledger-eu-m1', 'ledger-eu-m2'], 16);
  down(['orders-eu-shard2-m1', 'orders-eu-shard2-m2'], 15);
  down(['payments-eu-m1', 'payments-eu-m2', 'payments-eu-m3'], 14);
  ['billing-eu-m3', 'ledger-eu-m3', 'orders-eu-shard2-m3'].forEach((member) => {
    delete snapshot.metrics?.lag[serviceId(member)];
  });
  down(['cart-eu-m4', 'cart-eu-m5'], 18);
  down(['users-eu-m4', 'users-eu-m5'], 17);
  down(['search-eu-m4', 'search-eu-m5'], 16);
  down(['sessions-eu-m3'], 15);
  down(['profiles-eu-m3'], 12);
  down(['inventory-eu-m3'], 9);

  ['reports-eu', 'audit-eu', 'reports-us', 'audit-us'].forEach((cluster) =>
    fleetMembers(cluster).forEach((member) =>
      silence(snapshot, member, now, 9 * MINUTE_MS)
    )
  );
  LAGGING.forEach((member, index) =>
    fallBehind(snapshot, member, 35 + index * 8)
  );
  shortenOplog(snapshot, 'billing-eu', 10);
  shortenOplog(snapshot, 'ledger-eu', 12);
  shortenOplog(snapshot, 'cart-eu', 16);
  shortenOplog(snapshot, 'search-eu', 18);
  shortenOplog(snapshot, 'events-eu', 20);
  leaveOnVersion(snapshot, 'inventory-apac-m1');

  ['orders-us', 'catalog-us', 'events-apac'].forEach((cluster) =>
    failBackup(snapshot, cluster, now, 8 * HOUR_MS, 32 * HOUR_MS)
  );
  PITR_CLUSTERS.forEach((cluster) =>
    stopPitr(snapshot, cluster, now, 17 * MINUTE_MS)
  );

  snapshot.alerts = [
    alert(now, 'pmm_mongodb_down', 'critical', 'billing-eu-m1', 18),
    alert(now, 'pmm_mongodb_down', 'critical', 'ledger-eu-m1', 16),
    alert(now, 'pmm_mongodb_down', 'critical', 'orders-eu-shard2-m1', 15),
    alert(now, 'pmm_mongodb_down', 'critical', 'payments-eu-m1', 14),
    ...[
      'cart-eu-m4',
      'users-eu-m4',
      'search-eu-m4',
      'sessions-eu-m3',
      'profiles-eu-m3',
      'inventory-eu-m3',
    ].map((member, index) =>
      alert(now, 'pmm_mongodb_down', 'error', member, 18 - index)
    ),
    ...LAGGING.map((member, index) =>
      alert(now, 'pmm_mongodb_replication_lag', 'warning', member, 25 - index)
    ),
  ];
  snapshot.alertRules = {
    templates: [
      'pmm_mongodb_down',
      'pmm_mongodb_replication_lag',
      'pmm_mongodb_restarted',
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
    sessionServiceIds: ['events-us-m1', 'orders-us-m1'].map(serviceId),
    queries: [
      runningQuery('events-us-m1', 22, 'aggregate', 'events'),
      runningQuery('orders-us-m1', 14, 'aggregate', 'orders'),
      runningQuery('cart-us-m1', 6, 'find', 'carts'),
      runningQuery('users-us-m1', 3, 'find', 'users'),
    ],
  };
  snapshot.attention = unhealthyAttention(ops);

  return snapshot;
};
