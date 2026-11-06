export const Messages = {
  title: 'Amazon RDS',
  alerting: 'Alerting',
  advisors: 'Advisors',
  loading: 'Loading Amazon RDS instances',
  fetchError: 'Could not load the Amazon RDS instances.',
  noData: 'No Amazon RDS instances found',
  noMatch: 'No groups match the filters',
  clearFilters: 'Clear filters',
  search: 'Search',
  searchLabel: 'Search DB identifiers, types, instances, accounts',
  allRegions: 'All regions',
  allAccounts: 'All accounts',
  allEngines: 'All engines',
  allTypes: 'All types',
  allGroups: 'All DB identifiers',
  shownGroups: (shown: number, total: number) => `${shown} of ${total} groups`,
  selectedValues: (first: string, more: number) =>
    more ? `${first} +${more}` : first,
  expandAll: 'Expand all',
  collapseAll: 'Collapse all',
  expandGroup: (name: string) => `Expand ${name}`,
  collapseGroup: (name: string) => `Collapse ${name}`,
  writers: (writers: number) =>
    writers === 1 ? '1 writer' : `${writers} writers`,
  noWriter: 'no writer',
  writerUnknown: 'unknown',
  noAlerts: 'None',
  firingAlerts: (count: number) =>
    count === 1 ? '1 firing alert' : `${count} firing alerts`,
  firingFor: (age: string) => `firing for ${age}`,
  showAllAlerts: 'Show all in Alert groups',
  instances: (connected: number, monitored: number) =>
    `${connected} / ${monitored}`,
  up: 'Up',
  down: 'Down',
  storageUsed: (value: string) => `${value} used`,
  storageFree: (value: string) => `${value} free`,
  storageLowestFree: (value: string) => `${value} free (min)`,
  groupTypes: {
    'aurora-cluster': 'Aurora cluster',
    'multi-az-cluster': 'Multi-AZ DB cluster',
    replication: 'Replication group',
    instance: 'Instance',
    unavailable: 'Unavailable',
  },
  groupTypeTitles: {
    'aurora-cluster': 'Aurora DB cluster',
    'multi-az-cluster': 'Multi-AZ DB cluster',
    replication:
      'Primary instance and its read replicas, named after the primary',
    instance: 'Instance without a cluster or read replicas',
    unavailable:
      'PMM has no AWS details for these instances. If they are monitored by another pmm-client, upgrade it.',
  },
  links: {
    mysql: 'MySQL',
    postgresql: 'PostgreSQL',
    os: 'OS',
    details: (serviceName: string) =>
      `MySQL Amazon Aurora Details for ${serviceName}`,
    postgresqlDetails: (serviceName: string) =>
      `PostgreSQL Amazon Aurora Details for ${serviceName}`,
    mysqlSummary: (serviceName: string) =>
      `MySQL Instance Summary for ${serviceName}`,
    postgresqlSummary: (serviceName: string) =>
      `PostgreSQL Instance Summary for ${serviceName}`,
    nodeSummary: (nodeName: string) => `Node Summary for ${nodeName}`,
  },
  columns: {
    region: 'Region',
    regionAccount: 'Region / Account',
    awsAccount: 'AWS account',
    identifier: 'DB identifier',
    type: 'Type',
    typeEngine: 'Type / Engine',
    engine: 'Engine',
    alerts: 'Alerts',
    alertsDescription:
      'Firing alerts of the instances, except silenced ones. An alert counts when its service_name or node_name label names an instance.',
    instancesWriters: 'Instances / Writers',
    instancesWritersDescription:
      'Instances whose exporter is connected, out of the monitored instances, and the number of writer instances.',
    throughput: 'Throughput',
    throughputDescription:
      'MySQL and MariaDB: queries per second. PostgreSQL: transactions (commits and rollbacks) per second.',
    storage: 'Storage',
    storageDescription:
      'Aurora: cluster volume used. Other engines: free storage space, the lowest instance for a group.',
    service: 'Service',
    links: 'Dashboards',
    role: 'Role',
    status: 'Status',
  },
};
