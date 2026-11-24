import {
  PMM_LOGIN_URL,
  PMM_LOGOUT_URL,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
  EXTENSIONS_MYSQL_BACKUPS_PATH,
  PMM_ADD_SERVICE_URL,
  PMM_INVENTORY_NODES_PATH,
  PMM_INVENTORY_PATH,
  PMM_INVENTORY_SERVICES_PATH,
} from 'lib/constants';
import { clearReturnTo, saveReturnTo } from 'contexts/auth/auth.returnTo';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import AccountCircleOutlined from '@mui/icons-material/AccountCircleOutlined';
import AppsRounded from '@mui/icons-material/AppsRounded';
import DashboardOutlined from '@mui/icons-material/DashboardOutlined';
import ExploreOutlined from '@mui/icons-material/ExploreOutlined';
import Groups from '@mui/icons-material/Groups';
import HelpOutline from '@mui/icons-material/HelpOutline';
import Logout from '@mui/icons-material/Logout';
import MemoryOutlined from '@mui/icons-material/MemoryOutlined';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import SettingsApplicationsOutlined from '@mui/icons-material/SettingsApplicationsOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';
import PageviewOutlined from '@mui/icons-material/PageviewOutlined';
import Search from '@mui/icons-material/Search';
import {
  CirclesExtIcon,
  Graph4Icon,
  Graph5Icon,
  HaproxyIcon,
  NetworkIntelligenceIcon,
  NetworkNodeIcon,
  PerconaMoIcon,
  PerconaMyIcon,
  PerconaPoIcon,
  PerconaVaIcon,
  ProxyIcon,
  QueryStatsIcon,
  SearchInsightsIcon,
} from '@percona/peak-ui';
import { NavItem } from 'types/navigation.types';

//
// MySQL dashboards
//
export const NAV_MYSQL: NavItem = {
  id: 'mysql',
  text: 'MySQL',
  icon: PerconaMyIcon,
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-overview/mysql-instances-overview`,
  children: [
    {
      id: 'mysql-overview',
      icon: PageviewOutlined,
      text: 'Overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-overview/mysql-instances-overview`,
    },
    {
      id: 'mysql-summary',
      icon: AppsRounded,
      text: 'Summary',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-summary/mysql-instance-summary`,
    },
    {
      id: 'mysql-high-availability',
      icon: Graph5Icon,
      text: 'High availability',
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
      id: 'mysql-table-details',
      text: 'Table details',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-table/mysql-table-details`,
    },
    {
      id: 'mysql-myrocks-details',
      text: 'MyRocks details',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-myrocks/mysql-myrocks-details`,
    },
  ],
};

//
// MongoDB dashboards
//
export const NAV_MONGO: NavItem = {
  id: 'mongo',
  icon: PerconaMoIcon,
  text: 'MongoDB',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-overview/mongodb-instances-overview`,
  children: [
    {
      id: 'mongo-overview',
      icon: PageviewOutlined,
      text: 'Overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-overview/mongodb-instances-overview`,
    },
    {
      id: 'mongo-summary',
      icon: AppsRounded,
      text: 'Summary',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-summary/mongodb-instance-summary`,
    },
    {
      id: 'mongo-high-availability',
      icon: Graph5Icon,
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
      id: 'mongo-backup-details',
      text: 'Backup status',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-backup-details/mongodb-backup-details`,
    },
    {
      id: 'mongo-collections-overview',
      text: 'Collections',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-collections-overview/mongodb-collections-overview`,
    },
    {
      id: 'mongo-unused-indexes',
      text: 'Unused indexes',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-unused-indexes/mongodb-unused-indexes`,
    },
    {
      id: 'mongo-top-queries',
      text: 'Top queries',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-top-queries/mongodb-top-queries`,
    },
    {
      id: 'mongo-oplog-details',
      text: 'Oplog',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-oplog-details/mongodb-oplog-details`,
    },
    {
      id: 'mongo-clustersync-details',
      text: 'ClusterSync',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-clustersync-details/mongodb-clustersync-details`,
    },
  ],
};

//
// PostgreSQL
//
export const NAV_POSTGRESQL: NavItem = {
  id: 'postgre',
  text: 'PostgreSQL',
  icon: PerconaPoIcon,
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-overview/postgresql-instances-overview`,
  children: [
    {
      id: 'postgresql-overwiew',
      text: 'Overview',
      icon: PageviewOutlined,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-overview/postgresql-instances-overview`,
    },
    {
      id: 'postgresql-summary',
      text: 'Summary',
      icon: AppsRounded,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-summary/postgresql-instance-summary`,
    },
    {
      id: 'postgresql-ha',
      text: 'High availability',
      icon: Graph5Icon,
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
  icon: SettingsApplicationsOutlined,
  text: 'Operating system',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-overview/nodes-overview`,
  children: [
    {
      id: 'node-overview',
      icon: PageviewOutlined,
      text: 'Overview',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-overview/nodes-overview`,
    },
    {
      id: 'node-summary',
      icon: AppsRounded,
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
// Valkey
//
export const NAV_VALKEY: NavItem = {
  id: 'valkey',
  text: 'Valkey',
  icon: PerconaVaIcon,
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-overview/valkey-redis-overview`,
  children: [
    {
      id: 'valkey-overview',
      text: 'Overview',
      icon: PageviewOutlined,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-overview/valkey-redis-overview`,
    },
    {
      id: 'valkey-load',
      text: 'Load',
      icon: Groups,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-load/valkey-redis-load`,
    },
    {
      id: 'valkey-memory',
      text: 'Memory',
      icon: MemoryOutlined,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-memory/valkey-redis-memory`,
    },
    {
      id: 'valkey-network',
      text: 'Network',
      icon: NetworkNodeIcon,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-network/valkey-redis-network`,
    },
    {
      id: 'valkey-clients',
      text: 'Clients',
      icon: Groups,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-clients/valkey-redis-clients`,
    },
    {
      id: 'valkey-cluster-details',
      text: 'Cluster Details',
      icon: Graph5Icon,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-cluster-details/valkey-redis-cluster-details`,
    },
    {
      id: 'valkey-replication',
      text: 'Replication',
      icon: Graph5Icon,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-replication/valkey-redis-replication`,
    },
    {
      id: 'valkey-persistence',
      text: 'Persistence',
      icon: Groups,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-persistence-details/valkey-redis-persistence-details`,
    },
    {
      id: 'valkey-commands',
      text: 'Command details',
      icon: Groups,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-command-details/valkey-redis-command-detail`,
    },
    {
      id: 'valkey-slowlog',
      text: 'Slow Log',
      icon: Groups,
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-slowlog/valkey-redis-slowlog`,
    },
  ],
};

//
// QAN
//
export const NAV_QAN: NavItem = {
  id: 'qan',
  icon: QueryStatsIcon,
  text: 'Query analytics',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`,
  matches: ['*'],
};

//
// Dashboards
//
// Grafana pages under /dashboard with no entry of their own, such as a new
// dashboard or an import, highlight the group.
export const NAV_DASHBOARDS: NavItem = {
  id: 'dashboards',
  icon: DashboardOutlined,
  text: 'Dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards`,
  matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/*`],
};

export const NAV_DASHBOARDS_BROWSE: NavItem = {
  id: 'dashboards-browse',
  icon: SearchInsightsIcon,
  text: 'All dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards`,
  matches: ['*'],
};

export const NAV_PROXYSQL: NavItem = {
  id: 'proxysql',
  icon: ProxyIcon,
  text: 'ProxySQL',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/proxysql-instance-summary/proxysql-instance-summary`,
};

export const NAV_HAPROXY: NavItem = {
  id: 'haproxy',
  icon: HaproxyIcon,
  text: 'HAProxy',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/haproxy-instance-summary/haproxy-instance-summary`,
};

export const NAV_DASHBOARDS_SHARED: NavItem = {
  id: 'dashboards-shared',
  text: 'Shared dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/public`,
  matches: ['*'],
};

export const NAV_DASHBOARDS_PLAYLISTS: NavItem = {
  id: 'dashboards-playlists',
  text: 'Playlists',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/playlists`,
  matches: ['*'],
};

export const NAV_DASHBOARDS_SNAPSHOTS: NavItem = {
  id: 'dashboards-snapshots',
  text: 'Snapshots',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/snapshots`,
  matches: ['*'],
};

export const NAV_DASHBOARDS_LIBRARY_PANELS: NavItem = {
  id: 'dashboards-library-panels',
  text: 'Library panels',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/library-panels`,
  matches: ['*'],
};

//
// Explore
//

export const NAV_EXPLORE: NavItem = {
  id: 'explore',
  icon: ExploreOutlined,
  text: 'Explore data',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/explore`,
  matches: [
    '*',
    `${PMM_NEW_NAV_GRAFANA_PATH}/drilldown`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/a/:appId/explore`,
  ],
};

//
// Alerting
//
export const NAV_ALERTS_TEMPLATES: NavItem = {
  id: 'alerts-templates',
  text: 'Templates',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/alert-rule-templates`,
  matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/alerting/new-from-template/*`],
};

export const NAV_ALERTS_RULES: NavItem = {
  id: 'alerts-rules',
  text: 'Alert rules',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/list`,
  matches: [
    `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/new/*`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/:id/edit`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/:id/edit`,
  ],
};

export const NAV_ALERTS_SILENCES: NavItem = {
  id: 'alerts-silences',
  text: 'Silences',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/silences`,
};

export const NAV_ALERTS_GROUPS: NavItem = {
  id: 'alerts-groups',
  text: 'Alert groups',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/groups`,
};

export const NAV_ALERTS_STATUS: NavItem = {
  id: 'alerts-status',
  text: 'Status',
  url: `${PMM_NEW_NAV_PATH}/alerting/status`,
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
  text: 'Alert settings',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/alerting/admin`,
  matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/connections/datasources/alertmanager`],
};

export const NAV_ALERTS: NavItem = {
  id: 'alerts',
  icon: NotificationsOutlined,
  text: 'Alerts',
  url: `${PMM_NEW_NAV_PATH}/alerting/status`,
};

export const NAV_ADVISORS: NavItem = {
  id: 'advisors',
  icon: NetworkIntelligenceIcon,
  text: 'Advisors',
  url: `${PMM_NEW_NAV_PATH}/advisors`,
};

// the sidebar links a parent item to its first child's url, so the Advisors
// children are listed explicitly (Insights is first, see addAdvisors)
export const NAV_ADVISORS_CHECKS: NavItem = {
  id: 'advisors-checks',
  text: 'Catalog',
  url: `${PMM_NEW_NAV_PATH}/advisors`,
};

export const NAV_ADVISORS_INSIGHTS = {
  id: 'advisors-insights',
  text: 'Insights',
  url: `${PMM_NEW_NAV_PATH}/advisors/insights`,
};

export const NAV_ADVISORS_RUNS = {
  id: 'advisors-runs',
  text: 'Run history',
  url: `${PMM_NEW_NAV_PATH}/advisors/runs`,
};

export const NAV_ADVISORS_SCHEDULER = {
  id: 'advisors-scheduler',
  text: 'Scheduler',
  url: `${PMM_NEW_NAV_PATH}/advisors/scheduler`,
};

//
// Inventory
//
export const NAV_INVENTORY: NavItem = {
  id: 'inventory',
  icon: Graph4Icon,
  text: 'Inventory',
  url: PMM_INVENTORY_PATH,
  children: [
    {
      id: 'inventory-services',
      text: 'Services',
      isDefault: true,
      url: PMM_INVENTORY_SERVICES_PATH,
      matches: [
        '*',
        `${PMM_NEW_NAV_GRAFANA_PATH}/edit-instance/*`,
        PMM_ADD_SERVICE_URL,
        `${PMM_ADD_SERVICE_URL}/:type`,
      ],
      action: {
        label: 'Add a service',
        url: PMM_ADD_SERVICE_URL,
      },
    },
    {
      id: 'inventory-nodes',
      text: 'Nodes',
      url: PMM_INVENTORY_NODES_PATH,
      matches: ['*'],
    },
  ],
};

export const section = (id: string, text: string): NavItem => ({
  id: `section-${id}`,
  type: 'menu-section',
  text,
});

//
// Configuration
//
// Grafana's organization and user pages are listed under headings, so the
// group stays one level deep.
export const NAV_CONFIGURATION: NavItem = {
  id: 'configuration',
  icon: SettingsOutlined,
  text: 'Configuration',
  url: `${PMM_NEW_NAV_PATH}/settings`,
  matches: [
    `${PMM_NEW_NAV_PATH}/settings`,
    `${PMM_NEW_NAV_PATH}/settings/*`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/plugins`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/admin`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/admin/general`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/admin/settings`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/admin/plugins`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/datasources/correlations`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/admin/extensions`,
  ],
  children: [
    {
      id: 'configuration-settings',
      text: 'Settings',
      url: `${PMM_NEW_NAV_PATH}/settings`,
      matches: [
        `${PMM_NEW_NAV_PATH}/settings`,
        `${PMM_NEW_NAV_PATH}/settings/*`,
      ],
    },
    {
      id: 'updates',
      text: 'Updates',
      url: `${PMM_NEW_NAV_PATH}/updates`,
    },
    section('organization', 'Organization'),
    {
      id: 'organizations',
      text: 'Organizations',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/admin/orgs`,
      matches: ['*'],
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
    section('users-and-access', 'Users and access'),
    {
      id: 'users',
      text: 'Users',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/admin/users',
      matches: ['*'],
    },
    {
      id: 'teams',
      text: 'Teams',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/org/teams',
      matches: ['*'],
    },
    {
      id: 'service-accounts',
      text: 'Service accounts',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/org/serviceaccounts',
      matches: ['*'],
    },
  ],
};

export const NAV_ACCESS_CONTROL: NavItem = {
  id: 'rbac-roles',
  text: 'Access roles',
  url: PMM_NEW_NAV_GRAFANA_PATH + '/roles',
};

//
// Account
//
export const NAV_ACCOUNT: NavItem = {
  id: 'account',
  icon: AccountCircleOutlined,
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
  ],
};

export const NAV_CHANGE_PASSWORD: NavItem = {
  id: 'password-change',
  text: 'Change password',
  url: PMM_NEW_NAV_GRAFANA_PATH + '/profile/password',
};

export const NAV_THEME_TOGGLE: NavItem = {
  id: 'theme-toggle',
  text: 'Switch to dark mode',
};

export const NAV_SIGN_OUT: NavItem = {
  id: 'sign-out',
  icon: Logout,
  text: 'Sign out',
  url: PMM_LOGOUT_URL,
  target: '_self',
  // Leaving on purpose should not drop the user back on that page at the next login.
  onClick: clearReturnTo,
};

export const NAV_HELP: NavItem = {
  id: 'help',
  icon: HelpOutline,
  text: 'Help',
  url: `${PMM_NEW_NAV_PATH}/help`,
};

export const NAV_SIGN_IN: NavItem = {
  id: 'sign-in',
  icon: LoginOutlinedIcon,
  text: 'Sign in',
  url: PMM_LOGIN_URL,
  target: '_self',
  // Remembers the page the way an involuntary bounce through redirectToLogin() does. Stays a real
  // anchor: /graph/login is exempt from the nginx redirect and must not enter the router.
  //
  // Cleared first because the bounce guard only makes sense for involuntary cycles - under
  // anonymous access this click often lands seconds after the shell restored this very page,
  // which is exactly the state the guard refuses.
  onClick: () => {
    clearReturnTo();
    saveReturnTo();
  },
};

/**
 * Mapping of menu items id to folders name.
 *
 * Folders are created based on the folder name in grafana-dashboards.
 */
export const NAV_FOLDER_MAP: Record<string, string> = {
  system: 'OS',
  mysql: 'MySQL',
  mongo: 'MongoDB',
  postgre: 'PostgreSQL',
  valkey: 'Valkey',
};

export const NAV_OTHER_DASHBOARDS_TEMPLATE: Partial<NavItem> = {
  icon: Search,
  text: 'Other dashboards',
};

/*
 * High Availability
 */
export const NAV_HIGH_AVAILABILITY: NavItem = {
  id: 'high-availability',
  icon: CirclesExtIcon,
  text: 'PMM HA',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-ha-health-overview/pmm-ha-health-overview`,
};

//
// Apps
//
// Listed under Apps, under their heading. Grafana's backup pages have no tabs
// of their own, so each one is an entry.
export const NAV_BACKUPS: NavItem[] = [
  section('backups', 'Backups'),
  {
    id: 'backup-inventory',
    text: 'All backups',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/inventory`,
    matches: [`${PMM_NEW_NAV_GRAFANA_PATH}/backup/new`],
  },
  {
    id: 'scheduled-backups',
    text: 'Scheduled backup jobs',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/scheduled`,
  },
  {
    id: 'restore-history',
    text: 'Restores',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/restore`,
  },
  {
    id: 'storage-locations',
    text: 'Storage locations',
    url: `${PMM_NEW_NAV_GRAFANA_PATH}/backup/locations`,
  },
];

export const NAV_EXTENSIONS_MYSQL_BACKUPS: NavItem = {
  id: 'extensions-mysql-backups',
  text: 'MySQL Backups',
  url: EXTENSIONS_MYSQL_BACKUPS_PATH,
  matches: ['*'],
};
