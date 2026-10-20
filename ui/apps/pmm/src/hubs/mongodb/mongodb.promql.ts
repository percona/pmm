import { matcher, type PromMatcher } from 'hubs/api/promql';
import type { PromSample } from 'hubs/api/victoriaMetrics';
import { HUB_NO_ENVIRONMENT } from 'hubs/template/hub.types';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import type { MongoMetrics } from './mongodb.types';

export type MongoQueryId =
  | 'up'
  | 'upSeen'
  | 'lastUp'
  | 'self'
  | 'selfSeen'
  | 'stateSince'
  | 'roles'
  | 'lag'
  | 'lagSustained'
  | 'oplogWindow'
  | 'oplogFill'
  | 'shards'
  | 'pbmConfigured';

export const MONGO_QUERY_IDS: MongoQueryId[] = [
  'up',
  'upSeen',
  'lastUp',
  'self',
  'selfSeen',
  'stateSince',
  'roles',
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
  // A member keeps its last reading until it is stale, then reads No data.
  const stale = `${T.staleSeconds / 60}m`;

  return {
    up: `max by (service_id, cluster_role) (last_over_time(mongodb_up{${m}}[${stale}]))`,
    // A label that flips (cluster_role when mongod stops) leaves two series in the window; the one seen last is the member now.
    upSeen: `max by (service_id, cluster_role) (tlast_over_time(mongodb_up{${m}}[${stale}]))`,
    lastUp: `max by (service_id) (tmax_over_time(mongodb_up{${m}}[24h]) and max_over_time(mongodb_up{${m}}[24h]) == 1)`,
    self: `max by (service_id, rs_nm, cl_role, member_state) (last_over_time(mongodb_members_self{${m}}[${stale}]))`,
    selfSeen: `max by (service_id, rs_nm, cl_role, member_state) (tlast_over_time(mongodb_members_self{${m}}[${stale}]))`,
    // When each state was first seen in the last hour, which the grace period for transitional states runs from.
    stateSince: `max by (service_id, member_state) (tfirst_over_time(mongodb_members_self{${m}}[1h]))`,
    // Set and role a day back, so a set whose members are all down still lands where it belongs.
    roles: `max by (service_id, rs_nm, cl_role) (last_over_time(mongodb_members_self{${m}}[24h]))`,
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

const UP_LABELS = ['service_id', 'cluster_role'];
const SELF_LABELS = ['service_id', 'rs_nm', 'cl_role', 'member_state'];
const STATE_LABELS = ['service_id', 'member_state'];

const seriesKey = (metric: Record<string, string>, labels: string[]) =>
  labels.map((label) => metric[label] ?? '').join('\u0000');

/** When each series was last seen, epoch seconds, by its labels. */
const seenAt = (samples: PromSample[], labels: string[]) =>
  new Map(
    samples.map(({ metric, value }) => [seriesKey(metric, labels), value])
  );

/** True when `at` is known to be newer than `than`; false when older, undefined when either is unknown or they tie. */
const isNewer = (at?: number, than?: number) =>
  at === undefined || than === undefined || at === than ? undefined : at > than;

const roleOf = (clRole?: string) =>
  clRole === 'configsvr' || clRole === 'shardsvr' ? clRole : undefined;

/** Folds the query results into the snapshot's metrics, every map keyed by service_id (or cluster). */
export const parseMongoMetrics = (
  results: Record<MongoQueryId, PromSample[]>
): MongoMetrics => {
  const up: MongoMetrics['up'] = {};
  const upSeen = seenAt(results.upSeen, UP_LABELS);
  const upAt: Record<string, number | undefined> = {};

  for (const { metric, value } of results.up) {
    const id = metric.service_id;

    if (!id) {
      continue;
    }
    const clusterRole = metric.cluster_role === 'mongos' ? 'mongos' : 'mongod';
    const reading = value >= 1 ? 1 : 0;
    const at = upSeen.get(seriesKey(metric, UP_LABELS));
    const previous = up[id];

    if (!previous) {
      up[id] = { value: reading, clusterRole };
      upAt[id] = at;
      continue;
    }
    const newer = isNewer(at, upAt[id]);

    up[id] = {
      // Without timestamps to tell them apart, any up series counts.
      value:
        newer === undefined
          ? reading === 1 || previous.value === 1
            ? 1
            : 0
          : newer
            ? reading
            : previous.value,
      clusterRole: previous.clusterRole === 'mongos' ? 'mongos' : clusterRole,
    };
    if (newer) {
      upAt[id] = at;
    }
  }

  const self: MongoMetrics['self'] = {};
  const selfSeen = seenAt(results.selfSeen, SELF_LABELS);
  const selfAt: Record<string, number | undefined> = {};

  for (const { metric } of results.self) {
    const id = metric.service_id;

    if (!id || !metric.rs_nm) {
      continue;
    }
    const at = selfSeen.get(seriesKey(metric, SELF_LABELS));

    if (self[id] && isNewer(at, selfAt[id]) === false) {
      continue;
    }
    const clRole = roleOf(metric.cl_role);

    self[id] = {
      rsName: metric.rs_nm,
      state: metric.member_state ?? '',
      ...(clRole ? { clRole } : {}),
    };
    selfAt[id] = at;
  }

  const statesSeen = seenAt(results.stateSince, STATE_LABELS);
  const stateSince: MongoMetrics['stateSince'] = {};

  for (const [id, { state }] of Object.entries(self)) {
    const since = statesSeen.get(
      seriesKey({ service_id: id, member_state: state }, STATE_LABELS)
    );

    if (since !== undefined) {
      stateSince[id] = since;
    }
  }

  const roles: MongoMetrics['roles'] = {};

  for (const { metric } of results.roles) {
    const id = metric.service_id;

    if (!id || !metric.rs_nm) {
      continue;
    }
    const clRole = roleOf(metric.cl_role);

    roles[id] = { rsName: metric.rs_nm, ...(clRole ? { clRole } : {}) };
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
    roles,
    stateSince,
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
