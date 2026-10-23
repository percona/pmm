// The module, not the package index, which would load the whole Operations UI.
import { formatCompactDuration } from '@pmm/plugins-om/src/format';
import { higherSeverity } from 'hubs/template/hub.attention';
import type { HubAttentionItem, HubSeverity } from 'hubs/template/hub.types';
import type { BackupFinding } from './mongodb.backups';
import {
  alertDoor,
  automationsDoor,
  incident,
  instanceSummary,
  MONGO_PATHS,
  oplogDetails,
  queryQan,
  replsetSummary,
  routerSummary,
  runningDoor,
} from './mongodb.doors';
import { Messages } from './mongodb.messages';
import {
  formatCalls,
  formatRatio,
  formatSeconds,
  type RunningQuery,
  runningAge,
  type SlowerQuery,
} from './mongodb.queries';
import { isLost, isStuck, versionDrift } from './mongodb.health';
import {
  type ClusterModel,
  clusterMembers,
  memberLagLevel,
  type MemberModel,
  placeOf,
  primaryOf,
  primaryOplogLevel,
  type SetModel,
  setHasNoPrimary,
  setHealthOf,
} from './mongodb.topology';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import type { MongoAlertInput, MongoOperationsInput } from './mongodb.types';

export const MONGODB_ATTENTION_ORDER = [
  'noPrimary',
  'memberDown',
  'memberLost',
  'memberNoData',
  'backupFailed',
  'pitrStopped',
  'oplogShort',
  'memberNotServing',
  'versionDrift',
  'behind',
  'alert',
  'querySlower',
  'queryRunning',
  'operationsFailed',
];

/** What an item is about, so a firing alert can find the item it repeats. */
interface Subject {
  clusterKey?: string;
  cluster?: string;
  set?: string;
  serviceNames: string[];
}

export interface AttentionEntry {
  item: HubAttentionItem;
  subject: Subject;
}

export interface AttentionInput {
  clusters: ClusterModel[];
  alerts: MongoAlertInput[];
  backupFindings: BackupFinding[];
  slower: SlowerQuery[];
  running: RunningQuery[];
  /** Null when Operations is off or its data failed. */
  operations: MongoOperationsInput | null;
  filter: string | null;
  now: number;
}

const HOUR_MS = 3600 * 1000;

const duration = (seconds: number) => formatCompactDuration(seconds) || '0s';

const isoFromSeconds = (seconds?: number) =>
  seconds === undefined ? undefined : new Date(seconds * 1000).toISOString();

/** "billing · prod" for a replica set, "orders › orders-shard1 · prod" inside a sharded cluster. */
const whereOf = (cluster: ClusterModel, inner?: string) =>
  Messages.where([
    cluster.kind === 'sharded' && inner
      ? Messages.path([cluster.name, inner])
      : cluster.name,
    cluster.environment,
  ]);

const since = (member: MemberModel) => isoFromSeconds(member.lastUpAt);

const setsOf = (
  cluster: ClusterModel
): Array<{ set: SetModel; inner?: string }> => [
  ...(cluster.config
    ? [{ set: cluster.config, inner: cluster.config.name }]
    : []),
  ...cluster.sets.map((set) => ({
    set,
    inner: cluster.kind === 'sharded' ? set.name : undefined,
  })),
];

const severityOfHealth = (down: boolean): HubSeverity =>
  down ? 'error' : 'warning';

const statePhrase = (state: string) =>
  Messages.memberStatePhrases[state] ?? Messages.memberStatePhrase(state);

const memberEntries = (cluster: ClusterModel): AttentionEntry[] => {
  const place = placeOf(cluster);
  const entries: AttentionEntry[] = [];
  const pushMember = (
    member: MemberModel,
    severity: HubSeverity,
    inner: string | undefined,
    to: string
  ) => {
    const { serviceId, serviceName } = member.service;
    const subject: Subject = {
      clusterKey: cluster.key,
      cluster: cluster.name,
      set: inner,
      serviceNames: [serviceName],
    };
    const start = since(member);

    if (member.verdict === 'down') {
      entries.push({
        item: {
          id: `memberDown:${serviceId}`,
          kind: 'memberDown',
          severity,
          title: Messages.attention.memberDown(serviceName),
          where: whereOf(cluster, inner),
          ...(start ? { since: start } : {}),
          door: {
            label: member.router
              ? Messages.doors.routerSummary
              : Messages.doors.instanceSummary,
            to: incident(to, start),
            context: Messages.doors.forSubject(serviceName),
          },
        },
        subject,
      });
    } else if (member.verdict === 'noData') {
      entries.push({
        item: {
          id: `memberNoData:${serviceId}`,
          kind: 'memberNoData',
          severity: 'warning',
          title: Messages.attention.memberNoData(serviceName),
          where: whereOf(cluster, inner),
          ...(start ? { since: start } : {}),
          door: {
            label: member.router
              ? Messages.doors.routerSummary
              : Messages.doors.instanceSummary,
            to: incident(to, start),
            context: Messages.doors.forSubject(serviceName),
          },
        },
        subject,
      });
    }
  };

  if (cluster.standalone) {
    pushMember(
      cluster.standalone,
      'error',
      undefined,
      instanceSummary(place, cluster.standalone.service.serviceName)
    );

    return entries;
  }

  const routersDown =
    cluster.routers.length > 0 &&
    cluster.routers.every((member) => member.verdict === 'down');

  cluster.routers.forEach((member) =>
    pushMember(
      member,
      severityOfHealth(routersDown),
      Messages.routersName,
      routerSummary(place, member.service.serviceName)
    )
  );

  setsOf(cluster).forEach(({ set, inner }) => {
    const noPrimary = setHasNoPrimary(set);
    const setDown = setHealthOf(set) === 'down';
    const down = set.members.filter((member) => member.verdict === 'down');

    if (noPrimary) {
      const start = down
        .map(since)
        .filter((value): value is string => !!value)
        .sort()[0];

      entries.push({
        item: {
          id: `noPrimary:${cluster.key}/${set.name}`,
          kind: 'noPrimary',
          severity: 'error',
          title: Messages.attention.noPrimary(set.name),
          where: whereOf(cluster, inner),
          ...(down.length
            ? {
                detail: Messages.attention.membersDown(
                  down.length,
                  set.members.length
                ),
              }
            : {}),
          ...(start ? { since: start } : {}),
          door: {
            label: Messages.doors.replsetSummary,
            to: incident(replsetSummary(place, set.name), start),
            context: Messages.doors.forSubject(set.name),
          },
        },
        subject: {
          clusterKey: cluster.key,
          cluster: cluster.name,
          set: set.name,
          serviceNames: set.members.map((member) => member.service.serviceName),
        },
      });
    }

    set.members.forEach((member) => {
      // A set without a primary already says its members are down.
      if (noPrimary && member.verdict === 'down') {
        return;
      }
      if ((isLost(member) || isStuck(member)) && member.state) {
        const { serviceId, serviceName } = member.service;
        const lost = isLost(member);
        const kind = lost ? 'memberLost' : 'memberNotServing';

        entries.push({
          item: {
            id: `${kind}:${serviceId}`,
            kind,
            severity: severityOfHealth(lost || setDown),
            title: lost
              ? Messages.attention.memberLost(
                  serviceName,
                  statePhrase(member.state)
                )
              : Messages.attention.notServing(
                  serviceName,
                  statePhrase(member.state),
                  member.stateSeconds !== undefined
                    ? duration(member.stateSeconds)
                    : undefined
                ),
            where: whereOf(cluster, inner),
            door: {
              label: Messages.doors.replsetSummary,
              to: replsetSummary(place, set.name),
              context: Messages.doors.forSubject(set.name),
            },
          },
          subject: {
            clusterKey: cluster.key,
            cluster: cluster.name,
            set: set.name,
            serviceNames: [serviceName],
          },
        });

        return;
      }
      pushMember(
        member,
        severityOfHealth(setDown),
        inner,
        instanceSummary(place, member.service.serviceName)
      );
    });
  });

  return entries;
};

const replicationEntries = (cluster: ClusterModel): AttentionEntry[] => {
  const place = placeOf(cluster);

  return setsOf(cluster).flatMap(({ set, inner }): AttentionEntry[] => {
    const entries: AttentionEntry[] = [];
    const primary = primaryOf(set);
    const oplog = primaryOplogLevel(set);

    if (primary?.oplogWindow !== undefined && oplog && oplog !== 'ok') {
      entries.push({
        item: {
          id: `oplogShort:${cluster.key}/${set.name}`,
          kind: 'oplogShort',
          severity: oplog,
          title: Messages.attention.oplogShort(
            set.name,
            duration(primary.oplogWindow)
          ),
          where: whereOf(cluster, inner),
          detail: Messages.attention.oplogFull(
            Math.round((primary.oplogFill ?? 0) * 100)
          ),
          door: {
            label: Messages.doors.oplogDetails,
            to: oplogDetails(place, set.name, primary.service.serviceName),
            context: Messages.doors.forSubject(set.name),
          },
        },
        subject: {
          clusterKey: cluster.key,
          cluster: cluster.name,
          set: set.name,
          serviceNames: [primary.service.serviceName],
        },
      });
    }

    set.members.forEach((member) => {
      const level = memberLagLevel(member);

      if (level === 'ok' || !member.lag) {
        return;
      }
      const { serviceId, serviceName } = member.service;

      entries.push({
        item: {
          id: `behind:${serviceId}`,
          kind: 'behind',
          severity: level,
          title: Messages.attention.behind(
            serviceName,
            duration(member.lag.now)
          ),
          where: whereOf(cluster, inner),
          detail: Messages.attention.lagDetail(duration(member.lag.sustained)),
          door: {
            label: Messages.doors.replsetSummary,
            to: replsetSummary(place, set.name),
            context: Messages.doors.forSubject(set.name),
          },
        },
        subject: {
          clusterKey: cluster.key,
          cluster: cluster.name,
          set: set.name,
          serviceNames: [serviceName],
        },
      });
    });

    return entries;
  });
};

/** A set whose members run different versions: an upgrade that stopped partway. */
const driftEntries = (cluster: ClusterModel): AttentionEntry[] => {
  const place = placeOf(cluster);

  return setsOf(cluster).flatMap(({ set, inner }): AttentionEntry[] => {
    const drift = versionDrift(
      set.members.map((member) => ({
        name: member.service.serviceName,
        verdict: member.verdict,
        version: member.version,
      }))
    );

    if (!drift) {
      return [];
    }

    return [
      {
        item: {
          id: `versionDrift:${cluster.key}/${set.name}`,
          kind: 'versionDrift',
          severity: 'warning',
          title: Messages.attention.versionDrift(set.name),
          where: whereOf(cluster, inner),
          detail: Messages.where(
            drift.map(({ version, members }) =>
              Messages.attention.versionOn(version, members)
            )
          ),
          door: {
            label: Messages.doors.replsetSummary,
            to: replsetSummary(place, set.name),
            context: Messages.doors.forSubject(set.name),
          },
        },
        subject: {
          clusterKey: cluster.key,
          cluster: cluster.name,
          set: set.name,
          serviceNames: set.members.map((member) => member.service.serviceName),
        },
      },
    ];
  });
};

const backupEntry = (
  finding: BackupFinding,
  clusters: ClusterModel[]
): AttentionEntry => {
  const cluster = clusters.find((item) => item.key === finding.clusterKey);
  const failed = finding.kind === 'backupFailed';

  return {
    item: {
      id: `${finding.kind}:${finding.clusterKey}`,
      kind: finding.kind,
      severity: 'warning',
      title: failed
        ? Messages.attention.backupFailed(finding.cluster, finding.name)
        : Messages.attention.pitrStopped(finding.cluster),
      where: Messages.where([finding.cluster, finding.environment]),
      since: finding.since,
      door: failed
        ? { label: Messages.doors.backups, to: MONGO_PATHS.backupInventory }
        : {
            label: Messages.doors.scheduledBackups,
            to: MONGO_PATHS.backupScheduled,
          },
    },
    subject: {
      clusterKey: finding.clusterKey,
      cluster: finding.cluster,
      serviceNames: cluster
        ? clusterMembers(cluster).map((member) => member.service.serviceName)
        : [],
    },
  };
};

/** critical, alert, emergency and error are errors; warning is a warning; anything lower is info. */
export const alertSeverity = (severity?: string): HubSeverity => {
  switch (severity?.toLowerCase()) {
    case 'critical':
    case 'alert':
    case 'emergency':
    case 'error':
      return 'error';
    case 'warning':
      return 'warning';
    default:
      return 'info';
  }
};

const MERGES: Record<string, string[]> = {
  pmm_mongodb_down: ['memberDown', 'memberNoData', 'noPrimary'],
  pmm_mongodb_replset_no_primary: ['noPrimary'],
  pmm_mongodb_replication_lag: ['behind'],
  pmm_mongodb_oplog_window: ['oplogShort'],
  pmm_mongodb_pbm_backup_failed: ['backupFailed'],
  pmm_mongodb_pbm_backup_stale: ['backupFailed'],
  pmm_mongodb_unusual_state: ['memberNotServing', 'memberLost'],
};

const sameCluster = (alert: MongoAlertInput, subject: Subject) =>
  !alert.cluster || alert.cluster === subject.cluster;

const matches = (alert: MongoAlertInput, entry: AttentionEntry): boolean => {
  const { subject, item } = entry;

  if (!sameCluster(alert, subject)) {
    return false;
  }
  switch (item.kind) {
    case 'memberDown':
    case 'memberNoData':
    case 'noPrimary':
      return alert.templateName === 'pmm_mongodb_replset_no_primary'
        ? alert.replicaSet === subject.set
        : !!alert.serviceName &&
            subject.serviceNames.includes(alert.serviceName);
    case 'behind':
    case 'memberNotServing':
    case 'memberLost':
      return !!alert.replicaSet && alert.replicaSet === subject.set;
    case 'oplogShort':
      return (
        (!!alert.replicaSet && alert.replicaSet === subject.set) ||
        (!!alert.serviceName &&
          subject.serviceNames.includes(alert.serviceName))
      );
    case 'backupFailed':
      return !!alert.cluster;
    default:
      return false;
  }
};

// Merged member alerts prefer the member's own row over the set row that folded it.
const KIND_PREFERENCE = ['memberDown', 'memberNoData', 'noPrimary'];

/** Where each host sits, by node name, and the members on it, for alerts about a host. */
export interface HostPlace {
  where: string;
  serviceNames: string[];
}

/** Folds firing alerts into the rows they repeat; the rest become rows of their own. */
export const mergeAlerts = (
  entries: AttentionEntry[],
  alerts: MongoAlertInput[],
  memberPlaces: Map<string, string>,
  hosts: Map<string, HostPlace> = new Map()
): AttentionEntry[] => {
  const merged = entries.map((entry) => ({
    ...entry,
    item: { ...entry.item },
  }));
  const own: AttentionEntry[] = [];

  alerts.forEach((alert) => {
    const kinds = alert.templateName ? MERGES[alert.templateName] : undefined;
    const target = kinds
      ? merged
          .filter(
            (entry) => kinds.includes(entry.item.kind) && matches(alert, entry)
          )
          .sort(
            (a, b) =>
              KIND_PREFERENCE.indexOf(a.item.kind) -
              KIND_PREFERENCE.indexOf(b.item.kind)
          )[0]
      : undefined;
    const severity = alertSeverity(alert.severity);

    if (target) {
      target.item.secondaryDoor = alertDoor(alert.serviceName);
      target.item.source = 'alert';
      target.item.severity = higherSeverity(target.item.severity, severity);

      return;
    }

    const subjectName =
      alert.serviceName ??
      alert.replicaSet ??
      alert.cluster ??
      alert.nodeName ??
      '';
    // An alert about a host belongs where its members are.
    const host =
      !alert.serviceName && alert.nodeName
        ? hosts.get(alert.nodeName)
        : undefined;

    own.push({
      item: {
        id: `alert:${alert.templateName ?? alert.name}:${subjectName}`,
        kind: 'alert',
        severity,
        title: alert.summary || alert.name || Messages.attention.alertFallback,
        where:
          (alert.serviceName && memberPlaces.get(alert.serviceName)) ||
          host?.where ||
          Messages.where([subjectName, alert.environment]),
        since: alert.activeAt,
        door: {
          ...alertDoor(alert.serviceName),
          label: Messages.doors.openAlerts,
          ...(host && alert.nodeName
            ? { context: Messages.doors.forSubject(alert.nodeName) }
            : {}),
        },
        source: 'alert',
      },
      subject: {
        cluster: alert.cluster,
        set: alert.replicaSet,
        serviceNames: alert.serviceName
          ? [alert.serviceName]
          : (host?.serviceNames ?? []),
      },
    });
  });

  return [...merged, ...own];
};

const queryEntries = (
  slower: SlowerQuery[],
  running: RunningQuery[],
  filter: string | null,
  memberPlaces: Map<string, string>
): AttentionEntry[] => [
  ...slower.map(
    ({ query, usualSeconds, ratio }): AttentionEntry => ({
      item: {
        id: `querySlower:${query.queryId}`,
        kind: 'querySlower',
        severity: 'warning',
        title: Messages.attention.querySlower(
          query.fingerprint,
          formatRatio(ratio)
        ),
        where: Messages.doors.queryAnalyticsLink,
        detail: Messages.queries.slowerDetail(
          formatSeconds(query.avgQueryTime),
          formatSeconds(usualSeconds),
          formatCalls(query.numQueries)
        ),
        door: {
          label: Messages.doors.openQuery,
          to: queryQan(filter, query.queryId),
        },
      },
      subject: { serviceNames: [] },
    })
  ),
  ...running.map(
    ({ query }): AttentionEntry => ({
      item: {
        id: `queryRunning:${query.serviceId}:${query.queryText}`,
        kind: 'queryRunning',
        severity: 'warning',
        title: Messages.attention.queryRunning(
          query.operation || Messages.queries.unknownOperation,
          query.namespace || Messages.queries.unknownNamespace,
          runningAge(query)
        ),
        where:
          memberPlaces.get(query.serviceName) ??
          Messages.where([query.serviceName]),
        detail: query.serviceName,
        door: runningDoor(query.serviceId, query.serviceName),
      },
      subject: { serviceNames: [query.serviceName] },
    })
  ),
];

const operationsEntries = (
  operations: MongoOperationsInput | null,
  now: number
): AttentionEntry[] =>
  (operations?.failedRuns ?? [])
    .filter(
      (run) =>
        now - Date.parse(run.finishedAt) <= T.operationsLookbackHours * HOUR_MS
    )
    .map(
      (run): AttentionEntry => ({
        item: {
          id: `operationsFailed:${run.id}`,
          kind: 'operationsFailed',
          severity: 'warning',
          title:
            run.kind === 'install'
              ? Messages.attention.installFailed(run.target)
              : run.failingNodes
                ? Messages.attention.scanFailed(run.failingNodes)
                : Messages.attention.scanFailedNoCount,
          where: Messages.apps.operations,
          since: run.finishedAt,
          door: automationsDoor(run.kind, run.id, run.target),
          source: 'operations',
        },
        subject: { serviceNames: [] },
      })
    );

/** Each host's place and members, by node name, for rows that only know the host. */
export const hostPlaceIndex = (
  clusters: ClusterModel[]
): Map<string, HostPlace> => {
  const hosts = new Map<string, HostPlace>();
  const places = memberPlaceIndex(clusters);

  clusters.flatMap(clusterMembers).forEach(({ service }) => {
    const host = hosts.get(service.nodeName);

    hosts.set(service.nodeName, {
      where: host?.where ?? places.get(service.serviceName) ?? '',
      serviceNames: [...(host?.serviceNames ?? []), service.serviceName],
    });
  });

  return hosts;
};

/** Where each member sits, by service name, for rows that only know the service. */
export const memberPlaceIndex = (clusters: ClusterModel[]) => {
  const places = new Map<string, string>();

  clusters.forEach((cluster) => {
    cluster.routers.forEach((member) =>
      places.set(
        member.service.serviceName,
        whereOf(cluster, Messages.routersName)
      )
    );
    setsOf(cluster).forEach(({ set, inner }) =>
      set.members.forEach((member) =>
        places.set(member.service.serviceName, whereOf(cluster, inner))
      )
    );
    if (cluster.standalone) {
      places.set(cluster.standalone.service.serviceName, whereOf(cluster));
    }
  });

  return places;
};

/** Every attention row with what it is about, before ranking; one problem is one row. */
export const deriveAttention = ({
  clusters,
  alerts,
  backupFindings,
  slower,
  running,
  operations,
  filter,
  now,
}: AttentionInput): AttentionEntry[] => {
  const places = memberPlaceIndex(clusters);
  // An upgrade restarts members and mixes versions on purpose, so only alerts speak for a cluster while one runs.
  const steady = clusters.filter((cluster) => !cluster.upgrade);
  const derived = [
    ...steady.flatMap(memberEntries),
    ...backupFindings.map((finding) => backupEntry(finding, clusters)),
    ...steady.flatMap(replicationEntries),
    ...steady.flatMap(driftEntries),
  ];

  return [
    ...mergeAlerts(derived, alerts, places, hostPlaceIndex(clusters)),
    ...queryEntries(slower, running, filter, places),
    ...operationsEntries(operations, now),
  ];
};
