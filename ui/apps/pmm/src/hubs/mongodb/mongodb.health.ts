import type { HubHealth, HubLevel } from 'hubs/template/hub.types';
import { MONGODB_HUB_THRESHOLDS as T } from './mongodb.thresholds';
import type { MongoMetrics, MongoServiceInput } from './mongodb.types';

/** Up, Down, or No data: PMM has no up series and the inventory does not say down. */
export type MemberVerdict = 'up' | 'down' | 'noData';

/** Worst first. */
export const HEALTH_RANK: Record<HubHealth, number> = {
  down: 0,
  degraded: 1,
  noData: 2,
  healthy: 3,
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

export const memberHealth = (verdict: MemberVerdict): HubHealth =>
  verdict === 'up' ? 'healthy' : verdict === 'down' ? 'down' : 'noData';

export interface SetMemberHealthInput {
  verdict: MemberVerdict;
  /** member_state, such as PRIMARY. */
  state?: string;
}

/** At least one member answers and none of them is primary, so the set takes no writes. */
export const hasNoPrimary = (members: SetMemberHealthInput[]): boolean =>
  members.some((member) => member.verdict === 'up') &&
  !members.some(
    (member) => member.verdict === 'up' && member.state === 'PRIMARY'
  );

export interface SetHealthInput {
  members: SetMemberHealthInput[];
  /** Levels of each secondary's lag. */
  lagLevels: HubLevel[];
  primaryOplogLevel?: HubLevel;
}

/** A replica set, shard or config server set. */
export const setHealth = ({
  members,
  lagLevels,
  primaryOplogLevel,
}: SetHealthInput): HubHealth => {
  if (!members.length) {
    return 'noData';
  }
  if (members.every((member) => member.verdict === 'down')) {
    return 'down';
  }
  if (hasNoPrimary(members)) {
    return 'down';
  }
  if (members.every((member) => member.verdict === 'noData')) {
    return 'noData';
  }
  if (
    members.some((member) => member.verdict !== 'up') ||
    lagLevels.some(isPastWarning) ||
    isPastWarning(primaryOplogLevel)
  ) {
    return 'degraded';
  }

  return 'healthy';
};

export const routersHealth = (verdicts: MemberVerdict[]): HubHealth => {
  if (!verdicts.length) {
    return 'noData';
  }
  if (verdicts.every((verdict) => verdict === 'down')) {
    return 'down';
  }
  if (verdicts.every((verdict) => verdict === 'noData')) {
    return 'noData';
  }

  return verdicts.some((verdict) => verdict !== 'up') ? 'degraded' : 'healthy';
};

export interface ShardedHealthInput {
  shards: HubHealth[];
  config?: HubHealth;
  routers?: HubHealth;
}

export const shardedHealth = ({
  shards,
  config,
  routers,
}: ShardedHealthInput): HubHealth => {
  const children = [
    ...shards,
    ...(config ? [config] : []),
    ...(routers ? [routers] : []),
  ];

  if (!children.length) {
    return 'noData';
  }
  if (children.some((health) => health === 'down')) {
    return 'down';
  }
  if (children.every((health) => health === 'noData')) {
    return 'noData';
  }

  return children.some((health) => health !== 'healthy')
    ? 'degraded'
    : 'healthy';
};

export const standaloneHealth = (verdict: MemberVerdict): HubHealth =>
  memberHealth(verdict);

/** Sustained lag decides, so one slow scrape does not flag a member. */
export const lagLevel = (
  sustainedSeconds: number,
  primaryWindowSeconds?: number
): HubLevel => {
  if (
    sustainedSeconds >= T.lagErrorSeconds ||
    (primaryWindowSeconds !== undefined &&
      primaryWindowSeconds > 0 &&
      sustainedSeconds >= primaryWindowSeconds * T.lagErrorOplogRatio)
  ) {
    return 'error';
  }

  return sustainedSeconds >= T.lagWarningSeconds ? 'warning' : 'ok';
};

/** A short window counts only once the oplog is nearly full; an unknown fill is not full. */
export const oplogLevel = (windowSeconds: number, fill?: number): HubLevel => {
  if (fill === undefined || fill < T.oplogFullRatio) {
    return 'ok';
  }
  if (windowSeconds < T.oplogErrorSeconds) {
    return 'error';
  }

  return windowSeconds < T.oplogWarningSeconds ? 'warning' : 'ok';
};

export const compareHealth = (a: HubHealth, b: HubHealth) =>
  HEALTH_RANK[a] - HEALTH_RANK[b];

export const worstHealth = (healths: HubHealth[]): HubHealth =>
  healths.reduce<HubHealth>(
    (worst, health) => (compareHealth(health, worst) < 0 ? health : worst),
    'healthy'
  );
