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
  health: {
    noPrimary: "No primary: it can't take writes",
    allDown: 'Every member is down',
    noDataSince: (time: string) => `No data since ${time}`,
    downSince: (time: string) => `Down since ${time}`,
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
    lagRule: (level: 'warning' | 'error', lag: string) =>
      level === 'error'
        ? `Error at 10m sustained, or at half the primary's oplog window; this member is ${lag} behind`
        : `Warning above 30s sustained for 2m; this member is ${lag} behind`,
    worstLagRule: (member: string, lag: string) =>
      `The furthest behind is ${member}, ${lag} behind its primary`,
    oplogRule: (level: 'warning' | 'error', window: string, fill: number) =>
      level === 'error'
        ? `Error below 1h once the oplog is 90% full; this window is ${window} at ${fill}% full`
        : `Warning below 24h once the oplog is 90% full; this window is ${window} at ${fill}% full`,
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
    addService: 'Add MongoDB service',
    install: 'Install MongoDB',
    setUpBackup: 'Set up a backup',
    turnOnBackups: 'Turn it on in Settings',
    backupDashboard: 'Backup status dashboard',
    mongodbBackups: 'MongoDB backups',
    clusters: 'Clusters',
    needsAttention: 'Needs attention',
    lastGoodBackup: 'Last good backup',
    realtime: 'Real-time queries',
    queryAnalyticsLink: 'Query analytics',
    forSubject: (name: string) => `for ${name}`,
  },
  attention: {
    noPrimary: (rs: string) => `${rs} has no primary, it can't take writes`,
    membersDown: (down: number, total: number) =>
      `${down} of ${total} members down`,
    memberDown: (member: string) => `${member} is down`,
    memberNoData: (member: string) => `PMM gets no data from ${member}`,
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
