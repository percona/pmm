import type { HubMessages } from 'hubs/template/hub.types';

export const MONGODB_HUB_MESSAGES: HubMessages = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
  empty: {
    title: 'Start monitoring MongoDB',
    description:
      'Add a MongoDB service PMM can reach, or install a new replica set on a node that runs pmm-agent.',
    viewer: 'Ask an administrator to add a MongoDB service.',
  },
  clusters: { tableLabel: 'MongoDB clusters' },
  backups: {
    emptyTitle: 'No MongoDB backups yet',
    emptyDescription:
      'Set up a scheduled backup to keep a copy of each cluster you can restore.',
    noServices: 'Backups start once a MongoDB service is monitored.',
  },
  dashboards: { all: 'All MongoDB dashboards' },
};

const capitalize = (text: string) =>
  text.charAt(0).toUpperCase() + text.slice(1);

/** "a", "a or b", "a, b or c" */
const orList = (items: string[]) =>
  items.length < 2
    ? items.join('')
    : `${items.slice(0, -1).join(', ')} or ${items[items.length - 1]}`;

export const Messages = {
  noEnvironment: 'No environment',
  types: {
    sharded: 'Sharded cluster',
    replicaSet: 'Replica set',
    standalone: 'Standalone',
    routers: 'Routers',
    configServers: 'Config servers',
    shard: 'Shard',
    primary: 'Primary',
    secondary: 'Secondary',
    arbiter: 'Arbiter',
    router: 'Router',
    member: 'Member',
  },
  /** Type words for member states that serve nothing; any other state reads "Member". */
  memberStates: {
    STARTUP: 'Starting up',
    STARTUP2: 'Initial sync',
    RECOVERING: 'Recovering',
    ROLLBACK: 'Rolling back',
    REMOVED: 'Removed',
    UNKNOWN: 'Unknown state',
  } as Partial<Record<string, string>>,
  /** The same states inside a sentence: "orders-cfg-m3 has been recovering for 14m". */
  memberStatePhrases: {
    STARTUP: 'starting up',
    STARTUP2: 'in initial sync',
    RECOVERING: 'recovering',
    ROLLBACK: 'rolling back',
    REMOVED: 'removed from its set',
    UNKNOWN: 'in an unknown state',
  } as Partial<Record<string, string>>,
  memberStatePhrase: (state: string) => `in state ${state}`,
  routersName: 'Routers (mongos)',
  columns: {
    lag: {
      header: 'Replication lag',
      description: 'How far each secondary is behind its primary.',
    },
    oplogWindow: {
      header: 'Oplog window',
      description:
        'How much recent history the primary keeps for secondaries to catch up with.',
    },
  },
  /** What holds, beneath a health chip; the name of the row's own set or member is left out. */
  conditions: {
    upgrading: (member: number, total: number) =>
      `Rolling upgrade · member ${member} of ${total}`,
    allDown: (total: number, set?: string) =>
      set ? `${set}: all ${total} down` : `All ${total} members down`,
    noPrimary: (down: number, total: number, set?: string) =>
      [
        set ? `${set} has no primary` : 'No primary',
        down ? `${down} of ${total} down` : undefined,
      ]
        .filter(Boolean)
        .join(' · '),
    quorumLost: (down: number, total: number, set?: string) =>
      `${set ? `${set} lost its quorum` : 'Quorum lost'} · ${down} of ${total} down`,
    noData: (total: number, set?: string) =>
      set ? `No data from ${set}` : `No data from any of ${total} members`,
    memberDown: (member: string) => `${member} down`,
    memberNoData: (member: string) => `No data from ${member}`,
    /** "orders-cfg-m3 recovering for 14m", or "Recovering for 14m" on its own row. */
    memberState: (phrase: string, member?: string, duration?: string) =>
      `${member ? `${member} ${phrase}` : capitalize(phrase)}${duration ? ` for ${duration}` : ''}`,
    drift: (set?: string) =>
      set ? `${set}: mixed versions` : 'Mixed versions',
    oplogShort: (window: string, threshold: string, member?: string) =>
      `${member ? `${member} oplog window` : 'Oplog window'} ${window}, under ${threshold}`,
    behind: (lag: string, member?: string) =>
      member ? `${member} ${lag} behind` : `${lag} behind`,
  },
  /** The rule each condition breaks, for the tooltip. */
  rules: {
    upgrading:
      'Upgrading while Operations runs an upgrade on the cluster, whatever else holds',
    allDown: 'Down when every member is down',
    noPrimary:
      "Down when members answer but none is primary, so the set can't take writes",
    quorumLost:
      'Down when half the members or more are down, so no majority can keep a primary',
    memberLost: 'Down when a member rolls back or is removed from its set',
    noData: (stale: string) =>
      `No data when PMM gets nothing from any member for ${stale}`,
    memberDown:
      'Degraded when a member is down while the set keeps its primary and majority',
    memberNoData: (stale: string) =>
      `Degraded when PMM gets nothing from a member for ${stale}`,
    memberState: (grace: string) =>
      `Degraded when a member stays in STARTUP, STARTUP2, RECOVERING or UNKNOWN past ${grace}, or in any state but PRIMARY, SECONDARY or ARBITER`,
    drift: 'Degraded when members of one set run different versions',
    oplogShort: (threshold: string) =>
      `Degraded when the primary's oplog window is under ${threshold} once the oplog is 90% full`,
    behind: (threshold: string) =>
      `Degraded when a secondary is ${threshold} or more behind, sustained for 2m`,
  },
  measures: {
    primaryLag: 'A primary has no replication lag',
    arbiterLag: 'An arbiter holds no data',
    noPrimaryLag: 'No primary to measure lag against',
    downMember: 'Not reported while the member is down',
    noDataMember: 'Not reported while PMM gets no data from the member',
    routerLag: 'Routers do not replicate',
    routerOplog: 'Routers keep no oplog',
    standaloneLag: 'A standalone does not replicate',
    standaloneOplog: 'A standalone keeps no oplog',
    noPrimaryOplog: 'No primary to read the oplog window from',
    lagRule: (threshold: string, lag: string) =>
      `Warning at ${threshold} sustained for 2m; this member is ${lag} behind`,
    worstLagRule: (member: string, lag: string) =>
      `The furthest behind is ${member}, ${lag} behind its primary`,
    oplogRule: (window: string, fill: number) =>
      `Warning below 24h once the oplog is 90% full; this window is ${window} at ${fill}% full`,
    tightestOplogRule: (set: string, window: string) =>
      `The tightest window is ${set}, ${window}`,
  },
  doors: {
    openSummary: 'Open summary',
    queryAnalytics: 'Query analytics for this cluster',
    queryAnalyticsService: 'Query analytics for this service',
    backupStatus: 'Backup status',
    manage: 'Manage in Operations',
    membersOf: (name: string) => `members of ${name}`,
    lagOf: (name: string) => `replication lag of ${name}`,
    oplogOf: (name: string) => `oplog window of ${name}`,
    instanceSummary: 'Open Instance summary',
    routerSummary: 'Open Router summary',
    replsetSummary: 'Open ReplSet summary',
    oplogDetails: 'Open Oplog details',
    backups: 'Open MongoDB backups',
    scheduledBackups: 'Open scheduled backups',
    alert: 'Alert',
    alerts: 'Alerts',
    openAlerts: 'Open Alerts',
    openQuery: 'Open in Query analytics',
    openRealtime: 'Open Real-time queries',
    openAutomations: 'Open Automations',
    fleet: 'Fleet in Operations',
    instancesOverview: 'Instances overview',
    allApps: 'All apps',
    setUpBackup: 'Set up a backup',
    turnOnBackups: 'Turn it on in Settings',
    backupDashboard: 'Backup status dashboard',
    mongodbBackups: 'MongoDB backups',
    clusters: 'Clusters',
    behindOnPatches: 'Clusters behind on patches',
    firingAlerts: 'Firing alerts',
    needsAttention: 'Needs attention',
    lastGoodBackup: 'Last good backup',
    realtime: 'Real-time queries',
    queryAnalyticsLink: 'Query analytics',
    forSubject: (name: string) => `for ${name}`,
    forInstall: (target: string) => `for the install on ${target}`,
    forScan: 'for the scan',
  },
  attention: {
    noPrimary: (rs: string) => `${rs} has no primary, it can't take writes`,
    membersDown: (down: number, total: number) =>
      `${down} of ${total} members down`,
    memberDown: (member: string) => `${member} is down`,
    memberNoData: (member: string) => `PMM gets no data from ${member}`,
    notServing: (member: string, phrase: string, duration?: string) =>
      duration
        ? `${member} has been ${phrase} for ${duration}, it can't serve reads`
        : `${member} is ${phrase}, it can't serve reads`,
    memberLost: (member: string, phrase: string) =>
      `${member} is ${phrase}, it holds none of the set's data`,
    versionDrift: (rs: string) =>
      `${rs} runs mixed versions, an upgrade may have stopped partway`,
    versionOn: (version: string, members: string[]) =>
      `${version} on ${members.join(', ')}`,
    backupFailed: (cluster: string, name: string) =>
      `${cluster} ${name} failed`,
    backupFallbackName: 'backup',
    pitrStopped: (cluster: string) =>
      `${cluster} point-in-time recovery stopped`,
    oplogShort: (rs: string, window: string) =>
      `${rs} oplog window is ${window} and the oplog is full`,
    behind: (member: string, lag: string) =>
      `${member} is ${lag} behind its primary`,
    querySlower: (fingerprint: string, ratio: string) =>
      `${fingerprint} · ${ratio}× slower than usual`,
    queryRunning: (operation: string, namespace: string, age: string) =>
      `${operation} on ${namespace} · running for ${age}`,
    installFailed: (target: string) => `Install of MongoDB on ${target} failed`,
    scanFailed: (nodes: number) =>
      nodes === 1 ? 'Scan failed on 1 node' : `Scan failed on ${nodes} nodes`,
    scanFailedNoCount: 'Scan failed',
    oplogFull: (fill: number) => `${fill}% full`,
    lagDetail: (sustained: string) => `${sustained} sustained for 2m`,
    alertFallback: 'Alert firing',
  },
  /** The header's Add MongoDB and More actions menus. */
  actions: {
    monitor: 'Monitor an existing MongoDB service',
    install: 'Install a new MongoDB cluster',
    manageServices: 'Manage services',
    configureBackups: 'Configure backups',
    alertRules: 'Set up alert rules',
    advisors: 'Run Advisors checks',
    documentation: 'Documentation',
  },
  coverage: {
    memberDown: 'member down',
    lag: 'replication lag',
    oplog: 'oplog window',
    none: 'No alert rules cover MongoDB yet.',
    missing: (things: string[]) =>
      `No rule from the ${orList(things)} template yet.`,
    create: 'Create from templates',
    templates: (things: string[]) => `(${things.join(', ')})`,
    askAdmin: (things: string[]) =>
      `An administrator can create them from templates (${things.join(', ')}).`,
    askAdminOne: 'An administrator can create it from a template.',
  },
  where: (parts: Array<string | undefined>) =>
    parts.filter(Boolean).join(' · '),
  path: (parts: Array<string | undefined>) => parts.filter(Boolean).join(' › '),
  queries: {
    slower: (ratio: string) => `${ratio}× slower than usual`,
    slowerDetail: (avg: string, usual: string, calls: string) =>
      `${avg} avg · usual ${usual} · ${calls} calls in 1h`,
    running: (age: string) => `running for ${age}`,
    unknownOperation: 'query',
    unknownNamespace: 'an unknown collection',
  },
  backups: {
    pbmHint: (clusters: string[]) =>
      `PBM is not configured on ${clusters.join(', ')}`,
  },
  dashboards: {
    instancesOverview: 'Instances overview',
    instanceSummary: 'Instance summary',
    replsetSummary: 'ReplSet summary',
    oplogDetails: 'Oplog details',
    clusterSummary: 'Sharded cluster summary',
    routerSummary: 'Router summary',
    backupStatus: 'Backup status',
    collectionsOverview: 'Collections overview',
    unusedIndexes: 'Unused indexes',
  },
  apps: {
    operations: 'Operations for MongoDB',
    operationsUp: (up: number, total: number) =>
      `${up} of ${total} services up`,
    lastScan: (age: string) => `last scan ${age}`,
    noScanYet: 'no scan yet',
    operationsEmpty: 'No MongoDB yet. Install one from Nodes.',
    backups: 'MongoDB backups',
    schedules: (count: number) =>
      count === 1 ? '1 schedule' : `${count} schedules`,
    lastGood: (age: string) => `last good backup ${age}`,
    noGoodBackup: 'no good backup yet',
    failedRecently: (count: number) => `${count} failed in the last 24h`,
    backupsOff: 'Backup management is off',
    backupsNotSetUp: 'No backups set up yet',
    backupsUnknown: 'Backup status unknown',
    diagnostics: 'Diagnostics',
    diagnosticsSummary:
      'Diagnostic checks for MongoDB replication and performance',
    pages: {
      fleet: 'Fleet',
      nodes: 'Nodes',
      automations: 'Automations',
      inventory: 'Inventory',
      scheduled: 'Scheduled',
      restores: 'Restores',
      locations: 'Storage locations',
      diagnostics: 'Open Diagnostics',
    },
  },
};
