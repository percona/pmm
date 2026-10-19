import { matcher, type PromMatcher } from 'hubs/api/promql';
import type { PromSample } from 'hubs/api/victoriaMetrics';
import { HUB_NO_ENVIRONMENT } from 'hubs/template/hub.types';
import type { MongoMetrics } from './mongodb.types';

export type MongoQueryId =
  | 'up'
  | 'lastUp'
  | 'self'
  | 'lag'
  | 'lagSustained'
  | 'oplogWindow'
  | 'oplogFill'
  | 'shards'
  | 'pbmConfigured';

export const MONGO_QUERY_IDS: MongoQueryId[] = [
  'up',
  'lastUp',
  'self',
  'lag',
  'lagSustained',
  'oplogWindow',
  'oplogFill',
  'shards',
  'pbmConfigured',
];

/** `environment=""` matches series without the label, which is what "No environment" means. */
export const environmentMatchers = (
  environment: string | null
): PromMatcher[] =>
  environment === null
    ? []
    : [
        {
          label: 'environment',
          value: environment === HUB_NO_ENVIRONMENT ? '' : environment,
        },
      ];

/** Every hub query, keyed by id, for one environment filter (null for all). */
export const mongoQueries = (
  environment: string | null
): Record<MongoQueryId, string> => {
  const env = environmentMatchers(environment);
  const typeOnly = matcher([
    { label: 'service_type', value: 'mongodb' },
    ...env,
  ]);
  const m = matcher([
    { label: 'service_type', value: 'mongodb' },
    { label: 'job', op: '=~', value: '.*_hr' },
    ...env,
  ]);
  const ownLag = `mongodb_mongod_replset_member_replication_lag{${m},self="1",state="SECONDARY"}`;

  return {
    up: `max by (service_id, cluster_role) (last_over_time(mongodb_up{${m}}[1m]))`,
    lastUp: `max by (service_id) (tmax_over_time(mongodb_up{${m}}[24h]) and max_over_time(mongodb_up{${m}}[24h]) == 1)`,
    self: `max by (service_id, rs_nm, cl_role, member_state) (last_over_time(mongodb_members_self{${m}}[1m]))`,
    lag: `max by (service_id) (last_over_time(${ownLag}[1m]))`,
    lagSustained: `min by (service_id) (min_over_time(${ownLag}[2m]))`,
    oplogWindow: `max by (service_id) (last_over_time(mongodb_mongod_replset_oplog_head_timestamp{${m}}[5m]) - last_over_time(mongodb_mongod_replset_oplog_tail_timestamp{${m}}[5m]))`,
    // No job filter: the oplog collector's resolution differs by setup.
    oplogFill: `max by (service_id) (last_over_time(mongodb_oplog_stats_storageStats_size{${typeOnly}}[10m]) / last_over_time(mongodb_oplog_stats_storageStats_maxSize{${typeOnly}}[10m]))`,
    shards: `max by (cluster) (last_over_time(mongodb_mongos_sharding_shards_total{${m}}[5m]))`,
    pbmConfigured: `max by (cluster) (last_over_time(mongodb_pbm_cluster_backup_configured{${typeOnly}}[10m]))`,
  };
};

const byLabel = (
  samples: PromSample[],
  label: string
): Record<string, number> => {
  const values: Record<string, number> = {};

  for (const { metric, value } of samples) {
    const key = metric[label];

    if (key) {
      values[key] = value;
    }
  }

  return values;
};

/** Folds the query results into the snapshot's metrics, every map keyed by service_id (or cluster). */
export const parseMongoMetrics = (
  results: Record<MongoQueryId, PromSample[]>
): MongoMetrics => {
  const up: MongoMetrics['up'] = {};

  for (const { metric, value } of results.up) {
    const id = metric.service_id;

    if (!id) {
      continue;
    }
    const clusterRole = metric.cluster_role === 'mongos' ? 'mongos' : 'mongod';
    const previous = up[id];

    up[id] = {
      value: value >= 1 || previous?.value === 1 ? 1 : 0,
      clusterRole: previous?.clusterRole === 'mongos' ? 'mongos' : clusterRole,
    };
  }

  const self: MongoMetrics['self'] = {};

  for (const { metric } of results.self) {
    const id = metric.service_id;

    if (!id || !metric.rs_nm) {
      continue;
    }
    const clRole =
      metric.cl_role === 'configsvr' || metric.cl_role === 'shardsvr'
        ? metric.cl_role
        : undefined;

    self[id] = {
      rsName: metric.rs_nm,
      state: metric.member_state ?? '',
      ...(clRole ? { clRole } : {}),
    };
  }

  const lagNow = byLabel(results.lag, 'service_id');
  const lagSustained = byLabel(results.lagSustained, 'service_id');
  const lag: MongoMetrics['lag'] = {};

  for (const [id, now] of Object.entries(lagNow)) {
    lag[id] = { now, sustained: lagSustained[id] ?? now };
  }

  return {
    up,
    lastUpAt: byLabel(results.lastUp, 'service_id'),
    self,
    lag,
    oplogWindow: byLabel(results.oplogWindow, 'service_id'),
    oplogFill: byLabel(results.oplogFill, 'service_id'),
    shardsByCluster: byLabel(results.shards, 'cluster'),
    pbmConfiguredByCluster: Object.fromEntries(
      Object.entries(byLabel(results.pbmConfigured, 'cluster')).map(
        ([cluster, value]) => [cluster, value >= 1]
      )
    ),
  };
};
