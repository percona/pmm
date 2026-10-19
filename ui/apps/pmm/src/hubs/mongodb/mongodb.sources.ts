import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { OmInventoryRun, OmTopologyResponse } from '@pmm/plugins-om';
import { useCallback, useMemo } from 'react';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';
import { usePrometheusAlertRules } from 'hooks/api/usePrometheusAlertRules';
import { useRealtimeQueries, useRealtimeSessions } from 'hooks/api/useRealtime';
import { getRecentKey, readRecent } from 'search/search.recent';
import type { PrometheusAlertRulesResponse } from 'types/alerting.types';
import { type QueryData, RealtimeSessionStatus } from 'types/rta.types';
import {
  type ManagedService,
  ServiceStatus,
  ServiceType,
} from 'types/services.types';
import type { BackupArtifact, ScheduledBackup } from 'hubs/api/backups.types';
import { listArtifacts, listScheduled } from 'hubs/api/backups';
import { readImpressions, searchDashboards } from 'hubs/api/grafanaDashboards';
import { listManagedServicesQuietly } from 'hubs/api/inventory';
import {
  getBootstrapRuns,
  getRuns,
  getTopology,
  type OmBootstrapRunSummary,
} from 'hubs/api/operations';
import { getReport } from 'hubs/api/qan';
import type { QanReportLabel, QanReportRow } from 'hubs/api/qan.types';
import { instantQuery } from 'hubs/api/victoriaMetrics';
import { matchesEnvironment } from 'hubs/template/hub.params';
import {
  HUB_NO_ENVIRONMENT,
  type HubLiveContext,
  type HubLiveResult,
  type HubSourceId,
  type HubSourceState,
} from 'hubs/template/hub.types';
import {
  MONGO_QUERY_IDS,
  mongoQueries,
  parseMongoMetrics,
} from './mongodb.promql';
import type {
  MongoAlertInput,
  MongoBackupMode,
  MongoBackupsInput,
  MongoBackupStatus,
  MongoMetrics,
  MongoOperationsInput,
  MongoServiceInput,
  MongoSnapshot,
  QanRowInput,
  RtaQueryInput,
} from './mongodb.types';

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;

export const MONGO_HUB_INTERVALS = {
  inventory: MINUTE,
  metrics: 30 * SECOND,
  alerts: 30 * SECOND,
  backups: MINUTE,
  qan: 5 * MINUTE,
  realtimeSessions: 30 * SECOND,
  realtimeQueries: 15 * SECOND,
  topology: 30 * SECOND,
  runs: MINUTE,
  dashboardsStale: 10 * MINUTE,
} as const;

export const MONGO_HUB_KEYS = {
  services: 'mongodbHub:services',
  metrics: 'mongodbHub:metrics',
  backups: 'mongodbHub:backups',
  qan: 'mongodbHub:qan',
  topology: 'mongodbHub:omTopology',
  runs: 'mongodbHub:omRuns',
  bootstrapRuns: 'mongodbHub:omBootstrapRuns',
  dashboards: 'mongodbHub:dashboards',
  recentDashboards: 'mongodbHub:recentDashboards',
} as const;

const OM_RUNS_LIMIT = 20;
const DASHBOARD_QUERY = 'MongoDB';

const EMPTY_METRICS: MongoMetrics = {
  up: {},
  lastUpAt: {},
  self: {},
  roles: {},
  lag: {},
  oplogWindow: {},
  oplogFill: {},
  shardsByCluster: {},
  pbmConfiguredByCluster: {},
};

const INVENTORY_STATUS: Partial<
  Record<ServiceStatus, MongoServiceInput['inventoryStatus']>
> = {
  [ServiceStatus.up]: 'up',
  [ServiceStatus.down]: 'down',
};

/** Management services as the snapshot keeps them; an empty label is no label. */
export const toServiceInputs = (
  services: ManagedService[]
): MongoServiceInput[] =>
  services.map((service) => ({
    serviceId: service.serviceId,
    serviceName: service.serviceName,
    nodeName: service.nodeName,
    ...(service.environment ? { environment: service.environment } : {}),
    ...(service.cluster ? { cluster: service.cluster } : {}),
    ...(service.replicationSet
      ? { replicationSet: service.replicationSet }
      : {}),
    inventoryStatus:
      (service.status && INVENTORY_STATUS[service.status]) ?? 'unknown',
  }));

/** Every hub query at once; one failure fails the source, so nothing reads as no data by mistake. */
export const fetchMongoMetrics = async (
  environment: string | null,
  signal?: AbortSignal
): Promise<MongoMetrics> => {
  const queries = mongoQueries(environment);
  const results = await Promise.all(
    MONGO_QUERY_IDS.map((id) => instantQuery(queries[id], { signal }))
  );

  return parseMongoMetrics(
    Object.fromEntries(
      MONGO_QUERY_IDS.map((id, index) => [id, results[index]])
    ) as Record<(typeof MONGO_QUERY_IDS)[number], (typeof results)[number]>
  );
};

const isFiring = (state?: string) =>
  !!state && (state === 'firing' || state.startsWith('Alerting'));

/** Firing alerts about MongoDB: a mongodb service type, or a MongoDB service or cluster name. */
export const toAlertInputs = (
  response: PrometheusAlertRulesResponse | undefined,
  services: MongoServiceInput[]
): MongoAlertInput[] => {
  const serviceNames = new Set(services.map((service) => service.serviceName));
  const clusters = new Set(
    services.flatMap((service) => (service.cluster ? [service.cluster] : []))
  );

  return (response?.data.groups ?? []).flatMap((group) =>
    (group.rules ?? []).flatMap((rule) =>
      (rule.alerts ?? []).flatMap((alert): MongoAlertInput[] => {
        const labels = alert.labels ?? {};
        const firing = alert.state
          ? isFiring(alert.state)
          : rule.state === 'firing';
        const serviceName = labels.service_name || labels.service;
        const mongo =
          labels.service_type === 'mongodb' ||
          (!!serviceName && serviceNames.has(serviceName)) ||
          (!!labels.cluster && clusters.has(labels.cluster));

        if (!firing || !mongo) {
          return [];
        }
        const replicaSet = labels.rs_nm || labels.set || labels.replication_set;

        return [
          {
            name: labels.alertname || rule.name,
            ...(labels.template_name
              ? { templateName: labels.template_name }
              : {}),
            ...(labels.severity ? { severity: labels.severity } : {}),
            ...(serviceName ? { serviceName } : {}),
            ...(labels.cluster ? { cluster: labels.cluster } : {}),
            ...(labels.environment ? { environment: labels.environment } : {}),
            ...(replicaSet ? { replicaSet } : {}),
            activeAt: alert.activeAt ?? '',
            ...(alert.annotations?.summary
              ? { summary: alert.annotations.summary }
              : {}),
          },
        ];
      })
    )
  );
};

const BACKUP_MODE: Record<string, MongoBackupMode | undefined> = {
  BACKUP_MODE_SNAPSHOT: 'snapshot',
  BACKUP_MODE_INCREMENTAL: 'incremental',
  BACKUP_MODE_PITR: 'pitr',
};

const BACKUP_STATUS: Record<string, MongoBackupStatus | undefined> = {
  BACKUP_STATUS_SUCCESS: 'success',
  BACKUP_STATUS_ERROR: 'error',
  BACKUP_STATUS_IN_PROGRESS: 'inProgress',
  BACKUP_STATUS_PENDING: 'pending',
};

const isMongo = (vendor?: string) => vendor?.toLowerCase() === 'mongodb';

export const toBackupsInput = (
  enabled: boolean,
  artifacts: BackupArtifact[],
  scheduled: ScheduledBackup[]
): MongoBackupsInput => ({
  enabled,
  artifacts: artifacts
    .filter((a) => isMongo(a.vendor))
    .map((a) => ({
      artifactId: a.artifactId,
      name: a.name,
      serviceId: a.serviceId,
      serviceName: a.serviceName,
      mode: BACKUP_MODE[a.mode] ?? 'snapshot',
      status: BACKUP_STATUS[a.status] ?? 'other',
      createdAt: a.createdAt,
    })),
  scheduled: scheduled
    .filter((s) => isMongo(s.vendor))
    .map((s) => ({
      scheduledBackupId: s.scheduledBackupId,
      name: s.name,
      serviceId: s.serviceId,
      serviceName: s.serviceName,
      mode: BACKUP_MODE[s.mode] ?? 'snapshot',
      enabled: s.enabled,
      cronExpression: s.cronExpression,
      ...(s.nextRun ? { nextRun: s.nextRun } : {}),
      ...(s.lastRun ? { lastRun: s.lastRun } : {}),
    })),
});

/** Report rows with their average query time in seconds; the total row has no query id. */
// QAN sends "NaN" as a string for an empty window.
const finite = (value: unknown) => {
  const number = Number(value);

  return value != null && Number.isFinite(number) ? number : undefined;
};

export const toQanRows = (rows: QanReportRow[]): QanRowInput[] =>
  rows.flatMap((row) => {
    const numQueries =
      finite(row.numQueries) ??
      finite(row.metrics?.numQueries?.stats?.sum) ??
      0;
    const stats = row.metrics?.queryTime?.stats;
    const sum = finite(stats?.sum);
    const avgQueryTime =
      finite(stats?.avg) ??
      (sum !== undefined && numQueries > 0 ? sum / numQueries : undefined);

    return row.dimension && avgQueryTime !== undefined
      ? [
          {
            queryId: row.dimension,
            fingerprint: row.fingerprint ?? row.dimension,
            numQueries,
            avgQueryTime,
          },
        ]
      : [];
  });

export const qanLabels = (environment: string | null): QanReportLabel[] => [
  { key: 'service_type', value: ['mongodb'] },
  ...(environment === null
    ? []
    : [
        {
          key: 'environment',
          value: [environment === HUB_NO_ENVIRONMENT ? '' : environment],
        },
      ]),
];

/** The last hour against the 24 hours before it. */
export const fetchQan = async (
  environment: string | null,
  now = Date.now()
) => {
  const labels = qanLabels(environment);
  const [current, baseline] = await Promise.all([
    getReport({ from: new Date(now - HOUR), to: new Date(now), labels }),
    getReport({
      from: new Date(now - 25 * HOUR),
      to: new Date(now - HOUR),
      labels,
    }),
  ]);

  return { current: toQanRows(current), baseline: toQanRows(baseline) };
};

export const toRtaQueries = (queries: QueryData[]): RtaQueryInput[] =>
  queries.map((query) => {
    const { databaseName, collection, operation } = query.mongoDbPayload ?? {};

    return {
      serviceId: query.serviceId,
      serviceName: query.serviceName,
      queryText: query.queryText,
      ...(operation ? { operation } : {}),
      ...(databaseName && collection
        ? { namespace: `${databaseName}.${collection}` }
        : {}),
      // useRealtimeQueries stores seconds in this field, whatever its name says.
      durationMs: (query.queryExecutionDurationMs ?? 0) * 1000,
    };
  });

const TERMINAL_RUN = new Set([
  'RUN_STATUS_SUCCESS',
  'RUN_STATUS_PARTIAL',
  'RUN_STATUS_FAILED',
]);
const FAILED_RUN = new Set(['RUN_STATUS_PARTIAL', 'RUN_STATUS_FAILED']);
const FAILED_INSTALL = new Set(['failed', 'rolled_back']);

/** Cluster ids, service counts, and the failures worth a row: the newest scan if it failed, and failed installs. */
export const toOperationsInput = (
  topology: OmTopologyResponse,
  runs: OmInventoryRun[],
  bootstrapRuns: OmBootstrapRunSummary[],
  environment: string | null
): MongoOperationsInput => {
  const clusterIds: Record<string, string> = {};
  let services = 0;
  let up = 0;

  topology.environments.forEach((env) => {
    const shown = matchesEnvironment(env.env_name, environment);

    env.clusters.forEach((cluster) => {
      clusterIds[`${env.env_name ?? ''}/${cluster.name ?? ''}`] = cluster.id;
      if (shown) {
        services += cluster.services.length;
        up += cluster.services.filter(
          (service) => service.status === 'SERVICE_STATUS_UP'
        ).length;
      }
    });
  });

  const lastScan = [...runs]
    .filter((run) => TERMINAL_RUN.has(run.status))
    .sort((a, b) => Date.parse(b.start_time) - Date.parse(a.start_time))[0];
  const lastScanAt = lastScan
    ? (lastScan.end_time ?? lastScan.start_time)
    : undefined;

  return {
    clusterIds,
    summary: { services, up, ...(lastScanAt ? { lastScanAt } : {}) },
    failedRuns: [
      ...(lastScan && FAILED_RUN.has(lastScan.status)
        ? [
            {
              id: lastScan.run_id,
              kind: 'scan' as const,
              target: '',
              finishedAt: lastScanAt ?? lastScan.start_time,
              ...(lastScan.failing_nodes?.length
                ? { failingNodes: lastScan.failing_nodes.length }
                : {}),
            },
          ]
        : []),
      ...bootstrapRuns
        .filter(
          (run) =>
            FAILED_INSTALL.has(run.status) &&
            matchesEnvironment(run.environment, environment)
        )
        .map((run) => ({
          id: run.run_id,
          kind: 'install' as const,
          target:
            run.hosts.map((host) => host.host).join(', ') ||
            run.replica_set_name,
          finishedAt: run.finished_at ?? run.started_at,
        })),
    ],
  };
};

/** Grafana's recently viewed dashboards, else the ones PMM search recorded. */
export const readRecentDashboards = (user: {
  orgId: number;
  id: number;
}): Array<{ uid: string; visitedAt?: string }> => {
  const impressions = readImpressions(user.orgId);

  if (impressions.length) {
    return impressions.map((uid) => ({ uid }));
  }

  return readRecent(getRecentKey(user))
    .filter((entry) => entry.kind === 'dashboard')
    .sort((a, b) => b.at - a.at)
    .map((entry) => ({
      uid: entry.id.replace(/^dashboard:/, ''),
      visitedAt: new Date(entry.at).toISOString(),
    }));
};

const sourceState = (
  enabled: boolean,
  query: Pick<
    UseQueryResult<unknown>,
    'isPending' | 'isError' | 'error' | 'dataUpdatedAt'
  >
): HubSourceState => {
  if (!enabled) {
    return { status: 'disabled' };
  }
  if (query.isError) {
    return {
      status: 'error',
      ...(query.error instanceof Error ? { message: query.error.message } : {}),
    };
  }
  if (query.isPending) {
    return { status: 'loading' };
  }

  return {
    status: 'ready',
    ...(query.dataUpdatedAt ? { updatedAt: query.dataUpdatedAt } : {}),
  };
};

const combine = (...states: HubSourceState[]): HubSourceState =>
  states.find((state) => state.status === 'error') ??
  states.find((state) => state.status === 'loading') ??
  states.find((state) => state.status === 'ready') ??
  states[0];

/** Every live source of the MongoDB hub, each its own query, so one failure blanks only its section. */
export const useMongoLiveSnapshot = ({
  enabled,
  environment,
  operations,
}: HubLiveContext): HubLiveResult<MongoSnapshot> => {
  const { settings } = useSettings();
  const { user } = useUser();
  const alertsEnabled = enabled && settings?.alertingEnabled === true;
  const backupsEnabled = enabled && settings?.backupManagementEnabled === true;
  const opsEnabled = enabled && operations;

  const inventory = useQuery({
    queryKey: [MONGO_HUB_KEYS.services],
    queryFn: () => listManagedServicesQuietly(ServiceType.mongodb),
    enabled,
    refetchInterval: MONGO_HUB_INTERVALS.inventory,
  });
  const metrics = useQuery({
    queryKey: [MONGO_HUB_KEYS.metrics, { environment }],
    queryFn: ({ signal }) => fetchMongoMetrics(environment, signal),
    enabled,
    refetchInterval: MONGO_HUB_INTERVALS.metrics,
    refetchIntervalInBackground: true,
  });
  const alerts = usePrometheusAlertRules({
    enabled: alertsEnabled,
    refetchInterval: MONGO_HUB_INTERVALS.alerts,
    refetchIntervalInBackground: true,
  });
  const backups = useQuery({
    queryKey: [MONGO_HUB_KEYS.backups],
    queryFn: async () => {
      const [artifacts, scheduled] = await Promise.all([
        listArtifacts(),
        listScheduled(),
      ]);

      return toBackupsInput(true, artifacts, scheduled);
    },
    enabled: backupsEnabled,
    refetchInterval: MONGO_HUB_INTERVALS.backups,
  });
  const qan = useQuery({
    queryKey: [MONGO_HUB_KEYS.qan, { environment }],
    queryFn: () => fetchQan(environment),
    enabled,
    refetchInterval: MONGO_HUB_INTERVALS.qan,
  });
  const sessions = useRealtimeSessions({
    enabled,
    refetchInterval: MONGO_HUB_INTERVALS.realtimeSessions,
  });
  const services = useMemo(
    () => toServiceInputs(inventory.data ?? []),
    [inventory.data]
  );
  const sessionServiceIds = useMemo(() => {
    const mongo = new Set(services.map((service) => service.serviceId));

    return (sessions.data ?? [])
      .filter(
        (session) =>
          session.status === RealtimeSessionStatus.running &&
          mongo.has(session.serviceId)
      )
      .map((session) => session.serviceId)
      .sort();
  }, [services, sessions.data]);
  const realtime = useRealtimeQueries(
    { serviceIds: sessionServiceIds },
    {
      enabled: enabled && sessionServiceIds.length > 0,
      refetchInterval: MONGO_HUB_INTERVALS.realtimeQueries,
    }
  );
  const topology = useQuery({
    queryKey: [MONGO_HUB_KEYS.topology],
    queryFn: getTopology,
    enabled: opsEnabled,
    refetchInterval: MONGO_HUB_INTERVALS.topology,
    refetchIntervalInBackground: true,
  });
  const runs = useQuery({
    queryKey: [MONGO_HUB_KEYS.runs, OM_RUNS_LIMIT],
    queryFn: () => getRuns(OM_RUNS_LIMIT),
    enabled: opsEnabled,
    refetchInterval: MONGO_HUB_INTERVALS.runs,
  });
  const bootstrapRuns = useQuery({
    queryKey: [MONGO_HUB_KEYS.bootstrapRuns, OM_RUNS_LIMIT],
    queryFn: () => getBootstrapRuns(OM_RUNS_LIMIT),
    enabled: opsEnabled,
    refetchInterval: MONGO_HUB_INTERVALS.runs,
  });
  const catalog = useQuery({
    queryKey: [MONGO_HUB_KEYS.dashboards, DASHBOARD_QUERY],
    queryFn: () => searchDashboards(DASHBOARD_QUERY),
    enabled,
    staleTime: MONGO_HUB_INTERVALS.dashboardsStale,
  });
  const recent = useQuery({
    queryKey: [MONGO_HUB_KEYS.recentDashboards, user?.orgId, user?.id],
    queryFn: () => (user ? readRecentDashboards(user) : []),
    enabled: enabled && !!user,
    staleTime: 0,
    refetchOnWindowFocus: 'always',
  });

  // Keyed by content, so the sources (and the snapshot) keep their identity between renders.
  const sourcesKey = JSON.stringify({
    inventory: sourceState(enabled, inventory),
    metrics: sourceState(enabled, metrics),
    alerts: sourceState(alertsEnabled, alerts),
    backups: sourceState(backupsEnabled, backups),
    qan: sourceState(enabled, qan),
    realtime: combine(
      sourceState(enabled, sessions),
      ...(sessionServiceIds.length ? [sourceState(enabled, realtime)] : [])
    ),
    operations: combine(
      sourceState(opsEnabled, topology),
      sourceState(opsEnabled, runs),
      sourceState(opsEnabled, bootstrapRuns)
    ),
    dashboards: sourceState(enabled, catalog),
  } satisfies Record<HubSourceId, HubSourceState>);
  const sources = useMemo(
    () => JSON.parse(sourcesKey) as Record<HubSourceId, HubSourceState>,
    [sourcesKey]
  );

  const snapshot = useMemo<MongoSnapshot>(() => {
    const updated = Object.values(sources).flatMap((source) =>
      source.updatedAt ? [source.updatedAt] : []
    );

    return {
      fetchedAt: updated.length ? Math.max(...updated) : Date.now(),
      services: inventory.isError ? null : services,
      metrics: metrics.isError ? null : (metrics.data ?? EMPTY_METRICS),
      alerts: alerts.isError ? null : toAlertInputs(alerts.data, services),
      backups: !backupsEnabled
        ? toBackupsInput(false, [], [])
        : backups.isError
          ? null
          : (backups.data ?? null),
      qan: qan.isError ? null : (qan.data ?? { current: [], baseline: [] }),
      realtime:
        sessions.isError || realtime.isError
          ? null
          : {
              sessionServiceIds,
              queries: toRtaQueries(
                sessionServiceIds.length ? (realtime.data ?? []) : []
              ),
            },
      operations:
        opsEnabled && topology.data && !runs.isError && !bootstrapRuns.isError
          ? toOperationsInput(
              topology.data,
              runs.data ?? [],
              bootstrapRuns.data ?? [],
              environment
            )
          : null,
      dashboards:
        catalog.isError || !catalog.data
          ? null
          : { catalog: catalog.data, recent: recent.data ?? [] },
    };
  }, [
    sources,
    inventory.isError,
    services,
    metrics.isError,
    metrics.data,
    alerts.isError,
    alerts.data,
    backups.isError,
    backups.data,
    backupsEnabled,
    qan.isError,
    qan.data,
    sessions.isError,
    realtime.isError,
    realtime.data,
    sessionServiceIds,
    opsEnabled,
    topology.data,
    runs.isError,
    runs.data,
    bootstrapRuns.isError,
    bootstrapRuns.data,
    environment,
    catalog.isError,
    catalog.data,
    recent.data,
  ]);

  const hasSessions = sessionServiceIds.length > 0;
  const { refetch: refetchInventory } = inventory;
  const { refetch: refetchMetrics } = metrics;
  const { refetch: refetchAlerts } = alerts;
  const { refetch: refetchBackups } = backups;
  const { refetch: refetchQan } = qan;
  const { refetch: refetchSessions } = sessions;
  const { refetch: refetchRealtime } = realtime;
  const { refetch: refetchTopology } = topology;
  const { refetch: refetchRuns } = runs;
  const { refetch: refetchBootstrapRuns } = bootstrapRuns;
  const { refetch: refetchCatalog } = catalog;
  const { refetch: refetchRecent } = recent;
  const refetch = useCallback(() => {
    if (!enabled) {
      return;
    }
    const pending = [
      refetchInventory,
      refetchMetrics,
      refetchQan,
      refetchSessions,
      refetchCatalog,
      refetchRecent,
      ...(alertsEnabled ? [refetchAlerts] : []),
      ...(backupsEnabled ? [refetchBackups] : []),
      ...(hasSessions ? [refetchRealtime] : []),
      ...(opsEnabled
        ? [refetchTopology, refetchRuns, refetchBootstrapRuns]
        : []),
    ];

    pending.forEach((run) => void run());
  }, [
    enabled,
    alertsEnabled,
    backupsEnabled,
    hasSessions,
    opsEnabled,
    refetchInventory,
    refetchMetrics,
    refetchAlerts,
    refetchBackups,
    refetchQan,
    refetchSessions,
    refetchRealtime,
    refetchTopology,
    refetchRuns,
    refetchBootstrapRuns,
    refetchCatalog,
    refetchRecent,
  ]);

  return { snapshot, sources, refetch };
};
