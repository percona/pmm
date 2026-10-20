import type {
  HubHealth,
  HubOpsMode,
  HubRoleMode,
  HubScenario,
  HubSeverity,
} from './hub.types';

export const Messages = {
  sampleData: 'Sample data',
  refresh: 'Refresh',
  refreshed: 'Updated just now',
  updated: (age: string) => `Updated ${age}`,
  environment: {
    label: 'Environment',
    all: 'All environments',
    showAll: 'Show all environments',
    nothing: (name: string) => `No ${name} in this environment`,
    noMatch: (unit: string) => `No matching ${unit}`,
    noService: (name: string) => `No ${name} service is in this environment.`,
    pickAnother: 'Pick another one, or show them all.',
  },
  status: {
    empty: (name: string) => `No ${name} monitored yet`,
    clusters: (count: number, [one, many]: [string, string]) =>
      `${count} ${count === 1 ? one : many}`,
    attention: (count: number) =>
      count === 1 ? '1 needs attention' : `${count} need attention`,
    noAttention: 'none need attention',
    otherIssues: (count: number) =>
      count === 1 ? '1 other issue' : `${count} other issues`,
    patchesBehind: (count: number) => `${count} behind on patches`,
    patchesNone: 'none behind on patches',
    patchesUnknown: (count: number) => `, ${count} unknown`,
    clustersUnknown: ([, many]: [string, string]) => `— ${many}`,
    clustersUnknownHidden: ([, many]: [string, string]) =>
      `Could not count the ${many}`,
    attentionUnknown: '— need attention',
    attentionUnknownHidden: 'Could not check what needs attention',
    lastGoodBackup: (age: string) => `last good backup ${age}`,
    noGoodBackup: 'no good backup yet',
    notSetUp: 'no backups set up',
    setUp: 'set up a backup',
    featureOff: 'backup management is off',
    backupUnknown: 'last good backup unknown',
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
      label: 'Behind on patches',
      clear: 'Clear the filter: behind on patches',
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
  prototype: {
    trigger: (parts: string[]) => `Prototype: ${parts.join(' · ')}`,
    groups: {
      data: 'Data',
      operations: 'Operations',
      role: 'Signed in as',
    },
    scenario: {
      live: 'Live',
      empty: 'Empty',
      healthy: 'Healthy',
      wrong: 'Things wrong',
    } satisfies Record<HubScenario, string>,
    ops: {
      live: 'As configured',
      on: 'On',
      off: 'Off',
    } satisfies Record<HubOpsMode, string>,
    opsSummary: {
      on: 'Ops on',
      off: 'Ops off',
    } satisfies Record<Exclude<HubOpsMode, 'live'>, string>,
    role: {
      live: 'As signed in',
      admin: 'Admin',
      viewer: 'Viewer',
    } satisfies Record<HubRoleMode, string>,
    hints: {
      sampleOnly: 'Sample data only',
      adminOnly: 'Admins only',
    },
  },
};
