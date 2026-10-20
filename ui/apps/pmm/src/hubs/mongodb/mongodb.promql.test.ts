import { parseVector } from 'hubs/api/victoriaMetrics';
import { HUB_NO_ENVIRONMENT } from 'hubs/template/hub.types';
import { RECORDED_VECTORS } from './fixtures/recorded';
import {
  environmentMatchers,
  MONGO_QUERY_IDS,
  type MongoQueryId,
  mongoQueries,
  parseMongoMetrics,
} from './mongodb.promql';

const M = 'service_type="mongodb",job=~".*_hr"';

describe('mongoQueries', () => {
  it('builds every query of the design for all environments', () => {
    expect(mongoQueries(null)).toEqual({
      up: `max by (service_id, cluster_role) (last_over_time(mongodb_up{${M}}[5m]))`,
      upSeen: `max by (service_id, cluster_role) (tlast_over_time(mongodb_up{${M}}[5m]))`,
      lastUp: `max by (service_id) (tmax_over_time(mongodb_up{${M}}[24h]) and max_over_time(mongodb_up{${M}}[24h]) == 1)`,
      self: `max by (service_id, rs_nm, cl_role, member_state) (last_over_time(mongodb_members_self{${M}}[5m]))`,
      selfSeen: `max by (service_id, rs_nm, cl_role, member_state) (tlast_over_time(mongodb_members_self{${M}}[5m]))`,
      stateSince: `max by (service_id, member_state) (tfirst_over_time(mongodb_members_self{${M}}[1h]))`,
      roles: `max by (service_id, rs_nm, cl_role) (last_over_time(mongodb_members_self{${M}}[24h]))`,
      lag: `max by (service_id) (last_over_time(mongodb_mongod_replset_member_replication_lag{${M},self="1",state="SECONDARY"}[1m]))`,
      lagSustained: `min by (service_id) (min_over_time(mongodb_mongod_replset_member_replication_lag{${M},self="1",state="SECONDARY"}[2m]))`,
      oplogWindow: `max by (service_id) (last_over_time(mongodb_mongod_replset_oplog_head_timestamp{${M}}[5m]) - last_over_time(mongodb_mongod_replset_oplog_tail_timestamp{${M}}[5m]))`,
      oplogFill: `max by (service_id) (last_over_time(mongodb_oplog_stats_storageStats_size{service_type="mongodb"}[10m]) / last_over_time(mongodb_oplog_stats_storageStats_maxSize{service_type="mongodb"}[10m]))`,
      shards: `max by (cluster) (last_over_time(mongodb_mongos_sharding_shards_total{${M}}[5m]))`,
      pbmConfigured: `max by (cluster) (last_over_time(mongodb_pbm_cluster_backup_configured{service_type="mongodb"}[10m]))`,
    });
  });

  it('adds the environment to every selector', () => {
    const queries = mongoQueries('prod');

    MONGO_QUERY_IDS.forEach((id) => {
      const selectors = queries[id].match(/\{[^}]*\}/g) ?? [];

      expect(selectors.length).toBeGreaterThan(0);
      selectors.forEach((selector) =>
        expect(selector).toContain('environment="prod"')
      );
    });
  });

  it('matches services without the label for No environment', () => {
    expect(mongoQueries(HUB_NO_ENVIRONMENT).up).toContain('environment=""');
    expect(environmentMatchers(HUB_NO_ENVIRONMENT)).toEqual([
      { label: 'environment', value: '' },
    ]);
  });

  it('adds no environment matcher for all environments', () => {
    expect(environmentMatchers(null)).toEqual([]);
    MONGO_QUERY_IDS.forEach((id) =>
      expect(mongoQueries(null)[id]).not.toContain('environment=')
    );
  });

  it('escapes quotes and backslashes in the environment', () => {
    expect(mongoQueries('a"b\\c').up).toContain('environment="a\\"b\\\\c"');
  });
});

const recorded = (): Record<MongoQueryId, ReturnType<typeof parseVector>> =>
  Object.fromEntries(
    MONGO_QUERY_IDS.map((id) => [
      id,
      parseVector({
        status: 'success',
        data: { resultType: 'vector', result: RECORDED_VECTORS[id] },
      }),
    ])
  ) as Record<MongoQueryId, ReturnType<typeof parseVector>>;

describe('parseMongoMetrics', () => {
  const BILLING_M1 = 'b594af62-61db-443f-bb0b-c48c25c74467';
  const BILLING_M2 = '04404804-9152-445a-acfa-48ae86463b75';
  const MONGOS = '0886785b-e3e1-4a67-8299-591f22e28a02';
  const CFG = 'c125d425-10bb-4be3-ab03-8ee3c006c00e';
  const SHARD2_M3 = '20a3e890-ca31-4231-8445-97761381f714';

  it('keys everything by service id', () => {
    const metrics = parseMongoMetrics(recorded());

    expect(Object.keys(metrics.up)).toHaveLength(12);
    expect(metrics.up[MONGOS]).toEqual({ value: 1, clusterRole: 'mongos' });
    expect(metrics.up[BILLING_M1]).toEqual({ value: 1, clusterRole: 'mongod' });
    expect(metrics.lastUpAt[BILLING_M1]).toBeCloseTo(1792424598.686);
  });

  it('reads replica set, state and config or shard role from members_self', () => {
    const { self } = parseMongoMetrics(recorded());

    expect(self[CFG]).toEqual({
      rsName: 'orders-cfg',
      state: 'PRIMARY',
      clRole: 'configsvr',
    });
    expect(self[BILLING_M1]).toEqual({ rsName: 'billing', state: 'PRIMARY' });
    expect(self[MONGOS]).toBeUndefined();
  });

  it('pairs each secondary’s lag with its sustained lag', () => {
    const { lag } = parseMongoMetrics(recorded());

    expect(Object.keys(lag)).toHaveLength(6);
    expect(lag[SHARD2_M3]).toEqual({ now: 10, sustained: 0 });
  });

  it('reads oplog windows, fill, shards and PBM', () => {
    const metrics = parseMongoMetrics(recorded());

    expect(metrics.oplogWindow[BILLING_M1]).toBe(5721);
    expect(metrics.oplogFill[BILLING_M2]).toBeCloseTo(0.957, 3);
    expect(metrics.shardsByCluster).toEqual({ orders: 2 });
    expect(metrics.pbmConfiguredByCluster).toEqual({
      billing: false,
      orders: false,
      reports: false,
    });
  });

  it('counts a service as up when any of its series is up', () => {
    const metrics = parseMongoMetrics({
      ...recorded(),
      up: [
        {
          metric: { service_id: 'x', cluster_role: 'mongod' },
          value: 0,
          timestamp: 1,
        },
        {
          metric: { service_id: 'x', cluster_role: 'mongos' },
          value: 1,
          timestamp: 1,
        },
      ],
    });

    expect(metrics.up.x).toEqual({ value: 1, clusterRole: 'mongos' });
  });

  const sample = (metric: Record<string, string>, value: number) => ({
    metric,
    value,
    timestamp: 1000,
  });

  it('reads a member that just stopped as down, though its old series is still up', () => {
    const stopped = { service_id: 'x', cluster_role: 'mongod' };
    const now = { service_id: 'x' };
    const metrics = parseMongoMetrics({
      ...recorded(),
      up: [sample(stopped, 1), sample(now, 0)],
      upSeen: [sample(stopped, 900), sample(now, 950)],
    });

    expect(metrics.up.x).toEqual({ value: 0, clusterRole: 'mongod' });
  });

  it('reads a member that came back as up at once', () => {
    const back = { service_id: 'x', cluster_role: 'mongod' };
    const stopped = { service_id: 'x' };
    const metrics = parseMongoMetrics({
      ...recorded(),
      up: [sample(back, 1), sample(stopped, 0)],
      upSeen: [sample(back, 990), sample(stopped, 950)],
    });

    expect(metrics.up.x).toEqual({ value: 1, clusterRole: 'mongod' });
  });

  it('takes the member state seen last when an election leaves two series', () => {
    const series = (member_state: string) => ({
      service_id: 'x',
      rs_nm: 'billing',
      member_state,
    });
    const metrics = parseMongoMetrics({
      ...recorded(),
      self: [sample(series('PRIMARY'), 1), sample(series('SECONDARY'), 1)],
      selfSeen: [
        sample(series('PRIMARY'), 1208),
        sample(series('SECONDARY'), 1183),
      ],
    });

    expect(metrics.self.x).toEqual({ rsName: 'billing', state: 'PRIMARY' });
  });

  it('dates each member’s current state from when it was first seen in the hour', () => {
    const series = (member_state: string) => ({
      service_id: 'x',
      member_state,
    });
    const metrics = parseMongoMetrics({
      ...recorded(),
      self: [sample({ ...series('RECOVERING'), rs_nm: 'billing' }, 1)],
      selfSeen: [],
      stateSince: [
        sample(series('SECONDARY'), 400),
        sample(series('RECOVERING'), 700),
      ],
    });

    expect(metrics.stateSince.x).toBe(700);
  });

  it('remembers the set and role of the last day', () => {
    const { roles } = parseMongoMetrics(recorded());

    expect(roles[CFG]).toEqual({ rsName: 'orders-cfg', clRole: 'configsvr' });
    expect(roles[BILLING_M1]).toEqual({ rsName: 'billing' });
  });

  it('keeps a member that is down as 0', () => {
    const metrics = parseMongoMetrics({
      ...recorded(),
      up: [{ metric: { service_id: 'x' }, value: 0, timestamp: 1 }],
    });

    expect(metrics.up.x).toEqual({ value: 0, clusterRole: 'mongod' });
  });

  it('skips series without the key label', () => {
    const metrics = parseMongoMetrics({
      ...recorded(),
      oplogWindow: [{ metric: {}, value: 5, timestamp: 1 }],
    });

    expect(metrics.oplogWindow).toEqual({});
  });
});
