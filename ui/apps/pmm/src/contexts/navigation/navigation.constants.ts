import { GRAFANA_SUB_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import { NavItem } from 'lib/types';

export const PMM_NAV_OS: NavItem = {
  id: 'system',
  text: 'Operating system',
  icon: 'percona-system',
  url: `${GRAFANA_SUB_PATH}/d/node-instance-overview/nodes-overview`,
  children: [
    {
      id: 'node-overview',
      text: 'Overview',
      icon: 'percona-nav-overview',
      url: `${GRAFANA_SUB_PATH}/d/node-instance-overview/nodes-overview`,
    },
    {
      id: 'node-summary',
      text: 'Summary',
      icon: 'percona-nav-summary',
      url: `${GRAFANA_SUB_PATH}/d/node-instance-summary/node-summary`,
    },
    {
      id: 'cpu-utilization',
      text: 'CPU utilization',
      icon: 'percona-cpu',
      url: `${GRAFANA_SUB_PATH}/d/node-cpu/cpu-utilization-details`,
    },
    {
      id: 'disk',
      text: 'Disk',
      icon: 'percona-disk',
      url: `${GRAFANA_SUB_PATH}/d/node-disk/disk-details`,
    },
    {
      id: 'memory',
      text: 'Memory',
      icon: 'percona-memory',
      url: `${GRAFANA_SUB_PATH}/d/node-memory/memory-details`,
    },
    {
      id: 'network',
      text: 'Network',
      icon: 'percona-network',
      url: `${GRAFANA_SUB_PATH}/d/node-network/network-details`,
    },
    {
      id: 'temperature',
      text: 'Temperature',
      icon: 'percona-temperature',
      url: `${GRAFANA_SUB_PATH}/d/node-temp/node-temperature-details`,
    },
    {
      id: 'numa',
      text: 'NUMA',
      icon: 'percona-cluster-network',
      url: `${GRAFANA_SUB_PATH}/d/node-memory-numa/numa-details`,
    },
    {
      id: 'processes',
      text: 'Processes',
      icon: 'percona-process',
      url: `${GRAFANA_SUB_PATH}/d/node-cpu-process/processes-details`,
    },
  ],
};

export const PMM_NAV_MYSQL: NavItem = {
  id: 'mysql',
  text: 'MySQL',
  icon: 'percona-database-mysql',
  url: `${GRAFANA_SUB_PATH}/d/mysql-instance-overview/mysql-instances-overview`,
  children: [
    {
      id: 'mysql-overview',
      text: 'Overview',
      icon: 'percona-nav-overview',
      url: `${GRAFANA_SUB_PATH}/d/mysql-instance-overview/mysql-instances-overview`,
    },
    {
      id: 'mysql-summary',
      text: 'Summary',
      icon: 'percona-nav-summary',
      url: `${GRAFANA_SUB_PATH}/d/mysql-instance-summary/mysql-instance-summary`,
    },
    {
      id: 'mysql-ha',
      text: 'High availability',
      icon: 'percona-cluster',
      url: `${GRAFANA_SUB_PATH}/d/mysql-group-replicaset-summary`,
      children: [
        {
          id: 'mysql-group-replication-summary',
          text: 'Group replication summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/mysql-group-replicaset-summary/mysql-group-replication-summary`,
        },
        {
          id: 'mysql-replication-summary',
          text: 'Replication summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/mysql-replicaset-summary/mysql-replication-summary`,
        },
        {
          id: 'pxc-cluster-summary',
          text: 'PXC/Galera cluster summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/pxc-cluster-summary/pxc-galera-cluster-summary`,
        },
        {
          id: 'pxc-node-summary',
          text: 'PXC/Galera node summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/pxc-node-summary/pxc-galera-node-summary`,
        },
        {
          id: 'pxc-nodes-compare',
          text: 'PXC/Galera nodes compare',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/pxc-nodes-compare/pxc-galera-nodes-compare`,
        },
      ],
    },
    {
      id: 'mysql-command-handler-counters-compare',
      text: 'Command/Handler counters compare',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-commandhandler-compare/mysql-command-handler-counters-compare`,
    },
    {
      id: 'mysql-innodb-details',
      text: 'InnoDB details',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-innodb/mysql-innodb-details`,
    },
    {
      id: 'mysql-innodb-compression-details',
      text: 'InnoDB compression',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-innodb-compression/mysql-innodb-compression-details`,
    },
    {
      id: 'mysql-performance-schema-details',
      text: 'Performance schema',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-performance-schema/mysql-performance-schema-details`,
    },
    {
      id: 'mysql-query-response-time-details',
      text: 'Query response time',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-queryresponsetime/mysql-query-response-time-details`,
    },
    {
      id: 'mysql-table-details',
      text: 'Table details',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-table/mysql-table-details`,
    },
    {
      id: 'mysql-tokudb-details',
      text: 'TokuDB details',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mysql-tokudb/mysql-tokudb-details`,
    },
  ],
};

export const PMM_NAV_MONGO: NavItem = {
  id: 'mongo',
  text: 'MongoDB',
  icon: 'percona-database-mongodb',
  url: `${GRAFANA_SUB_PATH}/d/mongodb-instance-overview/mongodb-instances-overview`,
  children: [
    {
      id: 'mongo-overview',
      text: 'Overview',
      icon: 'percona-nav-overview',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-instance-overview/mongodb-instances-overview`,
    },
    {
      id: 'mongo-summary',
      text: 'Summary',
      icon: 'percona-nav-summary',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-instance-summary/mongodb-instance-summary`,
    },
    {
      id: 'mongo-ha',
      text: 'High availability',
      icon: 'percona-cluster',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-cluster-summary`,
      children: [
        {
          id: 'mongo-cluster-summary',
          text: 'Cluster summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary`,
        },
        {
          id: 'mongo-rplset-summary',
          text: 'ReplSet summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/mongodb-replicaset-summary/mongodb-replset-summary`,
        },
        {
          id: 'mongo-router-summary',
          text: 'Router summary',
          icon: 'percona-cluster',
          url: `${GRAFANA_SUB_PATH}/d/mongodb-router-summary/mongodb-router-summary`,
        },
      ],
    },
    {
      id: 'mongo-memory-details',
      text: 'InMemory',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-inmemory/mongodb-inmemory-details`,
    },
    {
      id: 'mondo-wiredtiger-details',
      text: 'WiredTiger',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-wiredtiger/mongodb-wiredtiger-details`,
    },
    {
      id: 'mongo-collections-overview',
      text: 'Collections',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-collections-overview/mongodb-collections-overview`,
    },
    {
      id: 'mongo-oplog-details',
      text: 'Oplog',
      icon: 'sitemap',
      url: `${GRAFANA_SUB_PATH}/d/mongodb-oplog-details/mongodb-oplog-details`,
    },
  ],
};

export const PMM_NAV_POSTGRE: NavItem = {
  id: 'postgre',
  text: 'PostgreSQL',
  icon: 'percona-database-postgresql',
  url: `${GRAFANA_SUB_PATH}/d/postgresql-instance-overview/postgresql-instances-overview`,
  children: [
    {
      id: 'postgre-overwiew',
      text: 'Overview',
      icon: 'percona-nav-overview',
      url: `${GRAFANA_SUB_PATH}/d/postgresql-instance-overview/postgresql-instances-overview`,
    },
    {
      id: 'postgre-summary',
      text: 'Summary',
      icon: 'percona-nav-summary',
      url: `${GRAFANA_SUB_PATH}/d/postgresql-instance-summary/postgresql-instance-summary`,
    },
  ],
};

export const PMM_NAV_PROXYSQL: NavItem = {
  id: 'proxysql',
  text: 'ProxySQL',
  icon: 'percona-database-proxysql',
  url: `${GRAFANA_SUB_PATH}/d/proxysql-instance-summary/proxysql-instance-summary`,
};

export const PMM_NAV_HAPROXY: NavItem = {
  id: 'haproxy',
  text: 'HAProxy',
  icon: 'percona-database-haproxy',
  url: `${GRAFANA_SUB_PATH}/d/haproxy-instance-summary/haproxy-instance-summary`,
};

export const PMM_NAV_QAN: NavItem = {
  id: 'qan',
  text: 'Query Analytics (QAN)',
  icon: 'qan-logo',
  url: `${GRAFANA_SUB_PATH}/d/pmm-qan/pmm-query-analytics`,
};

export const PMM_BACKUP_PAGE: NavItem = {
  id: 'backup',
  icon: 'history',
  text: 'Backup',
  url: `${GRAFANA_SUB_PATH}/backup`,
  children: [
    {
      id: 'backup-inventory',
      text: 'All Backups',
      url: `${GRAFANA_SUB_PATH}/backup/inventory`,
    },
    {
      id: 'scheduled-backups',
      text: 'Scheduled Backup Jobs',
      url: `${GRAFANA_SUB_PATH}/backup/scheduled`,
    },
    {
      id: 'restore-history',
      text: 'Restores',
      url: `${GRAFANA_SUB_PATH}/backup/restore`,
    },
    {
      id: 'storage-locations',
      text: 'Storage Locations',
      url: `${GRAFANA_SUB_PATH}/backup/locations`,
    },
  ],
};

export const PMM_ALERTING_CREATE_ALERT_TEMPLATE: NavItem = {
  id: 'integrated-alerting-new-from-template',
  text: 'Create alert rule from template',
  icon: 'brackets-curly',
  url: `${GRAFANA_SUB_PATH}/alerting/new-from-template`,
};

export const PMM_ALERTING_FIRED_ALERTS: NavItem = {
  id: 'integrated-alerting-alerts',
  text: 'Fired alerts',
  icon: 'info-circle',
  url: `${GRAFANA_SUB_PATH}/alerting/alerts`,
};

export const PMM_ALERTING_RULE_TEMPLATES: NavItem = {
  id: 'integrated-alerting-templates',
  text: 'Alert rule templates',
  icon: 'brackets-curly',
  url: `${GRAFANA_SUB_PATH}/alerting/alert-rule-templates`,
};

export const PMM_ALERTING_PERCONA_ALERTS: NavItem[] = [
  PMM_ALERTING_FIRED_ALERTS,
  PMM_ALERTING_RULE_TEMPLATES,
  PMM_ALERTING_CREATE_ALERT_TEMPLATE,
];

export const PMM_SERVICES_PAGE: NavItem = {
  id: 'inventory-services',
  text: 'Services',
  url: `${GRAFANA_SUB_PATH}/inventory/services`,
};

export const PMM_NODES_PAGE: NavItem = {
  id: 'inventory-nodes',
  text: 'Nodes',
  url: `${GRAFANA_SUB_PATH}/inventory/nodes`,
};

export const PMM_INVENTORY_PAGE: NavItem = {
  id: 'inventory',
  icon: 'server-network',
  text: 'Inventory',
  url: `${GRAFANA_SUB_PATH}/inventory`,
  children: [PMM_SERVICES_PAGE, PMM_NODES_PAGE],
};

export const PMM_SETTINGS = {
  id: 'settings',
  icon: 'percona-setting',
  text: 'Settings',
  url: `${GRAFANA_SUB_PATH}/settings`,
  subTitle: 'Percona Settings',
  children: [
    {
      id: 'settings-metrics-resolution',
      text: 'Metrics Resolution',
      url: `${GRAFANA_SUB_PATH}/settings/metrics-resolution`,
    },
    {
      id: 'settings-advanced',
      text: 'Advanced Settings',
      url: `${GRAFANA_SUB_PATH}/settings/advanced-settings`,
    },
    {
      id: 'settings-ssh',
      text: 'SSH Key',
      url: `${GRAFANA_SUB_PATH}/settings/ssh-key`,
    },
    {
      id: 'settings-percona-platform',
      text: 'Percona Platform',
      url: `${GRAFANA_SUB_PATH}/settings/percona-platform`,
    },
  ],
};

export const PMM_UPDATES = {
  id: 'updates',
  text: 'Updates',
  url: PMM_NEW_NAV_PATH + '/updates',
};

export const PMM_ADD_INSTANCE_PAGE: NavItem = {
  id: 'add-instance',
  url: `${GRAFANA_SUB_PATH}/add-instance`,
  icon: 'plus',
  text: 'Add Service',
};

export const PMM_CONFIGURATION: NavItem = {
  id: 'pmmcfg',
  text: 'Configuration',
  icon: 'percona-nav-logo',
  url: `${GRAFANA_SUB_PATH}/inventory`,
  children: [
    PMM_ADD_INSTANCE_PAGE,
    PMM_INVENTORY_PAGE,
    PMM_SETTINGS,
    PMM_UPDATES,
  ],
};

const PMM_ADVISORS: NavItem = {
  id: `advisors`,
  icon: 'percona-database-checks',
  text: 'Advisors',
  url: `${GRAFANA_SUB_PATH}/advisors`,
  children: [
    {
      id: 'advisors-insights',
      text: 'Advisor Insights',
      url: `${GRAFANA_SUB_PATH}/advisors/insights`,
    },
  ],
};

export const INITIAL_ITEMS: NavItem[] = [
  {
    id: 'home-page',
    text: 'Home page',
    url: PMM_NEW_NAV_PATH + '/graph/d/pmm-home',
  },
  PMM_NAV_OS,
  PMM_NAV_MYSQL,
  PMM_NAV_MONGO,
  PMM_NAV_POSTGRE,
  PMM_NAV_PROXYSQL,
  PMM_NAV_HAPROXY,
  PMM_NAV_QAN,
  PMM_ADVISORS,
  PMM_BACKUP_PAGE,
  PMM_CONFIGURATION,
  {
    id: 'alerts',
    text: 'Alerts',
    url: PMM_NEW_NAV_PATH + '/graph/alerting/alerts',
  },
  {
    id: 'help',
    text: 'Help',
    url: PMM_NEW_NAV_PATH + '/help',
  },
];
