import {
  alertsLink,
  grafanaDashboard,
  listEnvironment,
  qanLink,
  realtimeLink,
  withIncidentFrame,
  withQuery,
} from 'hubs/template/hub.doors';
import type { HubDoor } from 'hubs/template/hub.types';
import { Messages } from './mongodb.messages';

/** `uid/slug` of each MongoDB dashboard the hub opens. */
export const MONGO_DASHBOARDS = {
  instancesOverview: 'mongodb-instance-overview/mongodb-instances-overview',
  instanceSummary: 'mongodb-instance-summary/mongodb-instance-summary',
  replsetSummary: 'mongodb-replicaset-summary/mongodb-replset-summary',
  oplogDetails: 'mongodb-oplog-details/mongodb-oplog-details',
  clusterSummary: 'mongodb-cluster-summary/mongodb-sharded-cluster-summary',
  routerSummary: 'mongodb-router-summary/mongodb-router-summary',
  backupStatus: 'mongodb-backup-details/mongodb-backup-details',
  collectionsOverview:
    'mongodb-collections-overview/mongodb-collections-overview',
  unusedIndexes: 'mongodb-unused-indexes/mongodb-unused-indexes',
} as const;

export type MongoDashboardId = keyof typeof MONGO_DASHBOARDS;

export const MONGO_PATHS = {
  addService: '/graph/add-instance/mongodb',
  install: '/operations/nodes?filter=unmonitored',
  backupInventory: '/graph/backup/inventory',
  backupScheduled: '/graph/backup/scheduled',
  backupNew: '/graph/backup/new',
  advancedSettings: '/settings/advanced-settings',
  operations: '/operations',
  operationsClusters: '/operations?tab=clusters',
  operationsAutomations: '/operations/automations',
  realtime: '/rta',
  allApps: '/prototype/all-apps',
  clusters: '/mongodb/clusters',
  inventoryServices: '/graph/inventory/services',
  advisors: '/graph/advisors/insights',
  documentation:
    'https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/install-pmm-client/connect-database/mongodb.html',
} as const;

/** Where an entity sits: its own environment and cluster, whatever the filter. */
export interface MongoPlace {
  environment?: string;
  cluster?: string;
}

const placeVariables = ({ environment, cluster }: MongoPlace) => ({
  environment,
  cluster,
});

export const instanceSummary = (place: MongoPlace, serviceName: string) =>
  grafanaDashboard(MONGO_DASHBOARDS.instanceSummary, {
    ...placeVariables(place),
    service_name: serviceName,
  });

export const replsetSummary = (place: MongoPlace, rs: string) =>
  grafanaDashboard(MONGO_DASHBOARDS.replsetSummary, {
    ...placeVariables(place),
    rs_nm: rs,
  });

export const clusterSummary = (place: MongoPlace) =>
  grafanaDashboard(MONGO_DASHBOARDS.clusterSummary, placeVariables(place));

export const routerSummary = (place: MongoPlace, serviceName?: string) =>
  grafanaDashboard(MONGO_DASHBOARDS.routerSummary, {
    ...placeVariables(place),
    service_name: serviceName,
  });

/** Its `service_name` is single-select without All, so a set's door names its primary. */
export const oplogDetails = (
  place: MongoPlace,
  rs: string,
  serviceName?: string
) =>
  grafanaDashboard(MONGO_DASHBOARDS.oplogDetails, {
    ...placeVariables(place),
    replset: rs,
    service_name: serviceName,
  });

export const backupStatus = (place: MongoPlace) =>
  grafanaDashboard(MONGO_DASHBOARDS.backupStatus, placeVariables(place));

/** A dashboard as a list door: the environment filter only. */
export const listDashboard = (id: MongoDashboardId, filter: string | null) =>
  grafanaDashboard(MONGO_DASHBOARDS[id], {
    environment: listEnvironment(filter),
  });

export const clusterQan = (place: MongoPlace, serviceName?: string) =>
  qanLink({ serviceType: 'mongodb', ...place, serviceName });

export const queryQan = (filter: string | null, queryId: string) =>
  qanLink({
    serviceType: 'mongodb',
    environment: listEnvironment(filter),
    queryId,
  });

export const lagDoor = (
  place: MongoPlace,
  rs: string,
  name: string
): HubDoor => ({
  label: name,
  to: withQuery(replsetSummary(place, rs), { from: 'now-3h', to: 'now' }),
  context: Messages.doors.lagOf(name),
});

export const oplogDoor = (
  place: MongoPlace,
  rs: string,
  name: string,
  serviceName?: string
): HubDoor => ({
  label: name,
  to: oplogDetails(place, rs, serviceName),
  context: Messages.doors.oplogOf(name),
});

export const incident = (to: string, since?: string) =>
  withIncidentFrame(to, since);

export const alertDoor = (serviceName?: string): HubDoor => ({
  label: Messages.doors.alert,
  to: alertsLink(serviceName),
  ...(serviceName ? { context: Messages.doors.forSubject(serviceName) } : {}),
});

export const runningDoor = (
  serviceId: string,
  serviceName: string
): HubDoor => ({
  label: Messages.doors.openRealtime,
  to: realtimeLink([serviceId]),
  context: Messages.doors.forSubject(serviceName),
});

/** A failed run in Automations, named after its run so two of them never share a name. */
export const automationsDoor = (
  kind: 'install' | 'scan',
  runId: string,
  target = ''
): HubDoor => ({
  label: Messages.doors.openAutomations,
  to: withQuery(MONGO_PATHS.operationsAutomations, {
    tab: kind === 'install' ? 'installs' : 'scans',
    expand: runId,
  }),
  context:
    kind === 'install'
      ? Messages.doors.forInstall(target)
      : Messages.doors.forScan,
});

export const manageDoor = (omClusterId?: string): HubDoor => ({
  label: Messages.doors.manage,
  to: withQuery(MONGO_PATHS.operations, {
    tab: 'clusters',
    cluster: omClusterId,
  }),
});
