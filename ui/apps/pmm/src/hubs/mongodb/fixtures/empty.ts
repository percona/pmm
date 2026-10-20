import type { MongoSnapshot } from '../mongodb.types';
import { DASHBOARD_CATALOG } from './estate';

/** No MongoDB monitored: no services, backups, alerts, queries or runs; the dashboards still exist. */
export const emptySnapshot = (now: number, ops: boolean): MongoSnapshot => ({
  fetchedAt: now,
  services: [],
  metrics: {
    up: {},
    lastUpAt: {},
    self: {},
    roles: {},
    stateSince: {},
    lag: {},
    oplogWindow: {},
    oplogFill: {},
    shardsByCluster: {},
    pbmConfiguredByCluster: {},
  },
  alerts: [],
  backups: { enabled: true, artifacts: [], scheduled: [] },
  qan: { current: [], baseline: [] },
  realtime: { sessionServiceIds: [], queries: [] },
  operations: ops
    ? {
        clusterIds: {},
        summary: { services: 0, up: 0 },
        upgrades: [],
        failedRuns: [],
      }
    : null,
  dashboards: { catalog: DASHBOARD_CATALOG, recent: [] },
});
