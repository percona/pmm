import type { HubHealth, HubSeverity } from './hub.types';

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
      firing: (count: number) =>
        count === 1 ? '1 alert firing' : `${count} alerts firing`,
      none: 'No alerts firing',
      notSetUp: 'No alerts set up',
      unknown: 'Alerts unavailable',
    },
    backups: {
      checking: 'Checking backups...',
      onSchedule: (age: string) => `Backups on schedule, last ${age}`,
      failed: (count: number, age?: string) =>
        `${count === 1 ? '1 backup' : `${count} backups`} failed, ${age ? `last good ${age}` : 'no good backup yet'}`,
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
  sections: {
    attention: 'Needs attention',
    clusters: 'Clusters',
    backups: 'Backups',
    queries: 'Queries',
    dashboards: 'Dashboards',
    apps: 'Apps',
  },
  sectionCount: {
    attention: (count: number) => (count === 1 ? 'item' : 'items'),
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
    calm: (age: string) => `Nothing needs attention · checked ${age}`,
    showMore: (count: number) => `Show ${count} more`,
    showFewer: 'Show fewer',
    severity: {
      error: 'Error',
      warning: 'Warning',
      info: 'Info',
    } satisfies Record<HubSeverity, string>,
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
  backups: {
    tableLabel: 'Backups by cluster',
    columns: {
      cluster: 'Cluster',
      lastGood: 'Last good backup',
      pitr: 'Point-in-time recovery',
      nextRun: 'Next run',
    },
    noneYet: 'None yet',
    failed: (age: string) => `failed ${age}`,
    lastGoodSince: (age: string) => `last good ${age}`,
    pitr: {
      on: 'On',
      off: 'Off',
      stopped: 'Stopped',
      notStarted: 'Not started',
    },
    notApplicable: 'Not applicable',
    featureOff: 'Backup management is off.',
  },
  queries: {
    slower: 'Slower than usual',
    runningLong: 'Running long now',
    none: 'No query is slower than usual or running long.',
  },
  dashboards: {
    recommended: 'Recommended for what’s running',
    forDevelopers: 'For developers',
    recent: 'Recently visited',
    recentEmpty: 'Dashboards you open appear here.',
  },
  demo: {
    trigger: 'Demo controls',
    groups: {
      scenario: 'Scenario',
      freshness: 'Freshness',
      operations: (name: string) => `Operations for ${name}`,
    },
    scenario: {
      healthy: 'Healthy',
      incomplete: 'Incomplete',
      unhealthy: 'Unhealthy',
    },
    freshness: { live: 'Live', stale: 'Stale' },
    operations: { on: 'On', off: 'Off' },
    adminOnly: 'Admins only',
  },
};
