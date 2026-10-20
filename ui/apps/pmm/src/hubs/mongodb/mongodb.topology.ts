// The module, not the package index, which would load the whole Operations UI.
import { formatCompactDuration } from '@pmm/plugins-om/src/format';
import type {
  HubCapabilities,
  HubDoor,
  HubHealth,
  HubLevel,
  HubMeasure,
  HubTreeNode,
  HubUpdate,
  HubVersion,
} from 'hubs/template/hub.types';
import {
  backupStatus,
  clusterQan,
  clusterSummary,
  instanceSummary,
  lagDoor,
  manageDoor,
  type MongoPlace,
  oplogDoor,
  replsetSummary,
  routerSummary,
} from './mongodb.doors';
import {
  compareHealth,
  hasNoPrimary,
  type HealthVerdict,
  lagLevel,
  type MemberHealthInput,
  type MemberVerdict,
  memberHealth,
  memberVerdict,
  oplogLevel,
  routersHealth,
  setHealth,
  shardedHealth,
  standaloneHealth,
  upgradingHealth,
  versionDrift,
  worstLevel,
} from './mongodb.health';
import { Messages } from './mongodb.messages';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import type {
  MongoMetrics,
  MongoServiceInput,
  MongoUpdateCheck,
  MongoUpgradeInput,
} from './mongodb.types';
import {
  compareVersions,
  groupByVersion,
  lowestVersion,
  type VersionGroup,
} from './mongodb.versions';

export interface MemberModel {
  service: MongoServiceInput;
  verdict: MemberVerdict;
  /** member_state while up: PRIMARY, SECONDARY, ARBITER and so on. */
  state?: string;
  /** How long the member has been in its state, as far as the last hour tells. */
  stateSeconds?: number;
  /** The server version it reported last, such as 7.0.24-13. */
  version?: string;
  router: boolean;
  /** Epoch seconds the member was last seen up. */
  lastUpAt?: number;
  lag?: { now: number; sustained: number };
  oplogWindow?: number;
  oplogFill?: number;
}

export interface SetModel {
  /** Exporter's replica set name (rs_nm), else PMM's replication_set label. */
  name: string;
  config: boolean;
  members: MemberModel[];
}

export interface ClusterModel {
  /** `${environment ?? ''}/${name}` */
  key: string;
  environment?: string;
  /** PMM's cluster label, else the replica set or service name the services group by. */
  name: string;
  /** PMM's cluster label; absent when the services carry none, so doors leave var-cluster out. */
  label?: string;
  kind: 'sharded' | 'replicaSet' | 'standalone';
  routers: MemberModel[];
  config?: SetModel;
  /** Shards of a sharded cluster, or the one set of a replica set. */
  sets: SetModel[];
  standalone?: MemberModel;
  /** An upgrade Operations runs on the cluster now. */
  upgrade?: MongoUpgradeInput;
}

export const clusterKey = (environment: string | undefined, name: string) =>
  `${environment ?? ''}/${name}`;

const toMember = (
  service: MongoServiceInput,
  metrics: MongoMetrics | null,
  now: number
): MemberModel => {
  const verdict = memberVerdict(service, metrics);
  const id = service.serviceId;
  const self = metrics?.self[id];
  const stateSince = metrics?.stateSince[id];

  return {
    service,
    verdict,
    state: verdict === 'up' ? self?.state : undefined,
    ...(verdict === 'up' && self?.state && stateSince !== undefined
      ? { stateSeconds: Math.max(0, Math.round(now / 1000 - stateSince)) }
      : {}),
    ...(metrics?.versions[id] ? { version: metrics.versions[id] } : {}),
    router: metrics?.up[id]?.clusterRole === 'mongos',
    lastUpAt: metrics?.lastUpAt[id],
    lag: metrics?.lag[id],
    oplogWindow: metrics?.oplogWindow[id],
    oplogFill: metrics?.oplogFill[id],
  };
};

const setNameOf = (member: MemberModel, metrics: MongoMetrics | null) =>
  metrics?.self[member.service.serviceId]?.rsName ||
  metrics?.roles[member.service.serviceId]?.rsName ||
  member.service.replicationSet ||
  undefined;

// A member that is down reports no role, so the one it had in the last day places its set.
const isConfigServer = (member: MemberModel, metrics: MongoMetrics | null) =>
  (metrics?.self[member.service.serviceId]?.clRole ??
    metrics?.roles[member.service.serviceId]?.clRole) === 'configsvr';

/** Topology order: primary, secondaries, arbiters, then members without a reported state. */
const STATE_ORDER: Record<string, number> = {
  PRIMARY: 0,
  SECONDARY: 1,
  ARBITER: 2,
};

const byTopology = (a: MemberModel, b: MemberModel) =>
  (STATE_ORDER[a.state ?? ''] ?? 3) - (STATE_ORDER[b.state ?? ''] ?? 3) ||
  a.service.serviceName.localeCompare(b.service.serviceName);

const byName = (a: { name: string }, b: { name: string }) =>
  a.name.localeCompare(b.name);

/** Groups services by environment and cluster (else replication set); routers, config servers or two sets make it sharded. */
export const groupClusters = (
  services: MongoServiceInput[],
  metrics: MongoMetrics | null,
  now = Date.now()
): ClusterModel[] => {
  const groups = new Map<
    string,
    {
      environment?: string;
      name: string;
      label?: string;
      members: MemberModel[];
    }
  >();
  const standalones: ClusterModel[] = [];

  for (const service of services) {
    const name = service.cluster || service.replicationSet;
    const member = toMember(service, metrics, now);

    if (!name) {
      standalones.push({
        key: clusterKey(service.environment, service.serviceName),
        environment: service.environment,
        name: service.serviceName,
        kind: 'standalone',
        routers: [],
        sets: [],
        standalone: member,
      });
      continue;
    }
    const key = clusterKey(service.environment, name);
    const group = groups.get(key) ?? {
      environment: service.environment,
      name,
      members: [],
    };

    group.label ??= service.cluster || undefined;
    group.members.push(member);
    groups.set(key, group);
  }

  const clusters = [...groups.entries()].flatMap(
    ([key, { environment, name, label, members }]): ClusterModel[] => {
      const routers = members.filter((member) => member.router);
      const sets = new Map<string, SetModel>();
      const unplaced: MemberModel[] = [];

      for (const member of members) {
        if (member.router) {
          continue;
        }
        const setName = setNameOf(member, metrics);

        if (!setName) {
          unplaced.push(member);
          continue;
        }
        const set = sets.get(setName) ?? {
          name: setName,
          config: false,
          members: [],
        };

        set.config ||= isConfigServer(member, metrics);
        set.members.push(member);
        sets.set(setName, set);
      }

      const config = [...sets.values()].find((set) => set.config);
      const dataSets = [...sets.values()].filter((set) => !set.config);
      const sharded = routers.length > 0 || !!config || dataSets.length > 1;

      // A member without a set name in a sharded cluster is a router whose exporter is silent.
      if (sharded) {
        routers.push(...unplaced);
      } else if (dataSets.length === 1) {
        dataSets[0].members.push(...unplaced);
      } else {
        return unplaced.map((member) => ({
          key: clusterKey(environment, member.service.serviceName),
          environment,
          name: unplaced.length === 1 ? name : member.service.serviceName,
          ...(label ? { label } : {}),
          kind: 'standalone',
          routers: [],
          sets: [],
          standalone: member,
        }));
      }

      [...sets.values()].forEach((set) => set.members.sort(byTopology));
      routers.sort(byTopology);

      return [
        {
          key,
          environment,
          name,
          ...(label ? { label } : {}),
          kind: sharded ? 'sharded' : 'replicaSet',
          routers,
          config,
          sets: dataSets.sort(byName),
        },
      ];
    }
  );

  return [...clusters, ...standalones];
};

/** Every member of a cluster, routers and config servers included. */
export const clusterMembers = (cluster: ClusterModel): MemberModel[] => [
  ...cluster.routers,
  ...(cluster.config?.members ?? []),
  ...cluster.sets.flatMap((set) => set.members),
  ...(cluster.standalone ? [cluster.standalone] : []),
];

export const primaryOf = (set: SetModel) =>
  set.members.find(
    (member) => member.verdict === 'up' && member.state === 'PRIMARY'
  );

const isSecondary = (member: MemberModel) =>
  member.verdict === 'up' && member.state === 'SECONDARY';

const isPrimary = (member: MemberModel) =>
  member.verdict === 'up' && member.state === 'PRIMARY';

export const memberLagLevel = (member: MemberModel): HubLevel =>
  member.lag && isSecondary(member) ? lagLevel(member.lag.sustained) : 'ok';

export const primaryOplogLevel = (set: SetModel): HubLevel | undefined => {
  const primary = primaryOf(set);

  return primary?.oplogWindow !== undefined
    ? oplogLevel(primary.oplogWindow, primary.oplogFill)
    : undefined;
};

/** What the health rules read of a member; lag and oplog only count inside a set. */
const healthInput = (
  member: MemberModel,
  inSet = false
): MemberHealthInput => ({
  name: member.service.serviceName,
  verdict: member.verdict,
  ...(member.state ? { state: member.state } : {}),
  ...(member.stateSeconds !== undefined
    ? { stateSeconds: member.stateSeconds }
    : {}),
  ...(member.version ? { version: member.version } : {}),
  ...(inSet && member.lag && isSecondary(member) ? { lag: member.lag } : {}),
  ...(inSet && member.oplogWindow !== undefined && isPrimary(member)
    ? {
        oplog: {
          seconds: member.oplogWindow,
          ...(member.oplogFill !== undefined ? { fill: member.oplogFill } : {}),
        },
      }
    : {}),
});

export const setVerdictOf = (set: SetModel): HealthVerdict =>
  setHealth(
    set.members.map((member) => healthInput(member, true)),
    set.name
  );

export const setHealthOf = (set: SetModel): HubHealth =>
  setVerdictOf(set).health;

const routersVerdictOf = (cluster: ClusterModel): HealthVerdict =>
  routersHealth(
    cluster.routers.map((member) => healthInput(member)),
    Messages.routersName
  );

const memberVerdictOf = (member: MemberModel, set?: SetModel): HealthVerdict =>
  memberHealth(healthInput(member, !!set));

/** A cluster's health and the conditions behind it; an upgrade in progress decides before anything else. */
export const clusterVerdictOf = (cluster: ClusterModel): HealthVerdict => {
  const own =
    cluster.kind === 'standalone' && cluster.standalone
      ? standaloneHealth(healthInput(cluster.standalone))
      : cluster.kind === 'replicaSet'
        ? setVerdictOf(cluster.sets[0])
        : shardedHealth({
            shards: cluster.sets.map(setVerdictOf),
            config: cluster.config ? setVerdictOf(cluster.config) : undefined,
            routers: cluster.routers.length
              ? routersVerdictOf(cluster)
              : undefined,
          });

  return cluster.upgrade ? upgradingHealth(own, cluster.upgrade) : own;
};

export const clusterHealthOf = (cluster: ClusterModel): HubHealth =>
  clusterVerdictOf(cluster).health;

export const setHasNoPrimary = (set: SetModel) => hasNoPrimary(set.members);

const isoFromSeconds = (seconds?: number) =>
  seconds === undefined ? undefined : new Date(seconds * 1000).toISOString();

const duration = (seconds: number) => formatCompactDuration(seconds) || '0s';

export interface TreeOptions {
  capabilities: Pick<HubCapabilities, 'operations'>;
  /** `${env ?? ''}/${cluster}` to OmCluster.id, with Operations on. */
  omClusterIds?: Record<string, string>;
  /** The version check by cluster key; absent or null while nothing checks. */
  updates?: Record<string, MongoUpdateCheck> | null;
  now?: number;
}

const setDrift = (set: SetModel) =>
  versionDrift(set.members.map((member) => healthInput(member)));

/** The lowest version the reachable members run: what they are still exposed on. */
const versionOf = (
  members: MemberModel[],
  drift?: VersionGroup[]
): HubVersion | undefined => {
  const effective = lowestVersion(
    members.flatMap((member) =>
      member.verdict === 'up' && member.version ? [member.version] : []
    )
  );

  return effective ? { effective, ...(drift ? { drift } : {}) } : undefined;
};

/** A failed check, an old one or none at all never reads as up to date. */
export const updateOf = (
  effective: string,
  check: MongoUpdateCheck | undefined,
  now: number
): HubUpdate => {
  if (!check) {
    return { state: 'unknown', reason: 'stale' };
  }
  const { checkedAt, latest } = check;
  const age = now - Date.parse(checkedAt);

  if (Number.isNaN(age) || age > T.updateCheckStaleSeconds * 1000) {
    return { state: 'unknown', reason: 'stale', checkedAt };
  }
  if (!latest) {
    return { state: 'unknown', reason: 'failed', checkedAt };
  }

  return compareVersions(effective, latest) < 0
    ? { state: 'available', latest, checkedAt }
    : { state: 'upToDate', checkedAt };
};

const clusterVersion = (
  cluster: ClusterModel,
  version: HubVersion | undefined,
  options: TreeOptions
): HubVersion | undefined =>
  version && options.updates
    ? {
        ...version,
        update: updateOf(
          version.effective,
          options.updates[cluster.key],
          options.now ?? Date.now()
        ),
      }
    : version;

export const placeOf = (cluster: ClusterModel): MongoPlace => ({
  environment: cluster.environment,
  cluster: cluster.label,
});

const notApplicable = (reason: string): HubMeasure => ({
  kind: 'notApplicable',
  reason,
});

const noData = (member: MemberModel): HubMeasure => ({
  kind: 'noData',
  lastSeen: isoFromSeconds(member.lastUpAt),
});

const memberLag = (
  member: MemberModel,
  set: SetModel,
  place: MongoPlace
): HubMeasure => {
  if (member.verdict === 'down') {
    return notApplicable(Messages.measures.downMember);
  }
  if (member.verdict === 'noData') {
    return notApplicable(Messages.measures.noDataMember);
  }
  if (member.state === 'PRIMARY') {
    return notApplicable(Messages.measures.primaryLag);
  }
  if (member.state === 'ARBITER') {
    return notApplicable(Messages.measures.arbiterLag);
  }
  if (!member.lag) {
    return primaryOf(set)
      ? noData(member)
      : notApplicable(Messages.measures.noPrimaryLag);
  }
  const level = memberLagLevel(member);

  return {
    kind: 'value',
    seconds: member.lag.now,
    level,
    ...(level !== 'ok'
      ? {
          rule: Messages.measures.lagRule(
            duration(T.lagWarningSeconds),
            duration(member.lag.now)
          ),
        }
      : {}),
    door: lagDoor(place, set.name, member.service.serviceName),
  };
};

const fillPercent = (fill?: number) =>
  fill === undefined ? 0 : Math.round(fill * 100);

const memberOplog = (
  member: MemberModel,
  set: SetModel,
  place: MongoPlace
): HubMeasure => {
  if (member.verdict === 'down') {
    return notApplicable(Messages.measures.downMember);
  }
  if (member.verdict === 'noData') {
    return notApplicable(Messages.measures.noDataMember);
  }
  if (member.oplogWindow === undefined) {
    return member.state === 'ARBITER'
      ? notApplicable(Messages.measures.arbiterLag)
      : noData(member);
  }
  // Only the primary's window is a budget the set runs out of.
  const level =
    member.state === 'PRIMARY'
      ? oplogLevel(member.oplogWindow, member.oplogFill)
      : 'ok';

  return {
    kind: 'value',
    seconds: member.oplogWindow,
    level,
    ...(level !== 'ok'
      ? {
          rule: Messages.measures.oplogRule(
            duration(member.oplogWindow),
            fillPercent(member.oplogFill)
          ),
        }
      : {}),
    door: oplogDoor(
      place,
      set.name,
      member.service.serviceName,
      member.service.serviceName
    ),
  };
};

/** The furthest-behind secondary's lag. */
const setLag = (set: SetModel, place: MongoPlace): HubMeasure => {
  const lagging = set.members
    .filter((member) => isSecondary(member) && member.lag)
    .sort((a, b) => (b.lag?.now ?? 0) - (a.lag?.now ?? 0));

  if (!lagging.length) {
    return notApplicable(Messages.measures.noPrimaryLag);
  }
  const worst = lagging[0];
  const levels = lagging.map(memberLagLevel);
  const level = worstLevel(levels);
  const seconds = worst.lag?.now ?? 0;

  return {
    kind: 'value',
    seconds,
    level,
    ...(level !== 'ok'
      ? {
          rule: Messages.measures.worstLagRule(
            worst.service.serviceName,
            duration(seconds)
          ),
        }
      : {}),
    door: lagDoor(place, set.name, set.name),
  };
};

const setOplog = (set: SetModel, place: MongoPlace): HubMeasure => {
  const primary = primaryOf(set);

  if (!primary) {
    return notApplicable(Messages.measures.noPrimaryOplog);
  }
  if (primary.oplogWindow === undefined) {
    return noData(primary);
  }
  const level = oplogLevel(primary.oplogWindow, primary.oplogFill);

  return {
    kind: 'value',
    seconds: primary.oplogWindow,
    level,
    ...(level !== 'ok'
      ? {
          rule: Messages.measures.oplogRule(
            duration(primary.oplogWindow),
            fillPercent(primary.oplogFill)
          ),
        }
      : {}),
    door: oplogDoor(place, set.name, set.name, primary.service.serviceName),
  };
};

const LEVEL_ORDER: Record<HubLevel, number> = { error: 0, warning: 1, ok: 2 };

/** The measure that needs the reader most: worst level, then the higher lag or tighter window. */
type ValueMeasure = Extract<HubMeasure, { kind: 'value' }>;

const pickMeasure = (
  measures: HubMeasure[],
  prefer: 'max' | 'min'
): ValueMeasure | undefined =>
  measures
    .filter((measure): measure is ValueMeasure => measure.kind === 'value')
    .sort(
      (a, b) =>
        LEVEL_ORDER[a.level] - LEVEL_ORDER[b.level] ||
        (prefer === 'max' ? b.seconds - a.seconds : a.seconds - b.seconds)
    )[0];

const memberType = (member: MemberModel) => {
  if (member.router) {
    return Messages.types.router;
  }
  switch (member.state) {
    case 'PRIMARY':
      return Messages.types.primary;
    case 'SECONDARY':
      return Messages.types.secondary;
    case 'ARBITER':
      return Messages.types.arbiter;
    default:
      return (
        (member.state && Messages.memberStates[member.state]) ||
        Messages.types.member
      );
  }
};

const memberSince = (member: MemberModel) =>
  member.verdict === 'up' ? undefined : isoFromSeconds(member.lastUpAt);

const membersCount = (members: MemberModel[]) => ({
  up: members.filter((member) => member.verdict === 'up').length,
  total: members.length,
  noData: members.filter((member) => member.verdict === 'noData').length,
});

const membersDoor = (
  name: string,
  to: string,
  members: MemberModel[]
): HubDoor => {
  const { up, total } = membersCount(members);

  return {
    label: `${up}/${total}`,
    to,
    context: Messages.doors.membersOf(name),
  };
};

const earliestSince = (members: MemberModel[]) =>
  members
    .map(memberSince)
    .filter((since): since is string => !!since)
    .sort()[0];

const memberNode = (
  member: MemberModel,
  place: MongoPlace,
  set?: SetModel
): HubTreeNode => {
  const { serviceName, serviceId } = member.service;
  const summary = member.router
    ? routerSummary(place, serviceName)
    : instanceSummary(place, serviceName);
  const name: HubDoor = { label: serviceName, to: summary };
  const measures: Record<string, HubMeasure> = member.router
    ? {
        lag: notApplicable(Messages.measures.routerLag),
        oplogWindow: notApplicable(Messages.measures.routerOplog),
      }
    : set
      ? {
          lag: memberLag(member, set, place),
          oplogWindow: memberOplog(member, set, place),
        }
      : {
          lag: notApplicable(Messages.measures.standaloneLag),
          oplogWindow: notApplicable(Messages.measures.standaloneOplog),
        };
  const since = memberSince(member);
  const { health } = memberVerdictOf(member, set);

  return {
    id: `member:${serviceId}`,
    kind: 'member',
    name: serviceName,
    type: memberType(member),
    health,
    showHealth: health !== 'healthy',
    ...(since ? { since } : {}),
    ...(member.version ? { version: { effective: member.version } } : {}),
    measures,
    doors: {
      name,
      menu: [
        { label: Messages.doors.openSummary, to: summary },
        {
          label: Messages.doors.queryAnalyticsService,
          to: clusterQan(place, serviceName),
        },
      ],
    },
  };
};

const clusterMenu = (
  cluster: ClusterModel,
  summary: string,
  options: TreeOptions
): HubDoor[] => [
  { label: Messages.doors.openSummary, to: summary },
  { label: Messages.doors.queryAnalytics, to: clusterQan(placeOf(cluster)) },
  { label: Messages.doors.backupStatus, to: backupStatus(placeOf(cluster)) },
  ...(options.capabilities.operations
    ? [manageDoor(options.omClusterIds?.[cluster.key])]
    : []),
];

const upgradeSince = (cluster: ClusterModel) =>
  cluster.upgrade ? { since: cluster.upgrade.startedAt } : {};

const setNode = (
  set: SetModel,
  cluster: ClusterModel,
  type: string
): HubTreeNode => {
  const place = placeOf(cluster);
  const summary = replsetSummary(place, set.name);
  const health = cluster.upgrade ? 'upgrading' : setHealthOf(set);

  return {
    id: `set:${cluster.key}/${set.name}`,
    kind: 'set',
    name: set.name,
    type,
    health,
    showHealth: true,
    ...(setHasNoPrimary(set)
      ? { healthDetail: Messages.health.noPrimary }
      : {}),
    ...(health === 'down' && earliestSince(set.members)
      ? { since: earliestSince(set.members) }
      : {}),
    ...upgradeSince(cluster),
    members: membersCount(set.members),
    version: versionOf(set.members, setDrift(set)),
    measures: { lag: setLag(set, place), oplogWindow: setOplog(set, place) },
    doors: {
      name: { label: set.name, to: summary },
      members: membersDoor(set.name, summary, set.members),
      menu: [
        { label: Messages.doors.openSummary, to: summary },
        { label: Messages.doors.queryAnalytics, to: clusterQan(place) },
      ],
    },
    children: set.members.map((member) => memberNode(member, place, set)),
  };
};

const routersNode = (cluster: ClusterModel): HubTreeNode => {
  const place = placeOf(cluster);
  const summary = routerSummary(place);

  return {
    id: `group:${cluster.key}/routers`,
    kind: 'group',
    name: Messages.routersName,
    type: Messages.types.routers,
    health: cluster.upgrade ? 'upgrading' : routersVerdictOf(cluster).health,
    ...upgradeSince(cluster),
    showHealth: true,
    members: membersCount(cluster.routers),
    version: versionOf(cluster.routers),
    measures: {
      lag: notApplicable(Messages.measures.routerLag),
      oplogWindow: notApplicable(Messages.measures.routerOplog),
    },
    doors: {
      name: { label: Messages.routersName, to: summary },
      members: membersDoor(Messages.routersName, summary, cluster.routers),
      menu: [{ label: Messages.doors.openSummary, to: summary }],
    },
    children: cluster.routers.map((member) => memberNode(member, place)),
  };
};

/** The versions of every set that drifts, merged by version. */
const shardedDrift = (cluster: ClusterModel): VersionGroup[] | undefined => {
  const groups = [
    ...(cluster.config ? [cluster.config] : []),
    ...cluster.sets,
  ].flatMap((set) => setDrift(set) ?? []);

  return groups.length
    ? groupByVersion(
        groups.flatMap(({ version, members }) =>
          members.map((name) => ({ name, version }))
        )
      )
    : undefined;
};

const clusterNode = (
  cluster: ClusterModel,
  options: TreeOptions
): HubTreeNode => {
  const place = placeOf(cluster);
  const id = `cluster:${cluster.key}`;
  const health = clusterHealthOf(cluster);
  const members = clusterMembers(cluster);

  if (cluster.kind === 'standalone' && cluster.standalone) {
    const node = memberNode(cluster.standalone, place);

    return {
      ...node,
      id,
      kind: 'cluster',
      name: cluster.name,
      type: Messages.types.standalone,
      environment: cluster.environment,
      health,
      showHealth: true,
      ...upgradeSince(cluster),
      members: membersCount(members),
      version: clusterVersion(cluster, versionOf(members), options),
      doors: {
        name: { label: cluster.name, to: node.doors.name.to },
        members: membersDoor(cluster.name, node.doors.name.to, members),
        menu: clusterMenu(cluster, node.doors.name.to, options),
      },
    };
  }

  if (cluster.kind === 'replicaSet') {
    const set = setNode(cluster.sets[0], cluster, Messages.types.replicaSet);

    return {
      ...set,
      id,
      kind: 'cluster',
      name: cluster.name,
      environment: cluster.environment,
      version: clusterVersion(cluster, set.version, options),
      doors: {
        name: { label: cluster.name, to: set.doors.name.to },
        members: membersDoor(cluster.name, set.doors.name.to, members),
        menu: clusterMenu(cluster, set.doors.name.to, options),
      },
    };
  }

  const summary = clusterSummary(place);
  const shards = cluster.sets
    .map((set) => setNode(set, cluster, Messages.types.shard))
    .sort((a, b) => compareHealth(a.health, b.health) || byName(a, b));
  const config = cluster.config
    ? setNode(cluster.config, cluster, Messages.types.configServers)
    : undefined;
  const children = [
    ...(cluster.routers.length ? [routersNode(cluster)] : []),
    ...(config ? [config] : []),
    ...shards,
  ];
  const lag = pickMeasure(
    children.map((child) => child.measures.lag),
    'max'
  );
  const oplog = pickMeasure(
    shards.map((shard) => shard.measures.oplogWindow),
    'min'
  );

  return {
    id,
    kind: 'cluster',
    name: cluster.name,
    type: Messages.types.sharded,
    environment: cluster.environment,
    health,
    showHealth: true,
    ...upgradeSince(cluster),
    members: membersCount(members),
    version: clusterVersion(
      cluster,
      versionOf(members, shardedDrift(cluster)),
      options
    ),
    measures: {
      lag: lag
        ? { ...lag, door: lag.door && { ...lag.door, label: cluster.name } }
        : notApplicable(Messages.measures.noPrimaryLag),
      oplogWindow: oplog
        ? {
            ...oplog,
            ...(oplog.level !== 'ok' && oplog.door
              ? {
                  rule: Messages.measures.tightestOplogRule(
                    oplog.door.label,
                    duration(oplog.seconds)
                  ),
                }
              : {}),
          }
        : notApplicable(Messages.measures.noPrimaryOplog),
    },
    doors: {
      name: { label: cluster.name, to: summary },
      members: membersDoor(cluster.name, summary, members),
      menu: clusterMenu(cluster, summary, options),
    },
    children,
  };
};

/** The clusters tree, worst first; members keep topology order so a set keeps its shape. */
export const buildClusterTree = (
  clusters: ClusterModel[],
  options: TreeOptions
): HubTreeNode[] =>
  clusters
    .map((cluster) => clusterNode(cluster, options))
    .sort((a, b) => compareHealth(a.health, b.health) || byName(a, b));

/** A node that needs the reader; an upgrade is planned work, not a problem. */
export const isProblem = (node: Pick<HubTreeNode, 'health'>) =>
  node.health !== 'healthy' && node.health !== 'upgrading';

/** MRT's expanded state: every node with a problem and children, and every ancestor of one. */
export const expandedProblems = (
  nodes: HubTreeNode[]
): Record<string, true> => {
  const expanded: Record<string, true> = {};

  const visit = (node: HubTreeNode): boolean => {
    const childProblem = (node.children ?? []).map(visit).some(Boolean);
    const problem = isProblem(node) || childProblem;

    if (problem && node.children?.length) {
      expanded[node.id] = true;
    }

    return problem;
  };

  nodes.forEach(visit);

  return expanded;
};
