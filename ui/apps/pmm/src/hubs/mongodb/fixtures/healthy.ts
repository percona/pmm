import type { MongoSnapshot, QanRowInput } from '../mongodb.types';
import {
  DASHBOARD_CATALOG,
  estateServices,
  HEALTHY_ALERT_TEMPLATES,
  healthyBackups,
  healthyMetrics,
  healthyOperations,
  healthyUpdates,
  recentDashboards,
} from './estate';

export const USUAL_QUERIES: QanRowInput[] = [
  {
    queryId: 'q-orders-find',
    fingerprint: 'db.orders.find({status: ?, createdAt: ?})',
    numQueries: 1200,
    avgQueryTime: 0.14,
  },
  {
    queryId: 'q-billing-update',
    fingerprint: 'db.accounts.updateOne({_id: ?}, {$inc: {balance: ?}})',
    numQueries: 800,
    avgQueryTime: 0.02,
  },
];

/** All 21 members up, backups fresh, nothing firing, queries as usual; payments and catalog one patch behind. */
export const healthySnapshot = (now: number, ops: boolean): MongoSnapshot => ({
  fetchedAt: now,
  services: estateServices(),
  metrics: healthyMetrics(now),
  alerts: [],
  alertRules: { templates: HEALTHY_ALERT_TEMPLATES, custom: 0 },
  backups: healthyBackups(now),
  qan: { current: USUAL_QUERIES, baseline: USUAL_QUERIES },
  realtime: { sessionServiceIds: [], queries: [] },
  operations: ops ? healthyOperations(now) : null,
  updates: healthyUpdates(now),
  dashboards: { catalog: DASHBOARD_CATALOG, recent: recentDashboards(now) },
});
