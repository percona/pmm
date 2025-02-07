import { MenuItem } from './navigation.context.types';

const MYSQL_DASHBOARDS = {
  id: 'mysql',
  title: 'MySQL',
  icon: 'percona-database-mysql',
  to: '/graph/d/mysql-instance-overview/mysql-instances-overview',
  sortWeight: -1700,
  hideFromTabs: true,
  children: [
    {
      id: 'mysql-overview',
      title: 'Overview',
      icon: 'percona-nav-overview',
      to: '/graph/d/mysql-instance-overview/mysql-instances-overview',
      hideFromTabs: true,
    },
    {
      id: 'mysql-summary',
      title: 'Summary',
      icon: 'percona-nav-summary',
      to: '/graph/d/mysql-instance-summary/mysql-instance-summary',
      hideFromTabs: true,
    },
    {
      id: 'mysql-ha',
      title: 'High availability',
      icon: 'percona-cluster',
      hideFromTabs: true,
      showChildren: true,
      to: '/graph/d/mysql-group-replicaset-summary',
      children: [
        {
          id: 'mysql-group-replication-summary',
          title: 'Group replication summary',
          icon: 'percona-cluster',
          to: '/graph/d/mysql-group-replicaset-summary/mysql-group-replication-summary',
          hideFromTabs: true,
        },
        {
          id: 'mysql-replication-summary',
          title: 'Replication summary',
          icon: 'percona-cluster',
          to: '/graph/d/mysql-replicaset-summary/mysql-replication-summary',
          hideFromTabs: true,
        },
        {
          id: 'pxc-cluster-summary',
          title: 'PXC/Galera cluster summary',
          icon: 'percona-cluster',
          to: '/graph/d/pxc-cluster-summary/pxc-galera-cluster-summary',
          hideFromTabs: true,
        },
        {
          id: 'pxc-node-summary',
          title: 'PXC/Galera node summary',
          icon: 'percona-cluster',
          to: '/graph/d/pxc-node-summary/pxc-galera-node-summary',
          hideFromTabs: true,
        },
        {
          id: 'pxc-nodes-compare',
          title: 'PXC/Galera nodes compare',
          icon: 'percona-cluster',
          to: '/graph/d/pxc-nodes-compare/pxc-galera-nodes-compare',
          hideFromTabs: true,
        },
      ],
    },
    {
      id: 'mysql-command-handler-counters-compare',
      title: 'Command/Handler counters compare',
      icon: 'sitemap',
      to: '/graph/d/mysql-commandhandler-compare/mysql-command-handler-counters-compare',
    },
    {
      id: 'mysql-innodb-details',
      title: 'InnoDB details',
      icon: 'sitemap',
      to: '/graph/d/mysql-innodb/mysql-innodb-details',
    },
    {
      id: 'mysql-innodb-compression-details',
      title: 'InnoDB compression',
      icon: 'sitemap',
      to: '/graph/d/mysql-innodb-compression/mysql-innodb-compression-details',
    },
    {
      id: 'mysql-performance-schema-details',
      title: 'Performance schema',
      icon: 'sitemap',
      to: '/graph/d/mysql-performance-schema/mysql-performance-schema-details',
    },
    {
      id: 'mysql-query-response-time-details',
      title: 'Query response time',
      icon: 'sitemap',
      to: '/graph/d/mysql-queryresponsetime/mysql-query-response-time-details',
    },
    {
      id: 'mysql-table-details',
      title: 'Table details',
      icon: 'sitemap',
      to: '/graph/d/mysql-table/mysql-table-details',
    },
    {
      id: 'mysql-tokudb-details',
      title: 'TokuDB details',
      icon: 'sitemap',
      to: '/graph/d/mysql-tokudb/mysql-tokudb-details',
    },
    {
      id: 'mysql-other-dashboards',
      icon: 'search',
      title: 'Other dashboards',
      to: '/graph/dashboards/f/ae3tpjc6j2nswa/mysql',
    },
  ],
};

const PG_DASHBOARDS = {
  id: 'postgre',
  title: 'PostgreSQL',
  icon: 'percona-database-postgresql',
  to: '/graph/d/postgresql-instance-overview/postgresql-instances-overview',
  sortWeight: -1700,
  hideFromTabs: true,
  children: [
    {
      id: 'postgre-overwiew',
      title: 'Overview',
      icon: 'percona-nav-overview',
      to: '/graph/d/postgresql-instance-overview/postgresql-instances-overview',
      hideFromTabs: true,
    },
    {
      id: 'postgre-summary',
      title: 'Summary',
      icon: 'percona-nav-summary',
      to: '/graph/d/postgresql-instance-summary/postgresql-instance-summary',
      hideFromTabs: true,
    },
    {
      id: 'postgre-other-dashboards',
      icon: 'search',
      title: 'Other dashboards',
      to: '/graph/dashboards/f/be3tpjcbnv5dsa/postgre',
    },
  ],
};

export const initialNavtree: MenuItem[] = [
  {
    id: 'home',
    title: 'Home',
    to: '/graph/d/pmm-home',
  },
  {
    id: 'dashboards',
    icon: 'dashboards',
    title: 'Dashboards',
    children: [],
  },
  PG_DASHBOARDS,
  MYSQL_DASHBOARDS,
  {
    id: 'alerts',
    icon: 'alerts',
    title: 'Alerts',
    to: '/graph/alerting',
  },
  {
    id: 'query-analytics',
    title: 'Query Analytics',
    to: '/query-analytics',
  },
  {
    id: 'settings',
    icon: 'settings',
    title: 'Settings',
    children: [
      {
        id: 'metrics',
        title: 'Metrics',
        to: '/settings/metrics-resolution',
      },
      {
        id: 'updates',
        title: 'Updates',
        to: '/updates',
      },
    ],
  },
];

export const NAV_FOLDER_MAP: Record<string, string> = {
  system: 'OS',
  mysql: 'MySQL',
  mongo: 'MongoDB',
  postgre: 'PostgreSQL',
};
