import type { HubMessages } from 'hubs/template/hub.types';

export const MONGODB_HUB_MESSAGES: HubMessages = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
  dashboards: { all: 'All MongoDB dashboards' },
};

/** "a", "a or b", "a, b or c" */
const orList = (items: string[]) =>
  items.length < 2
    ? items.join('')
    : `${items.slice(0, -1).join(', ')} or ${items[items.length - 1]}`;

export const Messages = {
  noEnvironment: 'No environment',
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
  doors: {
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
  },
  destinations: {
    replsetSummary: 'ReplSet summary',
    shardedSummary: 'Sharded cluster summary',
    instancesOverview: 'Instances overview',
    backups: 'MongoDB backups',
    scheduledBackups: 'Scheduled backups',
    inventory: 'Inventory',
    realtime: 'Real-time queries',
    fleet: 'Fleet',
    affectedClusters: 'Affected clusters',
    clusters: 'Clusters',
    queryAnalytics: 'Query analytics',
    planInFleet: 'Plan in Fleet',
    turnOnPitr: 'Turn on in Scheduled backups',
    setUpBackups: 'Set up backups',
    advisors: 'Advisors',
    alertTemplates: 'Alert templates',
    howToEnable: 'How to enable',
    upgradeGuide: 'Upgrade guide',
    turnOnOperations: 'Turn on Operations',
    /** Where each one sits in the sidebar. */
    places: {
      apps: 'Apps',
      operations: 'Operations for MongoDB',
      dashboards: 'All dashboards',
      folder: 'MongoDB',
      replsetSummary: 'MongoDB ReplSet Summary',
      shardedSummary: 'MongoDB Sharded Cluster Summary',
      instancesOverview: 'MongoDB Instances Overview',
      clusters: 'Clusters',
      clustersFiltered: 'Clusters, filtered',
      newBackup: 'New backup',
      alerts: 'Alerts',
      templates: 'Templates',
      settings: 'Settings',
      advancedSettings: 'Advanced settings',
      docs: 'Percona documentation',
      queryCollection: 'MongoDB query source',
      upgrade60: 'Upgrade from 6.0 to 7.0',
    },
  },
  /** Where a Needs attention row comes from; kept with the row, never shown. */
  sources: {
    health: 'PMM health monitoring',
    alerts: 'Alert rules',
    agent: 'Agent connection',
    backups: 'Backups',
    realtime: 'Real-time queries',
    qan: 'Query Analytics',
    operations: 'Operations for MongoDB',
    advisors: 'Advisors',
    versions: 'Version check',
  },
  /** Live recommendations; the sample scenarios write theirs out. */
  recommendations: {
    allClusters: 'All clusters',
    noRules: 'No alert rules cover MongoDB',
    noRulesImpact: 'PMM won’t notify anyone if a cluster goes down.',
    ruleGaps: (words: string[]) => {
      const text = `${words.length > 1 ? `${words.slice(0, -1).join(', ')} and ${words[words.length - 1]}` : words[0]} ${words.length > 1 ? 'have' : 'has'} no alert rules`;

      return text.charAt(0).toUpperCase() + text.slice(1);
    },
    ruleGapsImpact:
      'You’ll only see these problems on this page, not as a notification.',
    noBackups: (count: number) =>
      count === 1
        ? '1 cluster has no backups'
        : `${count} clusters have no backups`,
    noBackupsImpact: 'If data is lost, there’s nothing to restore from.',
    clusters: (names: string[], shown = 3) =>
      names.length <= shown + 1
        ? names.join(', ')
        : `${names.slice(0, shown).join(', ')} and ${names.length - shown} more`,
  },
};
