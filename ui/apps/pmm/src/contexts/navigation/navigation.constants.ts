import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
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
import Security from '@mui/icons-material/Security';
import SettingsApplicationsOutlined from '@mui/icons-material/SettingsApplicationsOutlined';
import SettingsOutlined from '@mui/icons-material/SettingsOutlined';
import PageviewOutlined from '@mui/icons-material/PageviewOutlined';
import SpaceDashboardOutlined from '@mui/icons-material/SpaceDashboardOutlined';
import {
  CirclesExtIcon,
  Graph4Icon,
  Graph5Icon,
  NetworkIntelligenceIcon,
  NetworkNodeIcon,
  PerconaMoIcon,
  PerconaMyIcon,
  PerconaPoIcon,
  PerconaVaIcon,
  QueryStatsIcon,
} from '@percona/peak-ui';
import { NavItem } from 'types/navigation.types';

export const NAV_HOME_PAGE: NavItem = {
  id: 'home-page',
  icon: SpaceDashboardOutlined,
  text: 'Overview',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/`,
  matches: [
    `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-home/home-dashboard`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-home`,
  ],
};

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
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-cluster-details/valkey-redis-cluster-detail`,
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
  text: 'Query Analytics (QAN)',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics`,
  matches: ['*', `${PMM_NEW_NAV_PATH}/rta/*`],
};

//
// All Dashbaords
//
export const NAV_DASHBOARDS: NavItem = {
  id: 'dashboards',
  icon: DashboardOutlined,
  text: 'All dashboards',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/dashboards`,
  matches: [
    '*',
    `${PMM_NEW_NAV_GRAFANA_PATH}/dashboard/*`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/playlists`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/playlists/*`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/library-panels`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/library-panels/*`,
  ],
};

//
// Explore
//

export const NAV_EXPLORE: NavItem = {
  id: 'explore',
  icon: ExploreOutlined,
  text: 'Explore',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/explore`,
  matches: [
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
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/insights`,
  matches: [
    `${PMM_NEW_NAV_GRAFANA_PATH}/advisors`,
    `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/*`,
  ],
};

//
// Inventory
//
export const NAV_INVENTORY: NavItem = {
  id: 'inventory',
  icon: Graph4Icon,
  text: 'Inventory',
  url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory`,
  children: [
    {
      id: 'inventory-services',
      text: 'Services',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/services`,
      matches: [
        '*',
        `${PMM_NEW_NAV_GRAFANA_PATH}/edit-instance/*`,
        `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance`,
        `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance/:type`,
      ],
      action: {
        label: 'Add service',
        url: `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance`,
      },
    },
    {
      id: 'inventory-nodes',
      text: 'Nodes',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/inventory/nodes`,
      matches: ['*'],
      action: {
        label: 'Add node',
        url: `${PMM_NEW_NAV_GRAFANA_PATH}/add-instance`,
      },
    },
  ],
};

//
// Configuration
//
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
    {
      id: 'org-management',
      text: 'Org. management',
      url: `${PMM_NEW_NAV_GRAFANA_PATH}/admin/orgs`,
      children: [
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
      ],
    },
  ],
};

//
// Users and Access
//
export const NAV_USERS_AND_ACCESS: NavItem = {
  id: 'users-and-access',
  icon: Security,
  text: 'Users and access',
  url: PMM_NEW_NAV_GRAFANA_PATH + '/admin/users',
  children: [
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
    {
      id: 'rbac-roles',
      text: 'Access roles',
      url: PMM_NEW_NAV_GRAFANA_PATH + '/roles',
    },
  ],
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
  url: '/graph/logout',
  target: '_self',
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
  url: '/graph/login',
  target: '_self',
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
