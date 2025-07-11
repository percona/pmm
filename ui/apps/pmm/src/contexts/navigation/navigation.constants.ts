import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import { NavItem } from 'lib/types';

export const NAV_DIVIDERS = {
  home: {
    id: 'home-divider',
    isDivider: true,
  },
  inventory: {
    id: 'inventory-divider',
    isDivider: true,
  },
  backups: {
    id: 'backups-divider',
    isDivider: true,
  },
};

export const NAV_HOME_PAGE: NavItem = {
  id: 'home-page',
  icon: 'home',
  text: 'Home page',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-home`,
  children: [
    {
      id: 'home-page-dashboard',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-home/home-dashboard`,
      hidden: true,
    },
  ],
};

//
// MySQL dashboards
//
export const NAV_MYSQL: NavItem = {
  id: 'mysql',
  text: 'MySQL',
  icon: 'mysql',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-overview/mysql-instances-overview`,
  children: [
    {
      id: 'mysql-overview',
      icon: 'overview',
      text: 'Overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-overview/mysql-instances-overview`,
    },
    {
      id: 'mysql-summary',
      icon: 'summary',
      text: 'Summary',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-summary/mysql-instance-summary`,
    },
    {
      id: 'mysql-high-availability',
      icon: 'high-availability',
      text: 'High Availability',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-group-replicaset-summary`,
      children: [
        {
          id: 'mysql-group-replication-summary',
          text: 'Group replication',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-group-replicaset-summary/mysql-group-replication-summary`,
        },
        {
          id: 'mysql-replication-summary',
          text: 'Replication',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-replicaset-summary/mysql-replication-summary`,
        },
        {
          id: 'pxc-cluster-summary',
          text: 'PXC/Galera cluster',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pxc-cluster-summary/pxc-galera-cluster-summary`,
        },
        {
          id: 'pxc-node-summary',
          text: 'PXC/Galera node',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pxc-node-summary/pxc-galera-node-summary`,
        },
        {
          id: 'pxc-nodes-compare',
          text: 'PXC/Galera nodes',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pxc-nodes-compare/pxc-galera-nodes-compare`,
        },
      ],
    },
    {
      id: 'mysql-command-handler-counters-compare',
      text: 'Command/Handler counters compare',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-commandhandler-compare/mysql-command-handler-counters-compare`,
    },
    {
      id: 'mysql-innodb-details',
      text: 'InnoDB details',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-innodb/mysql-innodb-details`,
    },
    {
      id: 'mysql-innodb-compression-details',
      text: 'InnoDB compression',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-innodb-compression/mysql-innodb-compression-details`,
    },
    {
      id: 'mysql-performance-schema-details',
      text: 'Performance schema',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-performance-schema/mysql-performance-schema-details`,
    },
    {
      id: 'mysql-query-response-time-details',
      text: 'Query response time',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-queryresponsetime/mysql-query-response-time-details`,
    },
    {
      id: 'mysql-table-details',
      text: 'Table details',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-table/mysql-table-details`,
    },
    {
      id: 'mysql-tokudb-details',
      text: 'TokuDB details',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-tokudb/mysql-tokudb-details`,
    },
  ],
};

//
// MongoDB dashboards
//
export const NAV_MONGO: NavItem = {
  id: 'mongo',
  icon: 'mongo',
  text: 'MongoDB',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-overview/mongodb-instances-overview`,
  children: [
    {
      id: 'mongo-overview',
      icon: 'overview',
      text: 'Overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-overview/mongodb-instances-overview`,
    },
    {
      id: 'mongo-summary',
      icon: 'summary',
      text: 'Summary',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-summary/mongodb-instance-summary`,
    },
    {
      id: 'mongo-high-availability',
      icon: 'high-availability',
      text: 'High availability',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-cluster-summary`,
      children: [
        {
          id: 'mongo-cluster-summary',
          text: 'Cluster',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary`,
        },
        {
          id: 'mongo-rplset-summary',
          text: 'ReplSet',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-replicaset-summary/mongodb-replset-summary`,
        },
        {
          id: 'mongo-router-summary',
          text: 'Router',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-router-summary/mongodb-router-summary`,
        },
      ],
    },
    {
      id: 'mongo-memory-details',
      text: 'InMemory',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-inmemory/mongodb-inmemory-details`,
    },
    {
      id: 'mondo-wiredtiger-details',
      text: 'WiredTiger',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-wiredtiger/mongodb-wiredtiger-details`,
    },
    {
      id: 'mongo-collections-overview',
      text: 'Collections',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-collections-overview/mongodb-collections-overview`,
    },
    {
      id: 'mongo-oplog-details',
      text: 'Oplog',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-oplog-details/mongodb-oplog-details`,
    },
  ],
};

//
// PostgreSQL
//
export const NAV_POSTGRESQL: NavItem = {
  id: 'postgre',
  text: 'PostgreSQL',
  icon: 'postgresql',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-overview/postgresql-instances-overview`,
  children: [
    {
      id: 'postgresql-overwiew',
      text: 'Overview',
      icon: 'overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-overview/postgresql-instances-overview`,
    },
    {
      id: 'postgresql-summary',
      text: 'Summary',
      icon: 'summary',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-summary/postgresql-instance-summary`,
    },
    {
      id: 'postgresql-ha',
      text: 'High availability',
      icon: 'high-availability',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-replication-overview`,
      children: [
        {
          id: 'postgresql-replication',
          text: 'Replication',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-replication-overview/postgresql-replication-overview`,
        },
        {
          id: 'postgresql-patroni',
          text: 'Patroni',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-patroni-details/postgresql-patroni-details`,
        },
      ],
    },
    {
      id: 'postgresql-top-queries',
      text: 'Top queries',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-top-queries/postgresql-top-queries`,
    },
  ],
};

//
// OS dashboards
//
export const NAV_OS: NavItem = {
  id: 'system',
  icon: 'operating-system',
  text: 'Operating system',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-overview/nodes-overview`,
  children: [
    {
      id: 'node-overview',
      icon: 'overview',
      text: 'Overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-overview/nodes-overview`,
    },
    {
      id: 'node-summary',
      icon: 'summary',
      text: 'Summary',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-summary/node-summary`,
    },
    {
      id: 'cpu-utilization',
      text: 'CPU utilization',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-cpu/cpu-utilization-details`,
    },
    {
      id: 'disk',
      text: 'Disk',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-disk/disk-details`,
    },
    {
      id: 'memory',
      text: 'Memory',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-memory/memory-details`,
    },
    {
      id: 'network',
      text: 'Network',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-network/network-details`,
    },
    {
      id: 'temperature',
      text: 'Temperature',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-temp/node-temperature-details`,
    },
    {
      id: 'numa',
      text: 'NUMA',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-memory-numa/numa-details`,
    },
    {
      id: 'processes',
      text: 'Processes',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-cpu-process/processes-details`,
    },
  ],
};

//
// HAProxy
//
export const NAV_HAPROXY: NavItem = {
  id: 'haproxy',
  icon: 'haproxy',
  text: 'HAProxy',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/haproxy-instance-summary/haproxy-instance-summary`,
};

//
// ProxySQL
//
export const NAV_PROXYSQL: NavItem = {
  id: 'proxysql',
  icon: 'proxysql',
  text: 'ProxySQL',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/proxysql-instance-summary/proxysql-instance-summary`,
};

//
// QAN
//
export const NAV_QAN: NavItem = {
  id: 'qan',
  icon: 'qan',
  text: 'Query Analytics (QAN)',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`,
};

//
// All Dashbaords
//
export const NAV_DASHBOARDS: NavItem = {
  id: 'dashboards',
  icon: 'dashboards',
  text: 'All dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards`,
};

export const NAV_DASHBOARDS_BROWSE: NavItem = {
  id: 'dashboards-browse',
  icon: 'browse-dashboards',
  text: 'Browse all dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards`,
};

export const NAV_DASHBOARDS_SHARED: NavItem = {
  id: 'dashboards-shared',
  text: 'Shared dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/public`,
};

export const NAV_DASHBOARDS_PLAYLISTS: NavItem = {
  id: 'dashboards-playlists',
  text: 'Playlists',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/playlists`,
};

export const NAV_DASHBOARDS_SNAPSHOTS: NavItem = {
  id: 'dashboards-snapshots',
  text: 'Snapshots',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/snapshots`,
};

export const NAV_DASHBOARDS_LIBRARY_PANELS = {
  id: 'dashboards-library-panels',
  text: 'Library panels',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/library-panels`,
};

//
// Explore
//
export const NAV_EXPLORE_METRICS: NavItem = {
  id: 'explore-metrics',
  text: 'Explore metrics',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/explore/metrics`,
};

export const NAV_EXPLORE_BUILDER: NavItem = {
  id: 'explore-promsql-builder',
  text: 'PromSQL builder',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/explore`,
};

export const NAV_EXPLORE: NavItem = {
  id: 'explore',
  icon: 'explore',
  text: 'Explore',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/explore`,
};

//
// Alerting
//
export const NAV_ALERTS_TEMPLATES = {
  id: 'alerts-templates',
  text: 'Percona Alert Templates',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/alert-rule-templates`,
};

export const NAV_ALERTS_FIRED: NavItem = {
  id: 'alerts-fired',
  text: 'Fired Alerts',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/alerts`,
};

export const NAV_ALERTS_CONTACT_POINTS: NavItem = {
  id: 'alerts-contact-points',
  text: 'Contact points',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/notifications`,
};

export const NAV_ALERTS_NOTIFICATION_POLICIES: NavItem = {
  id: 'alerts-policies',
  text: 'Notification policies',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/routes`,
};

export const NAV_ALERTS_SETTINGS: NavItem = {
  id: 'alerts-settings',
  text: 'Alert Settings',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/admin`,
};

export const NAV_ALERTS: NavItem = {
  id: 'alerts',
  icon: 'alerts',
  text: 'Alerts',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/alerts`,
};

//
// Percona Intelligence
//
export const NAV_INTELLIGENCE_TEMPLATES = {
  ...NAV_ALERTS_TEMPLATES,
  id: 'intelligence-alerts-templates',
};

export const NAV_INTELLIGENCE: NavItem = {
  id: 'percona-intelligence',
  icon: 'intelligence',
  text: 'Percona Intelligence',
  url: NAV_INTELLIGENCE_TEMPLATES.url,
};

export const NAV_ADVISORS = {
  id: 'advisors',
  text: 'Percona Advisors',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/advisors`,
};

export const NAV_ADVISORS_INSIGHTS = {
  id: 'advisors-insights',
  text: 'Advisor Insights',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/insights`,
};

//
// Inventory
//
export const NAV_INVENTORY: NavItem = {
  id: 'inventory',
  icon: 'inventory',
  text: 'Inventory',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory`,
  children: [
    {
      id: 'add-instance',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance`,
      text: 'Add Service',
      children: [
        {
          id: 'add-instance-form',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance/:type`,
          text: 'Add Service',
          hidden: true,
        },
      ],
    },
    {
      id: 'inventory-services',
      text: 'Services',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/services`,
      children: [
        {
          id: 'inventory-services-agents',
          text: 'Services / Agents',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/services/:serviceId/agents`,
          hidden: true,
        },
      ],
    },
    {
      id: 'inventory-nodes',
      text: 'Nodes',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/nodes`,
      children: [
        {
          id: 'inventory-nodes-agents',
          text: 'Nodes / Agents',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/nodes/:nodeId/agents`,
          hidden: true,
        },
      ],
    },
  ],
};

//
// Backups
//
export const NAV_BACKUPS: NavItem = {
  id: 'backups',
  icon: 'backups',
  text: 'Backups',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
  children: [
    {
      id: 'backup-inventory',
      text: 'All Backups',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
      children: [
        {
          id: 'backups-new',
          text: 'Create backup',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/new`,
          hidden: true,
        },
      ],
    },
    {
      id: 'scheduled-backups',
      text: 'Scheduled Backup Jobs',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/scheduled`,
    },
    {
      id: 'restore-history',
      text: 'Restores',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/restore`,
    },
    {
      id: 'storage-locations',
      text: 'Storage Locations',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/locations`,
    },
  ],
};

//
// Configuration
//
export const NAV_CONFIGURATION: NavItem = {
  id: 'configuration',
  icon: 'configuration',
  text: 'Configuration',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/settings`,
  children: [
    {
      id: 'configuration-settings',
      text: 'Settings',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/settings/advanced-settings`,
    },
    {
      id: 'updates',
      text: 'Updates',
      url: `${PMM_NEW_NAV_PATH}/updates`,
    },
    {
      id: 'org-management',
      text: 'Org. management',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/admin/orgs`,
      children: [
        {
          id: 'organizations',
          text: 'Organizations',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/admin/orgs`,
        },
        {
          id: 'stats-and-licenses',
          text: 'Stats and licenses',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/admin/upgrading`,
        },
        {
          id: 'default-preferences',
          text: 'Default preferences',
          url: `${PMM_NEW_NAV_GRAFANA_PATH}/org`,
        },
      ],
    },
  ],
};

//
// Users and Access
//
export const NAV_USERS_AND_ACCESS: NavItem = {
  id: 'users-and-access',
  icon: 'encrypted',
  text: 'Users and access',
  url: PMM_NEW_NAV_GRAFANA_PATH + '/admin/access',
  children: [
    {
      id: 'users',
      text: 'Users',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/admin/users',
    },
    {
      id: 'teams',
      text: 'Teams',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/org/teams',
    },
    {
      id: 'service-accounts',
      text: 'Services accounts',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/org/serviceaccounts',
    },
  ],
};

//
// Account
//
export const NAV_ACCOUNT: NavItem = {
  id: 'account',
  icon: 'account',
  text: 'Account',
  url: PMM_NEW_NAV_GRAFANA_PATH + '/profile',
  children: [
    {
      id: 'profile',
      text: 'Profile',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/profile',
    },
    {
      id: 'notification-history',
      text: 'Notification history',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/profile/notifications',
    },
    {
      id: 'password-change',
      text: 'Change password',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/profile/password',
    },
  ],
};

export const NAV_THEME_TOGGLE: NavItem = {
  id: 'theme-toggle',
  text: 'Change to Dark Theme',
};

export const NAV_SIGN_OUT: NavItem = {
  id: 'sign-out',
  icon: 'sign-out',
  text: 'Sign out',
  url: '/graph/logout',
  target: '_self',
};

export const NAV_HELP: NavItem = {
  id: 'help',
  icon: 'help',
  text: 'Help',
  url: `${PMM_NEW_NAV_PATH}/help`,
};
