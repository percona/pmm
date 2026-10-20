import type { HubHealth, HubLevel } from 'hubs/template/hub.types';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import type { MongoMetrics, MongoServiceInput } from './mongodb.types';
import { groupByVersion, type VersionGroup } from './mongodb.versions';

/** Up, Down, or No data: PMM has no up series and the inventory does not say down. */
export type MemberVerdict = 'up' | 'down' | 'noData';

/** Worst first: No data sits next to Down, since nothing says the cluster is fine, and Upgrading is planned work. */
export const HEALTH_RANK: Record<HubHealth, number> = {
  down: 0,
  noData: 1,
  degraded: 2,
  upgrading: 3,
  healthy: 4,
};

const LEVEL_RANK: Record<HubLevel, number> = { error: 0, warning: 1, ok: 2 };

export const worstLevel = (levels: HubLevel[]): HubLevel =>
  levels.reduce<HubLevel>(
    (worst, level) => (LEVEL_RANK[level] < LEVEL_RANK[worst] ? level : worst),
    'ok'
  );

export const isPastWarning = (level: HubLevel | undefined) =>
  level === 'warning' || level === 'error';

/** With metrics missing altogether, the inventory's own status is all there is. */
export const memberVerdict = (
  service: MongoServiceInput,
  metrics: MongoMetrics | null
): MemberVerdict => {
  if (!metrics) {
    return service.inventoryStatus === 'up'
      ? 'up'
      : service.inventoryStatus === 'down'
        ? 'down'
        : 'noData';
  }
  const up = metrics.up[service.serviceId];

  if (up?.value === 0 || service.inventoryStatus === 'down') {
    return 'down';
  }

  return up ? 'up' : 'noData';
};

/** A rule a member, set, router group or standalone breaks: PMM-15299's triggering conditions. */
export type HealthCondition =
  | { kind: 'upgrading'; done: number; total: number; to?: string }
  | { kind: 'allDown'; total: number; set?: string }
  | { kind: 'noPrimary'; down: number; total: number; set?: string }
  | { kind: 'quorumLost'; down: number; total: number; set?: string }
  | { kind: 'memberLost'; member: string; state: string }
  | { kind: 'noData'; total: number; set?: string }
  | { kind: 'memberDown'; member: string }
  | { kind: 'memberNoData'; member: string }
  | { kind: 'memberState'; member: string; state: string; seconds?: number }
  | { kind: 'drift'; versions: VersionGroup[]; set?: string }
  | { kind: 'oplogShort'; member: string; seconds: number }
  | { kind: 'behind'; member: string; seconds: number };

export type HealthConditionKind = HealthCondition['kind'];

/** The health each condition gives its set, in the order the rules run: the first that holds decides. */
const CONDITION_HEALTH: Record<HealthConditionKind, HubHealth> = {
  upgrading: 'upgrading',
  allDown: 'down',
  noPrimary: 'down',
  quorumLost: 'down',
  memberLost: 'down',
  noData: 'noData',
  memberDown: 'degraded',
  memberNoData: 'degraded',
  memberState: 'degraded',
  drift: 'degraded',
  oplogShort: 'degraded',
  behind: 'degraded',
};

const CONDITION_ORDER = Object.keys(CONDITION_HEALTH) as HealthConditionKind[];

export interface HealthVerdict {
  health: HubHealth;
  /** Every condition that holds, the deciding one first. */
  conditions: HealthCondition[];
}

const byRule = (a: HealthCondition, b: HealthCondition) =>
  CONDITION_ORDER.indexOf(a.kind) - CONDITION_ORDER.indexOf(b.kind);

/** The first rule that holds decides; with none, healthy. */
export const verdictOf = (conditions: HealthCondition[]): HealthVerdict => {
  const sorted = [...conditions].sort(byRule);

  return {
    health: sorted.length ? CONDITION_HEALTH[sorted[0].kind] : 'healthy',
    conditions: sorted,
  };
};

export interface MemberHealthInput {
  /** Service name, which the conditions name. */
  name: string;
  verdict: MemberVerdict;
  /** member_state while up, such as PRIMARY. */
  state?: string;
  /** How long the member has been in its state; unknown when absent. */
  stateSeconds?: number;
  /** Secondaries only. */
  lag?: { now: number; sustained: number };
  /** Primaries only. */
  oplog?: { seconds: number; fill?: number };
  /** The server version, such as 7.0.24-13. */
  version?: string;
}

// PMM-15299's allow-list: the only states a healthy set's members hold.
const SERVING_STATES = new Set(['PRIMARY', 'SECONDARY', 'ARBITER']);
// A restart or a sync passes through these, so they count only past the grace period.
const TRANSITIONAL_STATES = new Set([
  'STARTUP',
  'STARTUP2',
  'RECOVERING',
  'UNKNOWN',
]);
// A member rolling back or removed from its set holds none of the set's data, whatever the others do.
const LOST_STATES = new Set(['ROLLBACK', 'REMOVED']);

type StateInput = Pick<MemberHealthInput, 'verdict' | 'state' | 'stateSeconds'>;

/** Up but rolling back or removed from its set. */
export const isLost = (member: StateInput): boolean =>
  member.verdict === 'up' && !!member.state && LOST_STATES.has(member.state);

/** Up outside the allow-list: in a transitional state past its grace period (or for an unknown time), or in any other state. */
export const isStuck = (member: StateInput): boolean =>
  member.verdict === 'up' &&
  !!member.state &&
  !SERVING_STATES.has(member.state) &&
  !LOST_STATES.has(member.state) &&
  (!TRANSITIONAL_STATES.has(member.state) ||
    member.stateSeconds === undefined ||
    member.stateSeconds >= T.transitionGraceSeconds);

/** The versions the answering members run, when they run more than one: an upgrade that stopped halfway. */
export const versionDrift = (
  members: Array<Pick<MemberHealthInput, 'name' | 'verdict' | 'version'>>
): VersionGroup[] | undefined => {
  const groups = groupByVersion(
    members.filter((member) => member.verdict === 'up')
  );

  return groups.length > 1 ? groups : undefined;
};

/** At least one member answers and none of them is primary, so the set takes no writes. */
export const hasNoPrimary = (members: StateInput[]): boolean =>
  members.some((member) => member.verdict === 'up') &&
  !members.some(
    (member) => member.verdict === 'up' && member.state === 'PRIMARY'
  );

/** Half the members or more are down, so no majority can elect or keep a primary; a silent member may still vote, so only down ones count. */
export const hasLostQuorum = (members: StateInput[]): boolean => {
  const down = members.filter((member) => member.verdict === 'down').length;

  return down > 0 && down * 2 >= members.length;
};

/** Sustained lag decides, so one slow scrape does not flag a member. */
export const lagLevel = (sustainedSeconds: number): HubLevel =>
  sustainedSeconds >= T.lagWarningSeconds ? 'warning' : 'ok';

/** A short window counts only once the oplog is nearly full; an unknown fill is not full. */
export const oplogLevel = (windowSeconds: number, fill?: number): HubLevel =>
  fill !== undefined &&
  fill >= T.oplogFullRatio &&
  windowSeconds < T.oplogWarningSeconds
    ? 'warning'
    : 'ok';

/** What one member breaks on its own; a down or silent one breaks nothing else PMM can see. */
export const memberConditions = (
  member: MemberHealthInput
): HealthCondition[] => {
  const { name } = member;

  if (member.verdict === 'down') {
    return [{ kind: 'memberDown', member: name }];
  }
  if (member.verdict === 'noData') {
    return [{ kind: 'memberNoData', member: name }];
  }
  const conditions: HealthCondition[] = [];

  if (member.state && isLost(member)) {
    conditions.push({ kind: 'memberLost', member: name, state: member.state });
  }
  if (member.state && isStuck(member)) {
    conditions.push({
      kind: 'memberState',
      member: name,
      state: member.state,
      ...(member.stateSeconds !== undefined
        ? { seconds: member.stateSeconds }
        : {}),
    });
  }
  if (
    member.oplog &&
    isPastWarning(oplogLevel(member.oplog.seconds, member.oplog.fill))
  ) {
    conditions.push({
      kind: 'oplogShort',
      member: name,
      seconds: member.oplog.seconds,
    });
  }
  if (member.lag && isPastWarning(lagLevel(member.lag.sustained))) {
    conditions.push({ kind: 'behind', member: name, seconds: member.lag.now });
  }

  return conditions;
};

/** A member on its own: Down or No data by its verdict, else what it breaks. */
export const memberHealth = (member: MemberHealthInput): HealthVerdict => {
  const conditions = memberConditions(member);

  if (member.verdict === 'down') {
    return { health: 'down', conditions };
  }
  if (member.verdict === 'noData') {
    return { health: 'noData', conditions };
  }

  return verdictOf(conditions);
};

const named = (set?: string) => (set ? { set } : {});

/** A replica set, shard or config server set, by PMM-15299's rules in their order. */
export const setHealth = (
  members: MemberHealthInput[],
  set?: string
): HealthVerdict => {
  if (!members.length) {
    return { health: 'noData', conditions: [] };
  }
  const total = members.length;
  const down = members.filter((member) => member.verdict === 'down').length;
  const setWide: HealthCondition[] = [];

  if (down === total) {
    setWide.push({ kind: 'allDown', total, ...named(set) });
  } else if (hasNoPrimary(members)) {
    setWide.push({ kind: 'noPrimary', down, total, ...named(set) });
  } else if (hasLostQuorum(members)) {
    setWide.push({ kind: 'quorumLost', down, total, ...named(set) });
  } else if (!members.some((member) => member.verdict === 'up')) {
    setWide.push({ kind: 'noData', total, ...named(set) });
  }

  // A set-wide rule already counts the down and silent members.
  const own = members
    .flatMap(memberConditions)
    .filter(
      (condition) =>
        !setWide.length ||
        (condition.kind !== 'memberDown' && condition.kind !== 'memberNoData')
    );
  const drift = versionDrift(members);

  return verdictOf([
    ...setWide,
    ...own,
    ...(drift
      ? [{ kind: 'drift' as const, versions: drift, ...named(set) }]
      : []),
  ]);
};

/** Routers share no data, so one answering keeps the cluster reachable. */
export const routersHealth = (
  routers: MemberHealthInput[],
  group?: string
): HealthVerdict => {
  if (!routers.length) {
    return { health: 'noData', conditions: [] };
  }
  const total = routers.length;

  if (routers.every((router) => router.verdict === 'down')) {
    return verdictOf([{ kind: 'allDown', total, ...named(group) }]);
  }
  if (!routers.some((router) => router.verdict === 'up')) {
    return verdictOf([{ kind: 'noData', total, ...named(group) }]);
  }

  return verdictOf(routers.flatMap(memberConditions));
};

export interface ShardedHealthInput {
  shards: HealthVerdict[];
  config?: HealthVerdict;
  routers?: HealthVerdict;
}

/** Down when any part is down, since its data or its way in is gone; No data only when every part is silent. */
export const shardedHealth = ({
  shards,
  config,
  routers,
}: ShardedHealthInput): HealthVerdict => {
  const parts = [
    ...shards,
    ...(config ? [config] : []),
    ...(routers ? [routers] : []),
  ];
  const conditions = parts.flatMap((part) => part.conditions).sort(byRule);

  if (!parts.length) {
    return { health: 'noData', conditions };
  }
  if (parts.some((part) => part.health === 'down')) {
    return { health: 'down', conditions };
  }
  if (parts.every((part) => part.health === 'noData')) {
    return { health: 'noData', conditions };
  }

  return {
    health: parts.some((part) => part.health !== 'healthy')
      ? 'degraded'
      : 'healthy',
    conditions,
  };
};

export const standaloneHealth = (member: MemberHealthInput): HealthVerdict => {
  if (member.verdict === 'down') {
    return verdictOf([{ kind: 'allDown', total: 1 }]);
  }
  if (member.verdict === 'noData') {
    return verdictOf([{ kind: 'noData', total: 1 }]);
  }

  return { health: 'healthy', conditions: [] };
};

/** PMM-15299 checks for an upgrade first: while one runs, the cluster reads Upgrading whatever else holds, and its mixed versions are the upgrade's. */
export const upgradingHealth = (
  own: HealthVerdict,
  upgrade: { done: number; total: number; to?: string }
): HealthVerdict => ({
  health: 'upgrading',
  conditions: [
    {
      kind: 'upgrading',
      done: upgrade.done,
      total: upgrade.total,
      ...(upgrade.to ? { to: upgrade.to } : {}),
    },
    ...own.conditions.filter((condition) => condition.kind !== 'drift'),
  ],
});

export const compareHealth = (a: HubHealth, b: HubHealth) =>
  HEALTH_RANK[a] - HEALTH_RANK[b];

export const worstHealth = (healths: HubHealth[]): HubHealth =>
  healths.reduce<HubHealth>(
    (worst, health) => (compareHealth(health, worst) < 0 ? health : worst),
    'healthy'
  );
