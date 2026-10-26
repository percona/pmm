import type {
  MongoBackupsInput,
  MongoMetrics,
  MongoOperationsInput,
  MongoServiceInput,
  MongoSnapshot,
  MongoUpdateCheck,
} from '../mongodb.types';
import {
  ago,
  DASHBOARD_CATALOG,
  HEALTHY_ALERT_TEMPLATES,
  HOUR_MS,
  LATEST,
  MINUTE_MS,
  nextRunAt,
  recentDashboards,
  serviceId,
} from './estate';
import { USUAL_QUERIES } from './healthy';

// The demo scenarios' fleet: 60 clusters across three regions and three environments. The rule tests keep the smaller estate in estate.ts.

type Role = 'router' | 'configsvr' | 'shardsvr' | 'replica';

interface FleetCluster {
  name: string;
  environment: 'prod' | 'staging' | 'dev';
  /** A replica set, or a sharded cluster with two shards, a config server set and two routers. */
  sharded?: boolean;
  /** Members per replica set; 3 unless given. */
  members?: 3 | 5;
  series: '7.0' | '8.0';
  /** One patch release behind the newest of its series. */
  behind?: boolean;
}

const prod = (
  names: string[],
  series: FleetCluster['series'],
  extra: Record<string, Partial<FleetCluster>> = {}
): FleetCluster[] =>
  names.map((name) => ({
    name,
    environment: 'prod',
    series,
    ...extra[name],
  }));

export const FLEET: FleetCluster[] = [
  ...prod(
    [
      'billing-eu',
      'ledger-eu',
      'payments-eu',
      'orders-eu',
      'cart-eu',
      'users-eu',
      'search-eu',
      'sessions-eu',
      'profiles-eu',
      'inventory-eu',
      'catalog-eu',
      'events-eu',
      'reports-eu',
      'audit-eu',
      'accounts-eu',
      'checkout-eu',
      'notifications-eu',
      'shipping-eu',
    ],
    '8.0',
    {
      'orders-eu': { sharded: true },
      'cart-eu': { members: 5 },
      'users-eu': { members: 5 },
      'search-eu': { members: 5 },
    }
  ),
  ...prod(
    [
      'billing-us',
      'ledger-us',
      'payments-us',
      'orders-us',
      'cart-us',
      'users-us',
      'search-us',
      'sessions-us',
      'profiles-us',
      'inventory-us',
      'catalog-us',
      'events-us',
      'accounts-us',
      'checkout-us',
      'notifications-us',
      'shipping-us',
    ],
    '7.0',
    { 'payments-us': { behind: true }, 'catalog-us': { behind: true } }
  ),
  ...prod(
    [
      'billing-apac',
      'payments-apac',
      'orders-apac',
      'cart-apac',
      'users-apac',
      'catalog-apac',
      'events-apac',
      'accounts-apac',
      'checkout-apac',
    ],
    '7.0',
    { 'orders-apac': { sharded: true }, 'catalog-apac': { behind: true } }
  ),
  ...[
    'inventory-apac',
    'reports-us',
    'audit-us',
    'reports-apac',
    'audit-apac',
    'search-apac',
    'sessions-apac',
    'profiles-apac',
    'notifications-apac',
  ].map(
    (name): FleetCluster => ({
      name,
      environment: 'staging',
      series: '7.0',
    })
  ),
  ...[
    'analytics-dev',
    'sandbox-dev',
    'orders-dev',
    'billing-dev',
    'payments-dev',
    'catalog-dev',
    'users-dev',
    'search-dev',
  ].map((name): FleetCluster => ({ name, environment: 'dev', series: '8.0' })),
];

interface FleetMember {
  name: string;
  cluster: FleetCluster;
  replicationSet?: string;
  role: Role;
  primary: boolean;
}

const setOf = (
  cluster: FleetCluster,
  name: string,
  role: Role,
  size: number
): FleetMember[] =>
  Array.from({ length: size }, (_, index) => ({
    name: `${name}-m${index + 1}`,
    cluster,
    replicationSet: name,
    role,
    primary: index === 0,
  }));

const membersOf = (cluster: FleetCluster): FleetMember[] =>
  cluster.sharded
    ? [
        ...['mongos', 'mongos-2'].map(
          (suffix): FleetMember => ({
            name: `${cluster.name}-${suffix}`,
            cluster,
            role: 'router',
            primary: false,
          })
        ),
        ...setOf(cluster, `${cluster.name}-cfg`, 'configsvr', 3),
        ...setOf(cluster, `${cluster.name}-shard1`, 'shardsvr', 3),
        ...setOf(cluster, `${cluster.name}-shard2`, 'shardsvr', 3),
      ]
    : setOf(cluster, cluster.name, 'replica', cluster.members ?? 3);

export const FLEET_MEMBERS: FleetMember[] = FLEET.flatMap(membersOf);

/** The member a cluster's backups run from: its primary, or its config server's. */
const backupMember = (cluster: FleetCluster) =>
  cluster.sharded ? `${cluster.name}-cfg-m1` : `${cluster.name}-m1`;

const PATCH_BEHIND = '7.0.24-13';

const versionOf = (cluster: FleetCluster) =>
  cluster.behind ? PATCH_BEHIND : LATEST[cluster.series];

const WINDOW_SECONDS: Record<Role, number | undefined> = {
  router: undefined,
  configsvr: 40 * 3600,
  shardsvr: 26 * 3600,
  replica: 40 * 3600,
};

const services = (): MongoServiceInput[] =>
  FLEET_MEMBERS.map((member) => ({
    serviceId: serviceId(member.name),
    serviceName: member.name,
    nodeName: member.name,
    environment: member.cluster.environment,
    cluster: member.cluster.name,
    ...(member.replicationSet ? { replicationSet: member.replicationSet } : {}),
    inventoryStatus: 'up',
  }));

/** Every member up and in its state for the last hour, lag 0 to 2 s, shard windows 26 h, the rest 40 h, oplogs full. */
const metrics = (now: number): MongoMetrics => {
  const result: MongoMetrics = {
    up: {},
    lastUpAt: {},
    self: {},
    roles: {},
    stateSince: {},
    versions: {},
    lag: {},
    oplogWindow: {},
    oplogFill: {},
    shardsByCluster: Object.fromEntries(
      FLEET.filter((cluster) => cluster.sharded).map((cluster) => [
        cluster.name,
        2,
      ])
    ),
    pbmConfiguredByCluster: Object.fromEntries(
      FLEET.map((cluster) => [cluster.name, true])
    ),
  };

  FLEET_MEMBERS.forEach((member, index) => {
    const id = serviceId(member.name);
    const window = WINDOW_SECONDS[member.role];

    result.up[id] = {
      value: 1,
      clusterRole: member.role === 'router' ? 'mongos' : 'mongod',
    };
    result.lastUpAt[id] = Math.floor(now / 1000);
    result.versions[id] = versionOf(member.cluster);
    if (member.replicationSet) {
      const clRole =
        member.role === 'configsvr' || member.role === 'shardsvr'
          ? member.role
          : undefined;

      result.self[id] = {
        rsName: member.replicationSet,
        state: member.primary ? 'PRIMARY' : 'SECONDARY',
        ...(clRole ? { clRole } : {}),
      };
      result.roles[id] = {
        rsName: member.replicationSet,
        ...(clRole ? { clRole } : {}),
      };
      result.stateSince[id] = Math.floor((now - HOUR_MS) / 1000);
      if (!member.primary) {
        const lag = index % 3;

        result.lag[id] = { now: lag, sustained: lag };
      }
    }
    if (window !== undefined) {
      result.oplogWindow[id] = window;
      result.oplogFill[id] = 1;
    }
  });

  return result;
};

const scheduleName = (cluster: FleetCluster) => `${cluster.name} nightly`;

/** A nightly snapshot for every cluster, the newest 25 minutes old and the rest a little older each, all within the day. */
const backups = (now: number): MongoBackupsInput => ({
  enabled: true,
  scheduled: FLEET.map((cluster, index) => {
    const member = backupMember(cluster);

    return {
      scheduledBackupId: `sched-${cluster.name}`,
      name: scheduleName(cluster),
      serviceId: serviceId(member),
      serviceName: member,
      mode: 'snapshot',
      enabled: true,
      cronExpression: `0 ${index % 24} * * *`,
      nextRun: nextRunAt(now, index % 24),
    };
  }),
  artifacts: FLEET.map((cluster, index) => {
    const member = backupMember(cluster);

    return {
      artifactId: `art-${cluster.name}`,
      name: `${scheduleName(cluster)} 1`,
      serviceId: serviceId(member),
      serviceName: member,
      mode: 'snapshot',
      status: 'success',
      createdAt: ago(now, 25 * MINUTE_MS + index * 17 * MINUTE_MS),
    };
  }),
});

/** Point-in-time recovery on the clusters that take payments. */
export const PITR_CLUSTERS = ['billing-eu', 'ledger-eu'];

const pitr = (now: number, backupsInput: MongoBackupsInput) => {
  PITR_CLUSTERS.forEach((name) => {
    const member = `${name}-m1`;

    backupsInput.scheduled.push({
      scheduledBackupId: `sched-${name}-pitr`,
      name: `${name} point-in-time recovery`,
      serviceId: serviceId(member),
      serviceName: member,
      mode: 'pitr',
      enabled: true,
      cronExpression: '',
    });
    backupsInput.artifacts.push({
      artifactId: `art-${name}-pitr`,
      name: `${name} point-in-time recovery`,
      serviceId: serviceId(member),
      serviceName: member,
      mode: 'pitr',
      status: 'inProgress',
      createdAt: ago(now, 6 * HOUR_MS),
    });
  });

  return backupsInput;
};

/** Each cluster checked 2 hours ago against the newest patch of its series. */
const updates = (now: number): Record<string, MongoUpdateCheck> =>
  Object.fromEntries(
    FLEET.map((cluster) => [
      `${cluster.environment}/${cluster.name}`,
      { checkedAt: ago(now, 2 * HOUR_MS), latest: LATEST[cluster.series] },
    ])
  );

const operations = (now: number): MongoOperationsInput => ({
  clusterIds: Object.fromEntries(
    FLEET.map((cluster) => [
      `${cluster.environment}/${cluster.name}`,
      `om-${cluster.name}`,
    ])
  ),
  summary: {
    services: FLEET_MEMBERS.length,
    up: FLEET_MEMBERS.length,
    lastScanAt: ago(now, 4 * MINUTE_MS),
  },
  upgrades: [],
  failedRuns: [],
});

/** The whole fleet up and healthy: backups fresh, nothing firing, queries as usual, a few clusters one patch behind. */
export const fleetSnapshot = (now: number, ops: boolean): MongoSnapshot => ({
  fetchedAt: now,
  services: services(),
  metrics: metrics(now),
  alerts: [],
  alertRules: { templates: HEALTHY_ALERT_TEMPLATES, custom: 0 },
  backups: pitr(now, backups(now)),
  qan: { current: USUAL_QUERIES, baseline: USUAL_QUERIES },
  realtime: { sessionServiceIds: [], queries: [] },
  operations: ops ? operations(now) : null,
  updates: updates(now),
  dashboards: { catalog: DASHBOARD_CATALOG, recent: recentDashboards(now) },
});

/** Members of a fleet cluster, by name. */
export const fleetMembers = (cluster: string) =>
  FLEET_MEMBERS.filter((member) => member.cluster.name === cluster).map(
    (member) => member.name
  );

/** A member stops answering and the inventory says it is down. */
export const takeDown = (
  snapshot: MongoSnapshot,
  name: string,
  now: number,
  sinceMs: number
) => {
  const id = serviceId(name);
  const { metrics: result } = snapshot;

  if (result) {
    result.up[id] = { ...result.up[id], value: 0 };
    result.lastUpAt[id] = Math.floor((now - sinceMs) / 1000);
    delete result.self[id];
    delete result.lag[id];
    delete result.oplogWindow[id];
    delete result.oplogFill[id];
    delete result.versions[id];
  }
  snapshot.services = (snapshot.services ?? []).map((service) =>
    service.serviceId === id ? { ...service, inventoryStatus: 'down' } : service
  );
};

/** PMM stops getting data from a member, and the inventory can't say whether it is up. */
export const silence = (
  snapshot: MongoSnapshot,
  name: string,
  now: number,
  sinceMs: number
) => {
  const id = serviceId(name);
  const { metrics: result } = snapshot;

  if (result) {
    delete result.up[id];
    delete result.self[id];
    delete result.lag[id];
    delete result.oplogWindow[id];
    delete result.oplogFill[id];
    delete result.versions[id];
    result.lastUpAt[id] = Math.floor((now - sinceMs) / 1000);
  }
  snapshot.services = (snapshot.services ?? []).map((service) =>
    service.serviceId === id
      ? { ...service, inventoryStatus: 'unknown' }
      : service
  );
};

/** A secondary this many seconds behind its primary, sustained. */
export const fallBehind = (
  snapshot: MongoSnapshot,
  name: string,
  seconds: number
) => {
  if (snapshot.metrics) {
    snapshot.metrics.lag[serviceId(name)] = {
      now: seconds,
      sustained: seconds - 2,
    };
  }
};

/** A set's oplog window, nearly full, on every member that has one. */
export const shortenOplog = (
  snapshot: MongoSnapshot,
  set: string,
  hours: number
) => {
  const { metrics: result } = snapshot;

  FLEET_MEMBERS.filter((member) => member.replicationSet === set).forEach(
    (member) => {
      const id = serviceId(member.name);

      if (result && result.oplogWindow[id] !== undefined) {
        result.oplogWindow[id] = hours * 3600;
        result.oplogFill[id] = 0.95;
      }
    }
  );
};

/** A member in a state that can't serve reads, past its grace period. */
export const holdInState = (
  snapshot: MongoSnapshot,
  name: string,
  state: string,
  now: number,
  sinceMs: number
) => {
  const id = serviceId(name);
  const { metrics: result } = snapshot;

  if (result) {
    result.self[id] = { ...result.self[id], state };
    result.stateSince[id] = Math.floor((now - sinceMs) / 1000);
    delete result.lag[id];
  }
};

/** A member left on an older release than the rest of its set. */
export const leaveOnVersion = (
  snapshot: MongoSnapshot,
  name: string,
  version: string = PATCH_BEHIND
) => {
  if (snapshot.metrics) {
    snapshot.metrics.versions[serviceId(name)] = version;
  }
};

/** A cluster with no backup schedule and no backup taken. */
export const unprotect = (snapshot: MongoSnapshot, cluster: string) => {
  if (!snapshot.backups) {
    return;
  }
  const own = new Set(fleetMembers(cluster));

  snapshot.backups.scheduled = snapshot.backups.scheduled.filter(
    (schedule) => !own.has(schedule.serviceName)
  );
  snapshot.backups.artifacts = snapshot.backups.artifacts.filter(
    (artifact) => !own.has(artifact.serviceName)
  );
};

/** A cluster's newest backup failed `failedMs` ago; the one before it succeeded `goodMs` ago. */
export const failBackup = (
  snapshot: MongoSnapshot,
  cluster: string,
  now: number,
  failedMs: number,
  goodMs: number
) => {
  if (!snapshot.backups) {
    return;
  }
  const own = new Set(fleetMembers(cluster));
  const artifact = snapshot.backups.artifacts.find(
    (item) => own.has(item.serviceName) && item.mode === 'snapshot'
  );

  if (!artifact) {
    return;
  }
  artifact.createdAt = ago(now, goodMs);
  snapshot.backups.artifacts.push({
    ...artifact,
    artifactId: `${artifact.artifactId}-failed`,
    name: artifact.name.replace(/ 1$/, ' 2'),
    status: 'error',
    createdAt: ago(now, failedMs),
  });
};

/** A cluster's point-in-time recovery stopped `sinceMs` ago. */
export const stopPitr = (
  snapshot: MongoSnapshot,
  cluster: string,
  now: number,
  sinceMs: number
) => {
  const artifact = snapshot.backups?.artifacts.find(
    (item) => item.artifactId === `art-${cluster}-pitr`
  );

  if (artifact) {
    artifact.status = 'error';
    artifact.createdAt = ago(now, sinceMs);
  }
};
