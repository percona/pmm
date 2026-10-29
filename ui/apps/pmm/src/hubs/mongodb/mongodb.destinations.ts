import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import HelpOutline from '@mui/icons-material/HelpOutline';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';
import TrackChanges from '@mui/icons-material/TrackChanges';
import {
  Graph4Icon,
  NetworkIntelligenceIcon,
  PerconaMoIcon,
  QueryStatsIcon,
} from '@percona/peak-ui';
import { ExtensionIcon } from 'lib/peak-ui-pending-icons';
import type {
  HubDestination,
  HubDestinationLook,
} from 'hubs/template/hub.types';
import {
  ALERT_TEMPLATES_PATH,
  listEnvironment,
  QAN_DASHBOARD,
} from 'hubs/template/hub.doors';
import { MONGO_PATHS } from './mongodb.doors';
import { Messages, MONGODB_HUB_MESSAGES } from './mongodb.messages';

const D = Messages.destinations;
const P = D.places;

const REPLSET_SUMMARY =
  '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary';
const SHARDED_SUMMARY =
  '/graph/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary';
const INSTANCES_OVERVIEW =
  '/graph/d/mongodb-instance-overview/mongodb-instances-overview';
const INVENTORY = '/graph/inventory';
const QUERY_ANALYTICS = `/graph/d/${QAN_DASHBOARD}`;
const QUERY_SOURCE_DOCS =
  'https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/install-pmm-client/connect-database/mongodb.html#step-2-configure-query-source-for-mongodb-query-analytics';
const UPGRADE_60_DOCS =
  'https://docs.percona.com/percona-server-for-mongodb/7.0/install/upgrade-from-60.html';

/** Where a Needs attention row can send the reader. */
export const MongoDestinations = {
  replsetSummary: (
    environment: string,
    cluster: string,
    set: string = cluster
  ): HubDestination => ({
    label: D.replsetSummary,
    route: REPLSET_SUMMARY,
    params: {
      'var-environment': environment,
      'var-cluster': cluster,
      'var-rs_nm': set,
    },
    kind: 'page',
  }),
  shardedSummary: (environment: string, cluster: string): HubDestination => ({
    label: D.shardedSummary,
    route: SHARDED_SUMMARY,
    params: { 'var-environment': environment, 'var-cluster': cluster },
    kind: 'page',
  }),
  backups: (): HubDestination => ({
    label: D.backups,
    route: MONGO_PATHS.backupInventory,
    kind: 'page',
  }),
  scheduledBackups: (): HubDestination => ({
    label: D.scheduledBackups,
    route: MONGO_PATHS.backupScheduled,
    kind: 'page',
  }),
  inventory: (): HubDestination => ({
    label: D.inventory,
    route: INVENTORY,
    kind: 'page',
  }),
  realtime: (): HubDestination => ({
    label: D.realtime,
    route: MONGO_PATHS.realtime,
    kind: 'page',
  }),
  /** Operations for MongoDB's Fleet, or the Instances overview dashboard while Operations is off. */
  fleet: (operations: boolean): HubDestination =>
    operations
      ? { label: D.fleet, route: MONGO_PATHS.operations, kind: 'page' }
      : {
          label: D.instancesOverview,
          route: INSTANCES_OVERVIEW,
          kind: 'page',
        },
  planInFleet: (): HubDestination => ({
    label: D.planInFleet,
    route: MONGO_PATHS.operations,
    kind: 'page',
  }),
  turnOnPitr: (): HubDestination => ({
    label: D.turnOnPitr,
    route: MONGO_PATHS.backupScheduled,
    kind: 'page',
  }),
  setUpBackups: (): HubDestination => ({
    label: D.setUpBackups,
    route: MONGO_PATHS.backupNew,
    kind: 'page',
  }),
  advisors: (): HubDestination => ({
    label: D.advisors,
    route: MONGO_PATHS.advisors,
    kind: 'page',
  }),
  alertTemplates: (): HubDestination => ({
    label: D.alertTemplates,
    route: ALERT_TEMPLATES_PATH,
    kind: 'page',
  }),
  /** Percona's documentation on MongoDB's query source, which Query analytics reads. */
  howToEnableQan: (): HubDestination => ({
    label: D.howToEnable,
    route: QUERY_SOURCE_DOCS,
    kind: 'external',
  }),
  /** Percona's documentation on the next upgrade from 6.0. */
  upgradeGuide: (): HubDestination => ({
    label: D.upgradeGuide,
    route: UPGRADE_60_DOCS,
    kind: 'external',
  }),
  /** Advanced settings, where Operations for MongoDB is turned on. */
  turnOnOperations: (): HubDestination => ({
    label: D.turnOnOperations,
    route: MONGO_PATHS.advancedSettings,
    kind: 'page',
  }),
  /** The Clusters page, filtered to the clusters a row is about. */
  affectedClusters: (clusters: string[]): HubDestination => ({
    label: D.affectedClusters,
    route: MONGO_PATHS.clusters,
    params: { clusters: clusters.join(',') },
    kind: 'page',
  }),
};

/** The hub's shortcut tiles, in order, with its environment filter where the page takes one; Fleet only with Operations on. */
export const mongoShortcuts = (
  filter: string | null,
  operations: boolean
): HubDestination[] => {
  const environment = listEnvironment(filter);
  const inEnvironment: Record<string, string> = environment
    ? { 'var-environment': environment }
    : {};

  return [
    { label: D.clusters, route: MONGO_PATHS.clusters, kind: 'page' },
    {
      label: D.instancesOverview,
      route: INSTANCES_OVERVIEW,
      params: inEnvironment,
      kind: 'page',
    },
    {
      label: D.queryAnalytics,
      route: QUERY_ANALYTICS,
      params: { 'var-service_type': 'mongodb', ...inEnvironment },
      kind: 'page',
    },
    MongoDestinations.realtime(),
    MongoDestinations.backups(),
    ...(operations ? [MongoDestinations.fleet(true)] : []),
  ];
};

const dashboard = (title: string): HubDestinationLook => ({
  icon: DashboardOutlined,
  path: [P.dashboards, P.folder, title],
});

const LOOKS: Record<string, HubDestinationLook> = {
  [REPLSET_SUMMARY]: dashboard(P.replsetSummary),
  [SHARDED_SUMMARY]: dashboard(P.shardedSummary),
  [INSTANCES_OVERVIEW]: dashboard(P.instancesOverview),
  [MONGO_PATHS.backupInventory]: {
    icon: ExtensionIcon,
    path: [P.apps, D.backups],
  },
  [MONGO_PATHS.backupScheduled]: {
    icon: ExtensionIcon,
    path: [P.apps, D.backups, D.scheduledBackups],
  },
  [INVENTORY]: { icon: Graph4Icon, path: [D.inventory] },
  [QUERY_ANALYTICS]: { icon: QueryStatsIcon, path: [D.queryAnalytics] },
  [MONGO_PATHS.realtime]: { icon: TrackChanges, path: [D.realtime] },
  [MONGO_PATHS.operations]: {
    icon: ExtensionIcon,
    path: [P.apps, P.operations, D.fleet],
  },
  [MONGO_PATHS.backupNew]: {
    icon: ExtensionIcon,
    path: [P.apps, D.backups, P.newBackup],
  },
  [MONGO_PATHS.advisors]: { icon: NetworkIntelligenceIcon, path: [D.advisors] },
  [ALERT_TEMPLATES_PATH]: {
    icon: NotificationsOutlined,
    path: [P.alerts, P.templates],
  },
  [MONGO_PATHS.advancedSettings]: {
    icon: SettingsOutlined,
    path: [P.settings, P.advancedSettings],
  },
  [QUERY_SOURCE_DOCS]: { icon: HelpOutline, path: [P.docs, P.queryCollection] },
  [UPGRADE_60_DOCS]: { icon: HelpOutline, path: [P.docs, P.upgrade60] },
};

const clusters = (filtered: boolean): HubDestinationLook => ({
  icon: PerconaMoIcon,
  path: [MONGODB_HUB_MESSAGES.name, filtered ? P.clustersFiltered : P.clusters],
});

/** The sidebar's icon for a destination and its place there; another dashboard gets the dashboard icon. */
export const describeMongoDestination = ({
  label,
  route,
  params,
}: HubDestination): HubDestinationLook =>
  (route === MONGO_PATHS.clusters ? clusters(!!params?.clusters) : undefined) ??
  LOOKS[route] ??
  (route.startsWith('/graph/d/')
    ? { icon: DashboardOutlined, path: [P.dashboards, label] }
    : { path: [label] });
