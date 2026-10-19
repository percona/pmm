import type {
  HubBackupRow,
  HubBackups,
  HubCapabilities,
  HubDoor,
  HubRole,
  HubStatus,
} from 'hubs/template/hub.types';
import { backupStatus, listDashboard, MONGO_PATHS } from './mongodb.doors';
import { Messages } from './mongodb.messages';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import { type ClusterModel, clusterMembers } from './mongodb.topology';
import type { MongoBackupsInput, MongoMetrics } from './mongodb.types';

type Artifact = MongoBackupsInput['artifacts'][number];
type Schedule = MongoBackupsInput['scheduled'][number];

/** A backup problem the Needs attention list reports. */
export interface BackupFinding {
  kind: 'backupFailed' | 'pitrStopped';
  clusterKey: string;
  cluster: string;
  environment?: string;
  /** Schedule name, else "backup". */
  name: string;
  since: string;
}

export interface BackupsDerivation {
  backups: HubBackups;
  findings: BackupFinding[];
  lastGoodBackup: HubStatus['lastGoodBackup'];
  /** Enabled schedules across the shown clusters. */
  schedules: number;
  /** Clusters whose newest backup failed in the last 24 hours. */
  failedRecently: number;
}

export interface BackupsInput {
  clusters: ClusterModel[];
  /** Null when the source failed. */
  input: MongoBackupsInput | null;
  metrics: MongoMetrics | null;
  capabilities: Pick<HubCapabilities, 'setUpBackup'>;
  role: HubRole;
  filter: string | null;
  now: number;
}

const FINE_PITR = new Set(['success', 'inProgress', 'pending']);
const HOUR_MS = 3600 * 1000;

const time = (iso?: string) => (iso ? Date.parse(iso) : NaN);

const newest = <T extends { createdAt: string }>(items: T[]) =>
  [...items].sort((a, b) => time(b.createdAt) - time(a.createdAt))[0];

const scheduleName = (artifact: Artifact, schedules: Schedule[]) =>
  (
    schedules.find((schedule) => artifact.name.startsWith(schedule.name)) ??
    schedules.find(
      (schedule) =>
        schedule.serviceId === artifact.serviceId &&
        schedule.mode === artifact.mode
    )
  )?.name;

const doors = ({
  capabilities,
  role,
  filter,
  featureOff,
}: Pick<BackupsInput, 'capabilities' | 'role' | 'filter'> & {
  featureOff: boolean;
}): HubBackups['doors'] => ({
  app: {
    label: Messages.doors.mongodbBackups,
    to: MONGO_PATHS.backupInventory,
  },
  dashboard: {
    label: Messages.doors.backupDashboard,
    to: listDashboard('backupStatus', filter),
  },
  ...(capabilities.setUpBackup && !featureOff
    ? {
        setUp: { label: Messages.doors.setUpBackup, to: MONGO_PATHS.backupNew },
      }
    : {}),
  ...(featureOff && role === 'admin'
    ? {
        settings: {
          label: Messages.doors.turnOnBackups,
          to: MONGO_PATHS.advancedSettings,
        },
      }
    : {}),
});

const rowDoors = (cluster: ClusterModel): HubBackupRow['doors'] => {
  const dashboard: HubDoor = {
    label: cluster.name,
    to: backupStatus({
      environment: cluster.environment,
      cluster: cluster.name,
    }),
  };

  return {
    cluster: dashboard,
    lastGood: {
      label: Messages.doors.lastGoodBackup,
      to: MONGO_PATHS.backupInventory,
      context: Messages.doors.forSubject(cluster.name),
    },
    schedule: {
      label: Messages.doors.scheduledBackups,
      to: MONGO_PATHS.backupScheduled,
      context: Messages.doors.forSubject(cluster.name),
    },
  };
};

const isProblem = (row: HubBackupRow) =>
  !!row.failedAt || row.pitr === 'stopped' || row.lastGood?.level === 'warning';

const rowOrder = (row: HubBackupRow) =>
  isProblem(row) ? 0 : row.lastGood ? 1 : 2;

/** One row per cluster: last good backup, newest failure after it, point-in-time recovery and next run. */
export const deriveBackups = ({
  clusters,
  input,
  metrics,
  capabilities,
  role,
  filter,
  now,
}: BackupsInput): BackupsDerivation => {
  const empty = (
    state: HubBackups['state'],
    lastGoodBackup: HubStatus['lastGoodBackup']
  ): BackupsDerivation => ({
    backups: {
      state,
      rows: [],
      doors: doors({
        capabilities,
        role,
        filter,
        featureOff: state === 'featureOff',
      }),
    },
    findings: [],
    lastGoodBackup,
    schedules: 0,
    failedRecently: 0,
  });

  if (!input) {
    return empty('unknown', 'unknown');
  }
  if (!input.enabled) {
    return empty('featureOff', 'featureOff');
  }
  if (!clusters.length) {
    return empty('noServices', 'none');
  }

  const clusterOf = new Map<string, ClusterModel>();

  clusters.forEach((cluster) =>
    clusterMembers(cluster).forEach((member) =>
      clusterOf.set(member.service.serviceId, cluster)
    )
  );
  const artifacts = input.artifacts.filter((a) => clusterOf.has(a.serviceId));
  const scheduled = input.scheduled.filter((s) => clusterOf.has(s.serviceId));

  if (!artifacts.length && !scheduled.length) {
    const notSetUp = empty('notSetUp', 'notSetUp');

    return {
      ...notSetUp,
      backups: { ...notSetUp.backups, ...pbmHint(clusters, metrics) },
    };
  }

  const findings: BackupFinding[] = [];
  let newestGood: { at: string; cluster: string } | undefined;
  let failedRecently = 0;

  const rows = clusters.map((cluster): HubBackupRow => {
    const own = artifacts.filter((a) => clusterOf.get(a.serviceId) === cluster);
    const schedules = scheduled.filter(
      (s) => clusterOf.get(s.serviceId) === cluster
    );
    const copies = own.filter((a) => a.mode !== 'pitr');
    const lastSuccess = newest(copies.filter((a) => a.status === 'success'));
    const failures = copies.filter(
      (a) =>
        a.status === 'error' &&
        (!lastSuccess || time(a.createdAt) > time(lastSuccess.createdAt))
    );
    const failure = newest(failures);
    const latest = newest(copies);

    if (
      lastSuccess &&
      (!newestGood || time(lastSuccess.createdAt) > time(newestGood.at))
    ) {
      newestGood = { at: lastSuccess.createdAt, cluster: cluster.name };
    }

    if (
      latest?.status === 'error' &&
      latest === failure &&
      now - time(latest.createdAt) <= T.backupFailedLookbackHours * HOUR_MS
    ) {
      failedRecently += 1;
      findings.push({
        kind: 'backupFailed',
        clusterKey: cluster.key,
        cluster: cluster.name,
        environment: cluster.environment,
        name:
          scheduleName(latest, schedules) ??
          Messages.attention.backupFallbackName,
        since: latest.createdAt,
      });
    }

    let pitr: HubBackupRow['pitr'] = 'off';

    if (cluster.kind === 'standalone') {
      pitr = 'notApplicable';
    } else if (schedules.some((s) => s.mode === 'pitr' && s.enabled)) {
      const latestPitr = newest(own.filter((a) => a.mode === 'pitr'));

      pitr = !latestPitr
        ? 'notStarted'
        : FINE_PITR.has(latestPitr.status)
          ? 'on'
          : 'stopped';

      if (pitr === 'stopped' && latestPitr) {
        findings.push({
          kind: 'pitrStopped',
          clusterKey: cluster.key,
          cluster: cluster.name,
          environment: cluster.environment,
          name:
            schedules.find((s) => s.mode === 'pitr' && s.enabled)?.name ??
            Messages.attention.backupFallbackName,
          since: latestPitr.createdAt,
        });
      }
    }

    const nextRun = schedules
      .filter((s) => s.enabled && s.nextRun)
      .map((s) => s.nextRun as string)
      .sort((a, b) => time(a) - time(b))[0];

    return {
      id: `backup:${cluster.key}`,
      cluster: cluster.name,
      ...(cluster.environment ? { environment: cluster.environment } : {}),
      lastGood: lastSuccess
        ? {
            at: lastSuccess.createdAt,
            level:
              now - time(lastSuccess.createdAt) > T.backupStaleSeconds * 1000
                ? 'warning'
                : 'ok',
          }
        : null,
      ...(failure ? { failedAt: failure.createdAt } : {}),
      pitr,
      ...(nextRun ? { nextRun } : {}),
      doors: rowDoors(cluster),
    };
  });

  rows.sort(
    (a, b) => rowOrder(a) - rowOrder(b) || a.cluster.localeCompare(b.cluster)
  );

  return {
    backups: {
      state: 'ok',
      rows,
      doors: doors({ capabilities, role, filter, featureOff: false }),
      ...pbmHint(clusters, metrics),
    },
    findings,
    lastGoodBackup: newestGood ?? 'none',
    schedules: scheduled.filter((s) => s.enabled).length,
    failedRecently,
  };
};

/** Clusters that can run PBM (not standalones) and report it unconfigured. */
const pbmHint = (
  clusters: ClusterModel[],
  metrics: MongoMetrics | null
): Pick<HubBackups, 'hint'> => {
  const missing = clusters
    .filter(
      (cluster) =>
        cluster.kind !== 'standalone' &&
        metrics?.pbmConfiguredByCluster[cluster.name] === false
    )
    .map((cluster) => cluster.name)
    .sort((a, b) => a.localeCompare(b));

  return missing.length ? { hint: Messages.backups.pbmHint(missing) } : {};
};
