import type {
  DashboardHit,
  MongoBackupsInput,
  MongoMetrics,
  MongoOperationsInput,
  MongoServiceInput,
} from '../mongodb.types';

// The recorded estate's names (recorded.ts), so doors land on dashboards with data; orders-mongos-2 and orders-cfg-m2/m3 exist only here.

export const MINUTE_MS = 60 * 1000;
export const HOUR_MS = 60 * MINUTE_MS;

export const serviceId = (name: string) => `svc-${name}`;

export const ago = (now: number, ms: number) =>
  new Date(now - ms).toISOString();

type Role = 'router' | 'configsvr' | 'shardsvr' | 'replica' | 'standalone';

interface EstateMember {
  name: string;
  environment: string;
  cluster: string;
  replicationSet?: string;
  role: Role;
  state?: 'PRIMARY' | 'SECONDARY';
}

const set = (
  environment: string,
  cluster: string,
  name: string,
  role: Role
): EstateMember[] =>
  ['m1', 'm2', 'm3'].map((member, index) => ({
    name: `${name}-${member}`,
    environment,
    cluster,
    replicationSet: name,
    role,
    state: index === 0 ? 'PRIMARY' : 'SECONDARY',
  }));

export const ESTATE: EstateMember[] = [
  {
    name: 'orders-mongos',
    environment: 'prod',
    cluster: 'orders',
    role: 'router',
  },
  {
    name: 'orders-mongos-2',
    environment: 'prod',
    cluster: 'orders',
    role: 'router',
  },
  ...set('prod', 'orders', 'orders-cfg', 'configsvr'),
  ...set('prod', 'orders', 'orders-shard1', 'shardsvr'),
  ...set('prod', 'orders', 'orders-shard2', 'shardsvr'),
  ...set('prod', 'billing', 'billing', 'replica'),
  {
    name: 'reports',
    environment: 'staging',
    cluster: 'reports',
    role: 'standalone',
  },
];

export const estateServices = (): MongoServiceInput[] =>
  ESTATE.map((member) => ({
    serviceId: serviceId(member.name),
    serviceName: member.name,
    nodeName: member.name,
    environment: member.environment,
    cluster: member.cluster,
    ...(member.replicationSet ? { replicationSet: member.replicationSet } : {}),
    inventoryStatus: 'up',
  }));

const LAG_SECONDS: Record<string, number> = {
  'orders-cfg-m2': 0,
  'orders-cfg-m3': 0,
  'orders-shard1-m2': 1,
  'orders-shard1-m3': 2,
  'orders-shard2-m2': 0,
  'orders-shard2-m3': 1,
  'billing-m2': 0,
  'billing-m3': 1,
};

const WINDOW_SECONDS: Record<Role, number | undefined> = {
  router: undefined,
  configsvr: 40 * 3600,
  shardsvr: 26 * 3600,
  replica: 40 * 3600,
  standalone: undefined,
};

/** Every member up, lag 0 to 2 s, shard windows 26 h, config and billing 40 h, oplogs full. */
export const healthyMetrics = (now: number): MongoMetrics => {
  const metrics: MongoMetrics = {
    up: {},
    lastUpAt: {},
    self: {},
    roles: {},
    lag: {},
    oplogWindow: {},
    oplogFill: {},
    shardsByCluster: { orders: 2 },
    pbmConfiguredByCluster: { orders: true, billing: true, reports: false },
  };

  ESTATE.forEach((member) => {
    const id = serviceId(member.name);
    const window = WINDOW_SECONDS[member.role];

    metrics.up[id] = {
      value: 1,
      clusterRole: member.role === 'router' ? 'mongos' : 'mongod',
    };
    metrics.lastUpAt[id] = Math.floor(now / 1000);
    if (member.replicationSet && member.state) {
      const clRole =
        member.role === 'configsvr' || member.role === 'shardsvr'
          ? member.role
          : undefined;

      metrics.self[id] = {
        rsName: member.replicationSet,
        state: member.state,
        ...(clRole ? { clRole } : {}),
      };
      metrics.roles[id] = {
        rsName: member.replicationSet,
        ...(clRole ? { clRole } : {}),
      };
    }
    if (member.state === 'SECONDARY') {
      const lag = LAG_SECONDS[member.name] ?? 0;

      metrics.lag[id] = { now: lag, sustained: lag };
    }
    if (window !== undefined) {
      metrics.oplogWindow[id] = window;
      metrics.oplogFill[id] = 1;
    }
  });

  return metrics;
};

/** The next `hour`:00 UTC after `now`. */
export const nextRunAt = (now: number, hour: number) => {
  const date = new Date(now);

  date.setUTCHours(hour, 0, 0, 0);
  if (date.getTime() <= now) {
    date.setUTCDate(date.getUTCDate() + 1);
  }

  return date.toISOString();
};

export const ORDERS_SCHEDULE = 'nightly backup';
export const BILLING_SCHEDULE = 'daily backup';
export const BILLING_PITR = 'point-in-time recovery';

/** Orders nightly at 02:00, billing daily at 03:00 plus PITR; reports has none. */
export const healthyBackups = (now: number): MongoBackupsInput => ({
  enabled: true,
  scheduled: [
    {
      scheduledBackupId: 'sched-orders',
      name: ORDERS_SCHEDULE,
      serviceId: serviceId('orders-cfg-m1'),
      serviceName: 'orders-cfg-m1',
      mode: 'snapshot',
      enabled: true,
      cronExpression: '0 2 * * *',
      nextRun: nextRunAt(now, 2),
    },
    {
      scheduledBackupId: 'sched-billing',
      name: BILLING_SCHEDULE,
      serviceId: serviceId('billing-m1'),
      serviceName: 'billing-m1',
      mode: 'snapshot',
      enabled: true,
      cronExpression: '0 3 * * *',
      nextRun: nextRunAt(now, 3),
    },
    {
      scheduledBackupId: 'sched-billing-pitr',
      name: BILLING_PITR,
      serviceId: serviceId('billing-m1'),
      serviceName: 'billing-m1',
      mode: 'pitr',
      enabled: true,
      cronExpression: '',
    },
  ],
  artifacts: [
    {
      artifactId: 'art-billing-snapshot',
      name: `${BILLING_SCHEDULE} 1`,
      serviceId: serviceId('billing-m1'),
      serviceName: 'billing-m1',
      mode: 'snapshot',
      status: 'success',
      createdAt: ago(now, 6 * HOUR_MS),
    },
    {
      artifactId: 'art-orders-snapshot',
      name: `${ORDERS_SCHEDULE} 1`,
      serviceId: serviceId('orders-cfg-m1'),
      serviceName: 'orders-cfg-m1',
      mode: 'snapshot',
      status: 'success',
      createdAt: ago(now, 7 * HOUR_MS),
    },
    {
      artifactId: 'art-billing-pitr',
      name: BILLING_PITR,
      serviceId: serviceId('billing-m1'),
      serviceName: 'billing-m1',
      mode: 'pitr',
      status: 'inProgress',
      createdAt: ago(now, 6 * HOUR_MS),
    },
  ],
});

export const OM_CLUSTER_IDS: Record<string, string> = {
  'prod/orders': 'om-orders',
  'prod/billing': 'om-billing',
  'staging/reports': 'om-reports',
};

export const healthyOperations = (now: number): MongoOperationsInput => ({
  clusterIds: OM_CLUSTER_IDS,
  summary: { services: 15, up: 15, lastScanAt: ago(now, 4 * MINUTE_MS) },
  failedRuns: [],
});

export const DASHBOARD_FOLDER = 'mongodb-folder';

const dashboard = (uid: string, slug: string, title: string): DashboardHit => ({
  uid,
  title,
  url: `/graph/d/${uid}/${slug}`,
  folderUid: DASHBOARD_FOLDER,
});

/** The 13 MongoDB dashboards, plus the one stray title match PMM's own search returns. */
export const DASHBOARD_CATALOG: DashboardHit[] = [
  dashboard(
    'mongodb-backup-details',
    'mongodb-backup-details',
    'MongoDB Backup Details'
  ),
  dashboard(
    'mongodb-collections-overview',
    'mongodb-collections-overview',
    'MongoDB Collections Overview'
  ),
  dashboard(
    'mongodb-inmemory',
    'mongodb-inmemory-details',
    'MongoDB InMemory Details'
  ),
  dashboard(
    'mongodb-instance-summary',
    'mongodb-instance-summary',
    'MongoDB Instance Summary'
  ),
  dashboard(
    'mongodb-instance-compare',
    'mongodb-instances-compare',
    'MongoDB Instances Compare'
  ),
  dashboard(
    'mongodb-instance-overview',
    'mongodb-instances-overview',
    'MongoDB Instances Overview'
  ),
  dashboard(
    'mongodb-mmapv1',
    'mongodb-mmapv1-details',
    'MongoDB MMAPv1 Details'
  ),
  dashboard(
    'mongodb-oplog-details',
    'mongodb-oplog-details',
    'MongoDB Oplog Details'
  ),
  dashboard(
    'mongodb-replicaset-summary',
    'mongodb-replset-summary',
    'MongoDB ReplSet Summary'
  ),
  dashboard(
    'mongodb-router-summary',
    'mongodb-router-summary',
    'MongoDB Router Summary'
  ),
  dashboard(
    'mongodb-cluster-summary',
    'mongodb-sharded-cluster-summary',
    'MongoDB Sharded Cluster Summary'
  ),
  dashboard(
    'mongodb-unused-indexes',
    'mongodb-unused-indexes',
    'MongoDB Unused Indexes'
  ),
  dashboard(
    'mongodb-wiredtiger',
    'mongodb-wiredtiger-details',
    'MongoDB WiredTiger Details'
  ),
  {
    uid: 'prometheus-status',
    title: 'Prometheus Exporter Status',
    url: '/graph/d/prometheus-status/prometheus-exporter-status',
    folderUid: 'insight-folder',
  },
];

/** ReplSet summary 5 minutes ago, Oplog details a day ago, Instance summary two days ago. */
export const recentDashboards = (now: number) => [
  { uid: 'mongodb-replicaset-summary', visitedAt: ago(now, 5 * MINUTE_MS) },
  { uid: 'mongodb-oplog-details', visitedAt: ago(now, 24 * HOUR_MS) },
  { uid: 'mongodb-instance-summary', visitedAt: ago(now, 48 * HOUR_MS) },
];
