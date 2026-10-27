import { resolveHubCapabilities } from 'hubs/template/hub.params';
import type {
  HubDeriveContext,
  HubRole,
  HubScenario,
  HubSourceId,
  HubSourceState,
} from 'hubs/template/hub.types';
import { serviceId } from './fixtures/estate';
import type { MongoMetrics, MongoServiceInput } from './mongodb.types';

export const NOW = Date.parse('2026-10-19T12:00:00.000Z');

const SOURCE_IDS: HubSourceId[] = [
  'inventory',
  'metrics',
  'alerts',
  'backups',
  'qan',
  'realtime',
  'operations',
  'dashboards',
];

export const sourcesWith = (
  status: HubSourceState['status'],
  overrides: Partial<Record<HubSourceId, HubSourceState>> = {}
): Record<HubSourceId, HubSourceState> => ({
  ...(Object.fromEntries(SOURCE_IDS.map((id) => [id, { status }])) as Record<
    HubSourceId,
    HubSourceState
  >),
  ...overrides,
});

export interface ContextOptions {
  scenario?: HubScenario;
  role?: HubRole;
  operations?: boolean;
  environment?: string | null;
  extensions?: boolean;
  backupManagementEnabled?: boolean;
  sources?: Record<HubSourceId, HubSourceState>;
}

/** A derive context as useTechnologyHub builds it, for sample data unless told otherwise. */
export const deriveContext = ({
  scenario = 'unhealthy',
  role = 'admin',
  operations = false,
  environment = null,
  extensions = true,
  backupManagementEnabled = true,
  sources = sourcesWith('disabled'),
}: ContextOptions = {}): HubDeriveContext => {
  const ops = role === 'admin' && operations;

  return {
    now: NOW,
    scenario,
    environment,
    role,
    operations: ops,
    capabilities: resolveHubCapabilities({
      role,
      operations: ops,
      backupManagementEnabled,
      extensionsEnabled: extensions,
      signedIn: true,
    }),
    sources,
  };
};

export const service = (
  name: string,
  overrides: Partial<MongoServiceInput> = {}
): MongoServiceInput => ({
  serviceId: serviceId(name),
  serviceName: name,
  nodeName: name,
  environment: 'prod',
  inventoryStatus: 'up',
  ...overrides,
});

export const emptyMetrics = (): MongoMetrics => ({
  up: {},
  lastUpAt: {},
  self: {},
  roles: {},
  stateSince: {},
  versions: {},
  lag: {},
  oplogWindow: {},
  oplogFill: {},
  shardsByCluster: {},
  pbmConfiguredByCluster: {},
});

/** One replica set's members up, the first primary, with lag and oplog as given. */
export const replicaSet = (
  name: string,
  {
    cluster = name,
    environment = 'prod',
    members = 3,
    lag = 0,
    window = 40 * 3600,
    fill = 1,
    clRole,
  }: {
    cluster?: string;
    environment?: string;
    members?: number;
    lag?: number;
    window?: number;
    fill?: number;
    clRole?: 'configsvr' | 'shardsvr';
  } = {}
) => {
  const services = Array.from({ length: members }, (_, index) =>
    service(`${name}-m${index + 1}`, {
      environment,
      cluster,
      replicationSet: name,
    })
  );
  const metrics = emptyMetrics();

  services.forEach((item, index) => {
    const id = item.serviceId;

    metrics.up[id] = { value: 1, clusterRole: 'mongod' };
    metrics.lastUpAt[id] = NOW / 1000;
    metrics.self[id] = {
      rsName: name,
      state: index === 0 ? 'PRIMARY' : 'SECONDARY',
      ...(clRole ? { clRole } : {}),
    };
    metrics.roles[id] = { rsName: name, ...(clRole ? { clRole } : {}) };
    metrics.stateSince[id] = NOW / 1000 - 3600;
    metrics.oplogWindow[id] = window;
    metrics.oplogFill[id] = fill;
    if (index > 0) {
      metrics.lag[id] = { now: lag, sustained: lag };
    }
  });

  return { services, metrics };
};

/** Merges metrics of several sets into one. */
export const mergeMetrics = (...all: MongoMetrics[]): MongoMetrics =>
  all.reduce(
    (merged, metrics) => ({
      up: { ...merged.up, ...metrics.up },
      lastUpAt: { ...merged.lastUpAt, ...metrics.lastUpAt },
      self: { ...merged.self, ...metrics.self },
      roles: { ...merged.roles, ...metrics.roles },
      stateSince: { ...merged.stateSince, ...metrics.stateSince },
      versions: { ...merged.versions, ...metrics.versions },
      lag: { ...merged.lag, ...metrics.lag },
      oplogWindow: { ...merged.oplogWindow, ...metrics.oplogWindow },
      oplogFill: { ...merged.oplogFill, ...metrics.oplogFill },
      shardsByCluster: {
        ...merged.shardsByCluster,
        ...metrics.shardsByCluster,
      },
      pbmConfiguredByCluster: {
        ...merged.pbmConfiguredByCluster,
        ...metrics.pbmConfiguredByCluster,
      },
    }),
    emptyMetrics()
  );
