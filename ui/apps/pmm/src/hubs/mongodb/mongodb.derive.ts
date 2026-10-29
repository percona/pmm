import {
  attentionInEnvironment,
  recommendationsInEnvironment,
  sortNeedsAttention,
  sortRecommendations,
} from 'hubs/template/hub.attention';
import { ALERT_TEMPLATES_PATH, ALERTS_PATH } from 'hubs/template/hub.doors';
import { Messages as HubText } from 'hubs/template/hub.messages';
import {
  environmentOptions,
  matchesEnvironment,
} from 'hubs/template/hub.params';
import type {
  HubAlertCoverage,
  HubDeriveContext,
  HubHealth,
  HubRecommendation,
  HubSectionId,
  HubSectionState,
  HubSourceId,
  HubSourceState,
  HubStatus,
  HubViewModel,
} from 'hubs/template/hub.types';
import { deriveActions } from './mongodb.actions';
import { deriveAttention, toNeedsAttention } from './mongodb.attention';
import { deriveBackups } from './mongodb.backups';
import { deriveDashboards } from './mongodb.dashboards';
import { MongoDestinations, mongoShortcuts } from './mongodb.destinations';
import { MONGO_PATHS } from './mongodb.doors';
import { worstHealth } from './mongodb.health';
import { Messages, MONGODB_HUB_MESSAGES } from './mongodb.messages';
import { deriveQueries } from './mongodb.queries';
import { clusterHealthOf, groupClusters } from './mongodb.topology';
import type {
  MongoAlertInput,
  MongoAlertRulesInput,
  MongoServiceInput,
  MongoSnapshot,
} from './mongodb.types';

// The templates for what the health model watches most: a member gone, a secondary behind, the oplog running short.
const KEY_TEMPLATES: Array<[string, string]> = [
  ['pmm_mongodb_down', Messages.coverage.memberDown],
  ['pmm_mongodb_replication_lag', Messages.coverage.lag],
  ['pmm_mongodb_oplog_window', Messages.coverage.oplog],
];

/** No MongoDB template and no custom rule: nothing alerts on MongoDB yet. */
const noRules = (rules: MongoAlertRulesInput) =>
  !rules.templates.some((template) => template.startsWith('pmm_mongodb_')) &&
  rules.custom === 0;

const countHealth = (health: HubHealth[]): Record<HubHealth, number> => {
  const counts: Record<HubHealth, number> = {
    healthy: 0,
    degraded: 0,
    down: 0,
    upgrading: 0,
    noData: 0,
  };

  health.forEach((value) => {
    counts[value] += 1;
  });

  return counts;
};

/** What the alert rules leave uncovered, if anything, with the way to cover it. */
const alertCoverage = (
  rules: MongoAlertRulesInput | null,
  admin: boolean
): HubAlertCoverage | undefined => {
  const missing = rules
    ? KEY_TEMPLATES.filter(([template]) => !rules.templates.includes(template))
    : [];

  if (!rules || !missing.length) {
    return undefined;
  }
  const words = missing.map(([, word]) => word);
  const door = admin
    ? { door: { label: Messages.coverage.create, to: ALERT_TEMPLATES_PATH } }
    : {};
  if (noRules(rules)) {
    return {
      text: Messages.coverage.none,
      ...door,
      detail: admin
        ? Messages.coverage.templates(words)
        : Messages.coverage.askAdmin(words),
    };
  }

  return {
    text: Messages.coverage.missing(words),
    ...door,
    ...(admin ? {} : { detail: Messages.coverage.askAdminOne }),
  };
};

const R = Messages.recommendations;

/** Live recommendations: the clusters no backup protects, and what the alert rules leave uncovered. */
const liveRecommendations = (
  rules: MongoAlertRulesInput | null,
  unprotected: string[],
  clusters: number
): HubRecommendation[] => {
  const items: HubRecommendation[] = [];

  if (unprotected.length) {
    items.push({
      kind: 'notset',
      title: R.noBackups(unprotected.length),
      impact: R.noBackupsImpact,
      location: R.clusters(unprotected),
      clusterCount: unprotected.length,
      source: Messages.sources.backups,
      destination: MongoDestinations.setUpBackups(),
    });
  }
  if (rules && clusters) {
    const missing = KEY_TEMPLATES.filter(
      ([template]) => !rules.templates.includes(template)
    ).map(([, word]) => word);
    const base = {
      location: R.allClusters,
      clusterCount: clusters,
      source: Messages.sources.alerts,
      destination: MongoDestinations.alertTemplates(),
    };

    if (noRules(rules)) {
      items.push({
        kind: 'notset',
        title: R.noRules,
        impact: R.noRulesImpact,
        ...base,
      });
    } else if (missing.length) {
      items.push({
        kind: 'info',
        title: R.ruleGaps(missing),
        impact: R.ruleGapsImpact,
        ...base,
      });
    }
  }

  return items;
};

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
      (!alert.serviceName && alert.cluster && item.cluster === alert.cluster) ||
      (!alert.serviceName &&
        !alert.cluster &&
        alert.nodeName &&
        item.nodeName === alert.nodeName)
  );

  return service ? (service.environment ?? '') : null;
};

const statusState = ({
  empty,
  metricsOk,
  shown,
  health,
}: {
  empty: boolean;
  metricsOk: boolean;
  shown: number;
  health: HubHealth[];
}): HubStatus['state'] => {
  if (empty) {
    return 'empty';
  }
  if (!metricsOk || !shown) {
    return 'noData';
  }

  return worstHealth(health);
};

/** The MongoDB snapshot as the hub's view: pure, so every rule is testable without React. */
export const deriveMongoHub = (
  snapshot: MongoSnapshot,
  ctx: HubDeriveContext
): HubViewModel => {
  const filter = ctx.environment;
  const { capabilities, sources } = ctx;
  const allServices = snapshot.services ?? [];
  const services = allServices.filter((service) =>
    matchesEnvironment(service.environment, filter)
  );
  const metricsOk = snapshot.metrics !== null;
  const operations = ctx.operations ? snapshot.operations : null;
  const upgrades = new Map(
    (operations?.upgrades ?? []).map((upgrade) => [upgrade.clusterKey, upgrade])
  );
  const models = groupClusters(services, snapshot.metrics, ctx.now).map(
    (cluster) => {
      const upgrade = upgrades.get(cluster.key);

      return upgrade ? { ...cluster, upgrade } : cluster;
    }
  );
  const health = metricsOk ? models.map(clusterHealthOf) : [];

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
  // Nothing monitored at all; a filter that matches nothing is not an empty estate.
  const empty =
    snapshot.services !== null &&
    !allServices.length &&
    sources.inventory?.status !== 'loading';
  // Without the inventory there are no clusters to count or map backups to, which is not zero.
  const inventoryOk = snapshot.services !== null;
  const status: HubStatus = {
    state: statusState({ empty, metricsOk, shown: services.length, health }),
    clusters: inventoryOk ? models.length : null,
    ...(metricsOk && inventoryOk ? { health: countHealth(health) } : {}),
    lastGoodBackup: inventoryOk ? backups.lastGoodBackup : 'unknown',
    backupsFailed: backups.failedRecently,
    ...(backups.failedOldestGood
      ? { backupsFailedOldestGood: backups.failedOldestGood }
      : {}),
    ...(inventoryOk && backups.coverage
      ? { backupCoverage: backups.coverage }
      : {}),
    // Alerting off says nothing, and a source still loading would read as none firing.
    ...(sources.alerts?.status === 'error'
      ? { alerts: null }
      : sources.alerts?.status === 'ready' && snapshot.alerts
        ? {
            alerts: alerts.length,
            alertsCritical: alerts.filter(
              (alert) => alert.severity === 'critical'
            ).length,
          }
        : {}),
    ...(snapshot.alertRules
      ? { alertRules: noRules(snapshot.alertRules) ? 'none' : 'some' }
      : {}),
    checkedAt: new Date(snapshot.fetchedAt || ctx.now).toISOString(),
    doors: {
      alerts: { label: HubText.banner.openAlerts, to: ALERTS_PATH },
      backups: {
        label: HubText.banner.openBackups,
        to: MONGO_PATHS.backupInventory,
      },
      ...(ctx.role === 'admin'
        ? {
            setUpAlerts: {
              label: HubText.banner.setUpAlerts,
              to: ALERT_TEMPLATES_PATH,
            },
          }
        : {}),
      ...(capabilities.setUpBackup
        ? {
            setUpBackups: {
              label: HubText.banner.setUpBackups,
              to: MONGO_PATHS.backupNew,
            },
          }
        : {}),
    },
  };

  // Nothing monitored has nothing to cover.
  const coverage = allServices.length
    ? alertCoverage(snapshot.alertRules, ctx.role === 'admin')
    : undefined;

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
    recommendations: sectionState(sources, ['inventory']),
    backups: sectionState(sources, ['inventory', 'backups']),
    queries: sectionState(sources, ['qan', 'realtime']),
    dashboards: sectionState(sources, ['dashboards']),
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
    shortcuts: mongoShortcuts(filter, capabilities.operations),
    status,
    recommendations: sortRecommendations(
      snapshot.recommendations
        ? recommendationsInEnvironment(snapshot.recommendations, filter)
        : liveRecommendations(
            snapshot.alertRules,
            backups.unprotected ?? [],
            models.length
          )
    ),
    attention: sortNeedsAttention(
      snapshot.attention
        ? attentionInEnvironment(snapshot.attention, filter)
        : toNeedsAttention(entries, ctx.now)
    ),
    ...(coverage ? { alertCoverage: coverage } : {}),
    backups: backups.backups,
    queries: queries.queries,
    dashboards: deriveDashboards({
      clusters: models,
      dashboards: snapshot.dashboards,
      hasBackups,
      filter,
    }),
    actions: deriveActions(capabilities, ctx.role),
    sectionDoors: {
      apps: { label: Messages.doors.allApps, to: MONGO_PATHS.allApps },
    },
    sections,
  };
};
