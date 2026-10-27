import type {
  HubAttentionSeverity,
  HubHealth,
  HubRecommendationKind,
  HubSeverity,
} from './hub.types';

export const Messages = {
  freshness: {
    selector: 'Auto-refresh',
    offOption: 'Off',
    offSelected: 'OFF',
    notYet: 'Not updated yet',
    stale: (age: string) => `Data paused, last update ${age}`,
    refresh: 'Refresh now',
    retry: 'Retry',
    refreshed: 'Updated just now',
    /** Said once, rather than an age that changes every minute. */
    announceStale: 'Data paused.',
  },
  add: 'Add',
  /** What Add adds, for assistive tech: "MongoDB". */
  addContext: (name: string) => name,
  moreActions: 'More actions',
  newTab: '(opens in a new tab)',
  environment: {
    label: 'Environment',
    all: 'All environments',
    showAll: 'Show all environments',
    nothing: (name: string) => `No ${name} in this environment`,
    noMatch: (unit: string) => `No matching ${unit}`,
    noService: (name: string) => `No ${name} service is in this environment.`,
    pickAnother: 'Pick another one, or show them all.',
  },
  banner: {
    /** The banner's accessible name: "MongoDB status". */
    label: (name: string) => `${name} status`,
    health: {
      checking: ([, many]: [string, string]) => `Checking ${many}...`,
      empty: (name: string) => `No ${name} monitored yet`,
      unknown: ([one]: [string, string]) =>
        `${one.charAt(0).toUpperCase()}${one.slice(1)} health unknown`,
      healthy: (total: number, [one, many]: [string, string]) =>
        total === 1
          ? `1 ${one} is healthy`
          : `All ${total} ${many} are healthy`,
      attention: (
        count: number,
        total: number,
        [one, many]: [string, string]
      ) =>
        total === 1
          ? `1 ${one} needs attention`
          : count === total
            ? `All ${total} ${many} need attention`
            : `${count} of ${total} ${many} ${count === 1 ? 'needs' : 'need'} attention`,
      degraded: (count: number, total: number, [one, many]: [string, string]) =>
        total === 1
          ? `1 ${one} is degraded`
          : count === total
            ? `All ${total} ${many} are degraded`
            : `${count} of ${total} ${many} degraded`,
      upgrading: (
        count: number,
        total: number,
        [one, many]: [string, string]
      ) =>
        total === 1
          ? `1 ${one} is upgrading`
          : `${count} of ${total} ${many} upgrading`,
      review: (count: number) =>
        count === 1 ? 'Review 1 issue' : `Review ${count} issues`,
    },
    alerts: {
      checking: 'Checking alerts...',
      firing: (count: number, critical = 0) =>
        `${count === 1 ? '1 alert' : `${count} alerts`} firing${critical ? `, ${critical === count ? 'all' : critical} critical` : ''}`,
      none: 'No alerts firing',
      notSetUp: 'No alerts set up',
      unknown: 'Alerts unavailable',
    },
    backups: {
      checking: 'Checking backups...',
      onSchedule: (age: string) => `Backups on schedule, last ${age}`,
      failed: (count: number, age?: string) =>
        `${count === 1 ? '1 backup' : `${count} backups`} failed, ${age ? `${count === 1 ? 'last' : 'oldest'} good backup ${age}` : 'no good backup yet'}`,
      protected: (
        count: number,
        total: number,
        [one, many]: [string, string]
      ) => `${count} of ${total} ${total === 1 ? one : many} protected`,
      notSetUp: 'No backups set up',
      noneYet: 'No backup taken yet',
      off: 'Backup management is off',
      unknown: 'Backups unavailable',
    },
    openAlerts: 'Open alerts',
    openBackups: 'Open backups',
    setUpAlerts: 'Set up alerts',
    setUpBackups: 'Set up backups',
  },
  /** The row of tiles under the header. */
  shortcuts: 'Shortcuts',
  sections: {
    attention: 'Needs attention',
    recommendations: 'Recommendations',
    clusters: 'Clusters',
    backups: 'Backups',
    queries: 'Queries',
    dashboards: 'Dashboards',
  },
  sectionCount: {
    attention: (count: number) => (count === 1 ? 'issue' : 'issues'),
    clusters: (count: number, [one, many]: [string, string]) =>
      count === 1 ? one : many,
  },
  sectionError: {
    fallback: 'Could not load this section.',
    retry: 'Retry',
    alerts: (reason?: string) =>
      `Firing alerts could not be read${reason ? ` (${reason})` : ''}, so none is listed here.`,
  },
  attention: {
    showMore: (count: number) => `Show ${count} more`,
    showFewer: 'Show fewer',
    severity: {
      critical: 'Critical',
      error: 'Error',
      warning: 'Warning',
    } satisfies Record<HubAttentionSeverity, string>,
    /** "for 18m", "for up to 40m"; an event keeps its "8h ago". */
    since: (since: string) => (/\bago$/.test(since) ? since : `for ${since}`),
    alertFiring: 'Alert firing',
  },
  recommendations: {
    kind: {
      notset: 'Not set up',
      info: 'Info',
    } satisfies Record<HubRecommendationKind, string>,
  },
  health: {
    healthy: 'Healthy',
    degraded: 'Degraded',
    down: 'Down',
    upgrading: 'Upgrading',
    noData: 'No data',
  } satisfies Record<HubHealth, string>,
  healthFor: (duration: string) => `for ${duration}`,
  healthSince: (health: string, time: string) => `${health} since ${time}`,
  conditionsMore: (count: number) => ` · ${count} more`,
  clusters: {
    columns: {
      name: 'Name',
      type: 'Type',
      environment: 'Environment',
      health: 'Health',
      members: 'Members up',
      version: 'Version',
    },
    patchesFilter: {
      label: (count: number) =>
        count > 0 ? `${count} behind on patches` : 'Behind on patches',
      none: 'No cluster is behind on patches',
      noneDescription:
        'Each one runs the newest patch release of its series, or its version check could not tell.',
      showAll: 'Show all clusters',
    },
    membersNoData: (count: number) => ` · ${count} no data`,
    expandColumn: 'Expand',
    inParent: (path: string) => `in ${path}`,
    path: (names: string[]) => names.join(' › '),
    expand: (name: string) => `Expand ${name}`,
    collapse: (name: string) => `Collapse ${name}`,
  },
  version: {
    mixed: 'mixed versions',
    drift: (parts: string[]) => `Mixed versions: ${parts.join('; ')}`,
    on: (version: string, members: string[]) =>
      `${version} on ${members.join(', ')}`,
    updateAvailable: 'Update available',
    checkFailed: 'Check failed',
    available: (latest: string, age: string) =>
      `${latest} is available, checked ${age}`,
    upToDate: (age: string) => `Up to date, checked ${age}`,
    failed: (age?: string) =>
      `The version check failed${age ? ` ${age}` : ''}, so PMM can't say whether an update exists`,
    stale: (age?: string) =>
      age
        ? `The version check last ran ${age}, too long ago to trust`
        : 'No version check has run yet',
  },
  measure: {
    over: {
      warning: 'over the warning threshold',
      error: 'over the error threshold',
    },
    noData: 'No data',
    lastSeen: (time: string) => `Last seen ${time}`,
  },
  queries: {
    slower: 'Slower than usual',
    none: 'No query is slower than usual.',
    severity: {
      error: 'Error',
      warning: 'Warning',
      info: 'Info',
    } satisfies Record<HubSeverity, string>,
  },
  dashboards: {
    recommended: 'Recommended for what’s running',
    forDevelopers: 'For developers',
    recent: 'Recently visited',
    recentEmpty: 'Dashboards you open appear here.',
  },
  clustersPage: {
    title: 'Clusters page (in design)',
    received: (count: number) =>
      count === 1
        ? 'The link asked for 1 cluster:'
        : `The link asked for ${count} clusters:`,
    none: 'The link asked for no cluster in particular.',
    fleet: 'It will be the Fleet page of Operations for MongoDB, improved.',
  },
  demo: {
    trigger: 'Demo controls',
    scenarioLabel: 'Scenario',
    scenario: {
      healthy: 'Healthy',
      incomplete: 'Incomplete',
      unhealthy: 'Unhealthy',
    },
    stale: 'Make the freshness indicator stale',
    operations: (name: string) => `Operations for ${name}`,
    adminOnly: 'Admins only',
  },
};
