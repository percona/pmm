import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import TrackChanges from '@mui/icons-material/TrackChanges';
import { Graph4Icon, PerconaMoIcon } from '@percona/peak-ui';
import { ExtensionIcon } from 'lib/peak-ui-pending-icons';
import type {
  HubDestination,
  HubDestinationLook,
} from 'hubs/template/hub.types';
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
  /** The Clusters page, filtered to the clusters a row is about. */
  affectedClusters: (clusters: string[]): HubDestination => ({
    label: D.affectedClusters,
    route: MONGO_PATHS.clusters,
    params: { clusters: clusters.join(',') },
    kind: 'page',
  }),
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
  [MONGO_PATHS.realtime]: { icon: TrackChanges, path: [D.realtime] },
  [MONGO_PATHS.operations]: {
    icon: ExtensionIcon,
    path: [P.apps, P.operations, D.fleet],
  },
  [MONGO_PATHS.clusters]: {
    icon: PerconaMoIcon,
    path: [MONGODB_HUB_MESSAGES.name, P.clustersFiltered],
  },
};

/** The sidebar's icon for a destination and its place there; another dashboard gets the dashboard icon. */
export const describeMongoDestination = ({
  label,
  route,
}: HubDestination): HubDestinationLook =>
  LOOKS[route] ??
  (route.startsWith('/graph/d/')
    ? { icon: DashboardOutlined, path: [P.dashboards, label] }
    : { path: [label] });
