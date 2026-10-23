import type { StateIndicatorState } from '@percona/peak-ui';
import { ATTENTION_SECTION_ID } from './hub.doors';
import { Messages } from './hub.messages';
import type { HubDoor, HubStatus } from './hub.types';
import { formatAge } from './sections/hub.time';

export interface HubStatusFacet {
  id: 'health' | 'alerts' | 'backups';
  state: StateIndicatorState;
  label: string;
  /** What to do about it; the banner puts the facet on a card with this action. */
  action?: HubDoor;
}

export interface HubStatusContext {
  /** Technology name, such as "MongoDB". */
  name: string;
  /** Singular and plural of the unit counted, such as ["cluster", "clusters"]. */
  unit: [string, string];
  loading: boolean;
  /** The environment filter matches no service, so there is nothing to count. */
  matchesNothing: boolean;
  /** Items in Needs attention, where a cluster problem is reviewed. */
  issues: number;
}

const B = Messages.banner;

/** Cluster health: problems first, then degraded, then upgrades in progress. */
const healthFacet = (
  { state, clusters: total, health }: HubStatus,
  { name, unit, loading, matchesNothing, issues }: HubStatusContext
): HubStatusFacet => {
  const facet = (
    facetState: HubStatusFacet['state'],
    label: string
  ): HubStatusFacet => ({ id: 'health', state: facetState, label });
  // A cluster problem is reviewed further down the page, in Needs attention.
  const problem = (facetState: HubStatusFacet['state'], label: string) => ({
    ...facet(facetState, label),
    ...(issues > 0
      ? {
          action: {
            label: B.health.review(issues),
            to: `#${ATTENTION_SECTION_ID}`,
          },
        }
      : {}),
  });

  if (loading) {
    return facet('loading', B.health.checking(unit));
  }
  if (state === 'empty') {
    return facet('notSetUp', B.health.empty(name));
  }
  if (matchesNothing) {
    return facet('default', Messages.environment.nothing(name));
  }
  if (!health || total == null) {
    return facet('default', B.health.unknown(unit));
  }
  // No data counts with down: nothing says the cluster works.
  const failing = health.down + health.noData;

  if (failing) {
    return problem(
      'mixedProblems',
      B.health.attention(failing + health.degraded, total, unit)
    );
  }
  if (health.degraded) {
    return problem('degraded', B.health.degraded(health.degraded, total, unit));
  }
  if (health.upgrading) {
    return facet(
      'information',
      B.health.upgrading(health.upgrading, total, unit)
    );
  }

  return facet('healthy', B.health.healthy(total, unit));
};

/** Firing alerts, else whether any rule watches the technology; nothing while alerting is off. */
const alertsFacet = (
  { alerts, alertRules, doors }: HubStatus,
  loading: boolean
): HubStatusFacet | undefined => {
  if (loading) {
    return { id: 'alerts', state: 'loading', label: B.alerts.checking };
  }
  if (alerts === undefined) {
    return undefined;
  }
  if (alerts === null) {
    return { id: 'alerts', state: 'default', label: B.alerts.unknown };
  }
  if (alerts > 0) {
    return {
      id: 'alerts',
      state: 'errors',
      label: B.alerts.firing(alerts),
      action: doors.alerts,
    };
  }
  if (alertRules === 'none') {
    return {
      id: 'alerts',
      state: 'notSetUp',
      label: B.alerts.notSetUp,
      ...(doors.setUpAlerts ? { action: doors.setUpAlerts } : {}),
    };
  }

  return { id: 'alerts', state: 'default', label: B.alerts.none };
};

/** Failed backups first, then clusters nothing protects, then the last good one. */
const backupsFacet = (
  { lastGoodBackup, backupsFailed, backupCoverage, doors }: HubStatus,
  { unit, loading }: HubStatusContext
): HubStatusFacet => {
  const setUp = doors.setUpBackups ? { action: doors.setUpBackups } : {};

  if (loading) {
    return { id: 'backups', state: 'loading', label: B.backups.checking };
  }
  if (lastGoodBackup === 'unknown') {
    return { id: 'backups', state: 'default', label: B.backups.unknown };
  }
  if (lastGoodBackup === 'featureOff') {
    return { id: 'backups', state: 'notSetUp', label: B.backups.off };
  }
  if (backupsFailed > 0) {
    return {
      id: 'backups',
      state: 'errors',
      label: B.backups.failed(
        backupsFailed,
        typeof lastGoodBackup === 'object'
          ? formatAge(lastGoodBackup.at)
          : undefined
      ),
      action: doors.backups,
    };
  }
  if (lastGoodBackup === 'notSetUp') {
    return {
      id: 'backups',
      state: 'notSetUp',
      label: B.backups.notSetUp,
      ...setUp,
    };
  }
  if (backupCoverage && backupCoverage.protected < backupCoverage.protectable) {
    return {
      id: 'backups',
      state: 'notSetUp',
      label: B.backups.protected(
        backupCoverage.protected,
        backupCoverage.protectable,
        unit
      ),
      ...setUp,
    };
  }
  if (lastGoodBackup === 'none') {
    return { id: 'backups', state: 'default', label: B.backups.noneYet };
  }

  return {
    id: 'backups',
    state: 'default',
    label: B.backups.onSchedule(formatAge(lastGoodBackup.at)),
  };
};

/** The status banner's facets: cluster health, alerts and backups, or health alone when there is nothing to count. */
export const statusFacets = (
  status: HubStatus,
  context: HubStatusContext
): HubStatusFacet[] => {
  const health = healthFacet(status, context);

  if (
    !context.loading &&
    (status.state === 'empty' || context.matchesNothing)
  ) {
    return [health];
  }
  const alerts = alertsFacet(status, context.loading);

  return [health, ...(alerts ? [alerts] : []), backupsFacet(status, context)];
};
