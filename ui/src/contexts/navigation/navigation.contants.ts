import { MenuItem } from './navigation.context.types';

export const initialNavtree: MenuItem[] = [
  {
    icon: 'dashboards',
    title: 'Dashboards',
    children: [
      {
        title: 'PostgreSQL Instances Overview',
        to: '/d/postgresql-instance-overview/postgresql-instances-overview',
      },
      {
        title: 'Nodes Overview',
        to: '/d/node-instance-overview/nodes-overview',
      },
    ],
  },
  {
    icon: 'alerts',
    title: 'Alerts',
    to: '/alerts',
  },
  {
    title: 'Query Analytics',
    to: '/query-analytics',
  },
  {
    icon: 'settings',
    title: 'Settings',
    children: [
      {
        title: 'Metrics',
        to: '/settings/metrics-resolution',
      },
      {
        title: 'Updates',
        to: '/updates',
      },
    ],
  },
];
