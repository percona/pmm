import { higherSeverity, rankAttention } from 'hubs/template/hub.attention';
import { ALERTS_PATH } from 'hubs/template/hub.doors';
import { Messages as HubText } from 'hubs/template/hub.messages';
import {
  environmentOptions,
  matchesEnvironment,
} from 'hubs/template/hub.params';
import type {
  HubDeriveContext,
  HubParams,
  HubSectionId,
  HubSectionState,
  HubSourceId,
  HubSourceState,
  HubStatus,
  HubViewModel,
} from 'hubs/template/hub.types';
import { deriveApps } from './mongodb.apps';
import {
  type AttentionEntry,
  deriveAttention,
  MONGODB_ATTENTION_ORDER,
} from './mongodb.attention';
import { deriveBackups } from './mongodb.backups';
import { deriveDashboards } from './mongodb.dashboards';
import { listDashboard, MONGO_PATHS, sectionAnchor } from './mongodb.doors';
import { worstHealth } from './mongodb.health';
import { Messages, MONGODB_HUB_MESSAGES } from './mongodb.messages';
import { deriveQueries } from './mongodb.queries';
import {
  buildClusterTree,
  type ClusterModel,
  clusterMembers,
  expandedProblems,
  groupClusters,
} from './mongodb.topology';
import type {
  MongoAlertInput,
  MongoServiceInput,
  MongoSnapshot,
} from './mongodb.types';

const READY: HubSectionState = { status: 'ready' };

/** A section is as good as the worst of the sources it reads; a disabled source is not a problem. */
const sectionState = (
  sources: Partial<Record<HubSourceId, HubSourceState>>,
  ids: HubSourceId[]
): HubSectionState => {
  for (const id of ids) {
    const source = sources[id];

    if (source?.status === 'error') {
      return {
        status: 'error',
        source: id,
        ...(source.message ? { message: source.message } : {}),
      };
    }
  }

  return ids.some((id) => sources[id]?.status === 'loading')
    ? { status: 'loading' }
    : READY;
};

/** An alert's environment: its own label, else that of the service or cluster it names. */
const alertEnvironment = (
  alert: MongoAlertInput,
  services: MongoServiceInput[]
): string | undefined | null => {
  if (alert.environment) {
    return alert.environment;
  }
  const service = services.find(
    (item) =>
      (alert.serviceName && item.serviceName === alert.serviceName) ||
      (!alert.serviceName && alert.cluster && item.cluster === alert.cluster)
  );

  return service ? (service.environment ?? '') : null;
};

const statusState = ({
  empty,
  metricsOk,
  shown,
  clusters,
}: {
  empty: boolean;
  metricsOk: boolean;
  shown: number;
  clusters: HubViewModel['clusters'];
}): HubStatus['state'] => {
  if (empty) {
    return 'empty';
  }
  if (!metricsOk || !shown) {
    return 'noData';
  }

  return worstHealth(clusters.map((node) => node.health));
};

/** The cluster an attention row is about, if any: its own, its service's, or the one its alert names. */
const clusterKeyOf = (
  { subject }: AttentionEntry,
  clusterByService: Map<string, ClusterModel>,
  clusters: ClusterModel[]
): string | undefined => {
  if (subject.clusterKey) {
    return subject.clusterKey;
  }
  const byService = subject.serviceNames
    .map((name) => clusterByService.get(name)?.key)
    .find(Boolean);

  return (
    byService ??
    clusters.find((cluster) => cluster.name === subject.cluster)?.key
  );
};

/** The MongoDB snapshot as the hub's view: pure, so every rule is testable without React. */
export const deriveMongoHub = (
  snapshot: MongoSnapshot,
  ctx: HubDeriveContext
): HubViewModel => {
  const filter = ctx.environment;
  const { capabilities, sources } = ctx;
  const params: HubParams = ctx.params ?? {
    environment: filter,
    scenario: ctx.scenario,
    ops: 'live',
    role: 'live',
  };
  const allServices = snapshot.services ?? [];
  const services = allServices.filter((service) =>
    matchesEnvironment(service.environment, filter)
  );
  const metricsOk = snapshot.metrics !== null;
  const models = groupClusters(services, snapshot.metrics);
  const operations = ctx.operations ? snapshot.operations : null;
  const clusters = metricsOk
    ? buildClusterTree(models, {
        capabilities,
        omClusterIds: operations?.clusterIds,
      })
    : [];

  const backups = deriveBackups({
    clusters: models,
    input: snapshot.backups,
    metrics: snapshot.metrics,
    capabilities,
    role: ctx.role,
    filter,
    now: ctx.now,
  });
  const queries = deriveQueries(snapshot, services, filter);
  const alerts = (snapshot.alerts ?? []).filter((alert) => {
    const environment = alertEnvironment(alert, allServices);

    return environment === null
      ? filter === null
      : matchesEnvironment(environment, filter);
  });

  const entries = deriveAttention({
    clusters: metricsOk ? models : [],
    alerts,
    backupFindings: backups.findings,
    slower: queries.slower,
    running: queries.running,
    operations,
    filter,
    now: ctx.now,
  });
  const clusterByService = new Map<string, ClusterModel>();

  models.forEach((cluster) =>
    clusterMembers(cluster).forEach((member) =>
      clusterByService.set(member.service.serviceName, cluster)
    )
  );
  const troubled = new Set<string>(
    clusters
      .filter((node) => node.health !== 'healthy')
      .map((node) => node.id.replace(/^cluster:/, ''))
  );

  let otherIssues = 0;

  entries.forEach((entry) => {
    const key = clusterKeyOf(entry, clusterByService, models);

    if (key) {
      troubled.add(key);
    } else {
      otherIssues += 1;
    }
  });
  const attentionSeverity = entries.reduce<HubStatus['attentionSeverity']>(
    (worst, { item }) =>
      worst ? higherSeverity(worst, item.severity) : item.severity,
    undefined
  );

  // Nothing monitored at all; a filter that matches nothing is not an empty estate.
  const empty =
    snapshot.services !== null &&
    !allServices.length &&
    sources.inventory?.status !== 'loading';
  // Without the inventory there are no clusters to count or map backups to, which is not zero.
  const inventoryOk = snapshot.services !== null;
  const status: HubStatus = {
    state: statusState({ empty, metricsOk, shown: services.length, clusters }),
    clusters: inventoryOk ? models.length : null,
    needAttention: metricsOk && inventoryOk ? troubled.size : null,
    otherIssues,
    ...(attentionSeverity ? { attentionSeverity } : {}),
    lastGoodBackup: inventoryOk ? backups.lastGoodBackup : 'unknown',
    checkedAt: new Date(snapshot.fetchedAt || ctx.now).toISOString(),
    doors: {
      clusters: {
        label: Messages.doors.clusters,
        to: sectionAnchor(params, 'clusters'),
      },
      attention: {
        label: Messages.doors.needsAttention,
        to: sectionAnchor(params, 'needs-attention'),
      },
      backup:
        backups.lastGoodBackup === 'notSetUp' && capabilities.setUpBackup
          ? { label: Messages.doors.setUpBackup, to: MONGO_PATHS.backupNew }
          : {
              label: Messages.doors.lastGoodBackup,
              to: MONGO_PATHS.backupInventory,
            },
    },
  };

  const hasBackups =
    backups.backups.state === 'ok' ||
    Object.values(snapshot.metrics?.pbmConfiguredByCluster ?? {}).some(Boolean);

  // Alerts add rows to what inventory and metrics already say, so their failure keeps those rows.
  const attentionState = sectionState(sources, ['inventory', 'metrics']);
  const sections: Record<HubSectionId, HubSectionState> = {
    attention:
      attentionState.status === 'ready' && sources.alerts?.status === 'error'
        ? {
            status: 'error',
            source: 'alerts',
            message: HubText.sectionError.alerts(sources.alerts.message),
            partial: true,
          }
        : attentionState,
    clusters: sectionState(sources, ['inventory', 'metrics']),
    backups: sectionState(sources, ['inventory', 'backups']),
    queries: sectionState(sources, ['qan', 'realtime']),
    dashboards: sectionState(sources, ['dashboards']),
    apps: sectionState(sources, ctx.operations ? ['operations'] : []),
  };

  return {
    technology: { id: 'mongodb', name: MONGODB_HUB_MESSAGES.name },
    scenario: ctx.scenario,
    role: ctx.role,
    capabilities,
    environment: {
      selected: filter,
      options: environmentOptions(
        allServices.map((service) => service.environment),
        Messages.noEnvironment,
        filter
      ),
      matchesNothing:
        filter !== null && allServices.length > 0 && services.length === 0,
    },
    status,
    attention: rankAttention(
      entries.map((entry) => entry.item),
      MONGODB_ATTENTION_ORDER
    ),
    clusters,
    expanded: expandedProblems(clusters),
    backups: backups.backups,
    queries: queries.queries,
    dashboards: deriveDashboards({
      clusters: models,
      dashboards: snapshot.dashboards,
      hasBackups,
      filter,
    }),
    apps: deriveApps({
      capabilities,
      emptyEstate: empty,
      operations,
      backups: {
        featureOff: backups.backups.state === 'featureOff',
        notSetUp: backups.backups.state === 'notSetUp',
        schedules: backups.schedules,
        failedRecently: backups.failedRecently,
        lastGoodBackup: backups.lastGoodBackup,
      },
      now: ctx.now,
    }),
    actions: {
      ...(capabilities.addService
        ? {
            addService: {
              label: Messages.doors.addService,
              to: MONGO_PATHS.addService,
            },
          }
        : {}),
      ...(capabilities.install
        ? {
            install: {
              label: Messages.doors.install,
              to: MONGO_PATHS.install,
              preview: 'developer' as const,
            },
          }
        : {}),
    },
    sectionDoors: {
      attention: { label: Messages.doors.alerts, to: ALERTS_PATH },
      clusters: capabilities.operations
        ? {
            label: Messages.doors.fleet,
            to: MONGO_PATHS.operationsClusters,
            preview: 'developer',
          }
        : {
            label: Messages.doors.instancesOverview,
            to: listDashboard('instancesOverview', filter),
          },
      apps: { label: Messages.doors.allApps, to: MONGO_PATHS.allApps },
    },
    sections,
  };
};
