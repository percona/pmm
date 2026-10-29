import type { MongoSnapshot } from '../mongodb.types';
import { serviceId } from './estate';
import { healthySnapshot } from './healthy';

const CATALOG = ['catalog-m1', 'catalog-m2', 'catalog-m3'];

/** A team partway through setting PMM up: orders and catalog degraded, no alert rule yet, and no backups for catalog. */
export const incompleteSnapshot = (
  now: number,
  ops: boolean
): MongoSnapshot => {
  const snapshot = healthySnapshot(now, ops);
  const metrics = snapshot.metrics;

  if (metrics) {
    // A shard secondary 40s behind.
    metrics.lag[serviceId('orders-shard1-m3')] = { now: 40, sustained: 38 };
    // An oplog window under a day, 95% full.
    CATALOG.forEach((name) => {
      metrics.oplogWindow[serviceId(name)] = 5 * 3600;
      metrics.oplogFill[serviceId(name)] = 0.95;
    });
    metrics.pbmConfiguredByCluster.catalog = false;
  }

  // Without rules nothing fires.
  snapshot.alerts = [];
  snapshot.alertRules = { templates: [], custom: 0 };

  if (snapshot.backups) {
    snapshot.backups.scheduled = snapshot.backups.scheduled.filter(
      (schedule) => !CATALOG.includes(schedule.serviceName)
    );
    snapshot.backups.artifacts = snapshot.backups.artifacts.filter(
      (artifact) => !CATALOG.includes(artifact.serviceName)
    );
  }

  return snapshot;
};
