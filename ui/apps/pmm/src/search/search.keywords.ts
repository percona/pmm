import {
  NAV_MONGO,
  NAV_MYSQL,
  NAV_OS,
  NAV_POSTGRESQL,
  NAV_VALKEY,
} from 'contexts/navigation/navigation.constants';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import type { NavItem } from 'types/navigation.types';
import type { SearchItem } from './search.types';
import { canonicalKey, techLabel } from './search.text';

export const TECHNOLOGIES: NavItem[] = [
  NAV_MYSQL,
  NAV_MONGO,
  NAV_POSTGRESQL,
  NAV_VALKEY,
  NAV_OS,
];

// Keyed by technology nav id.
export const TECH_ALIASES: Record<string, string[]> = {
  [NAV_POSTGRESQL.id]: ['postgres', 'pg'],
  [NAV_MONGO.id]: ['mongo', 'psmdb'],
  [NAV_VALKEY.id]: ['redis'],
  [NAV_MYSQL.id]: ['mariadb', 'percona server', 'pxc', 'galera'],
  [NAV_OS.id]: ['os', 'linux', 'host', 'node', 'server'],
};

// Keyed by the service type that /v1/management/services reports, lowercased.
export const SERVICE_TECH: Record<
  string,
  { name: string; nav?: NavItem; dashboard?: string }
> = {
  mysql: {
    name: 'MySQL',
    nav: NAV_MYSQL,
    dashboard: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-summary/mysql-instance-summary`,
  },
  mongodb: {
    name: 'MongoDB',
    nav: NAV_MONGO,
    dashboard: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mongodb-instance-summary/mongodb-instance-summary`,
  },
  postgresql: {
    name: 'PostgreSQL',
    nav: NAV_POSTGRESQL,
    dashboard: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-summary/postgresql-instance-summary`,
  },
  valkey: {
    name: 'Valkey',
    nav: NAV_VALKEY,
    dashboard: `${PMM_NEW_NAV_GRAFANA_PATH}/d/valkey-overview/valkey-redis-overview`,
  },
  proxysql: {
    name: 'ProxySQL',
    dashboard: `${PMM_NEW_NAV_GRAFANA_PATH}/d/proxysql-instance-summary`,
  },
  haproxy: {
    name: 'HAProxy',
    dashboard: `${PMM_NEW_NAV_GRAFANA_PATH}/d/haproxy-instance-summary`,
  },
  external: { name: 'External' },
};

const BACKUPS = ['backup', 'restore', 'pitr'];
const USERS_AND_ACCESS = ['users and access', 'rbac', 'permissions'];
const ADVISORS = ['checks', 'stt'];
const INVENTORY = ['agents', 'exporters'];
const FLEET = ['fleet', 'clusters'];
const AUTOMATIONS = ['scan', 'install', 'runs'];

// Every key must name an item of an admin index with every feature on; the drift test checks it.
export const KEYWORDS: Record<string, string[]> = {
  'page:qan': ['qan', 'slow queries', 'query performance'],
  'page:realtime-queries': ['rta', 'live queries', 'running queries'],
  'action:inventory-services-action': [
    'add instance',
    'add database',
    'register',
    'monitor',
    'connect',
  ],
  'page:inventory-services': INVENTORY,
  'page:inventory-nodes': INVENTORY,
  'page:advisors': ADVISORS,
  'page:advisors-configuration': ADVISORS,
  'page:advisors-performance': ADVISORS,
  'page:advisors-query': ADVISORS,
  'page:advisors-security': ADVISORS,
  'page:app-mongodb-backups': BACKUPS,
  'page:app-mysql-backups': BACKUPS,
  'page:app-postgresql-backups': BACKUPS,
  'page:app-valkey-backups': BACKUPS,
  'page:backup-scheduled': BACKUPS,
  'page:backup-restore': BACKUPS,
  'page:backup-locations': BACKUPS,
  'action:create-backup': BACKUPS,
  'action:new-mysql-backup': BACKUPS,
  'page:users': USERS_AND_ACCESS,
  'page:teams': USERS_AND_ACCESS,
  'page:rbac-roles': USERS_AND_ACCESS,
  'page:service-accounts': ['tokens', 'api key'],
  'page:alerts-contact-points': [
    'email',
    'smtp',
    'slack',
    'pagerduty',
    'webhook',
  ],
  'page:alerts-silences': ['mute', 'snooze'],
  'page:updates': ['upgrade', 'version'],
  'page:high-availability': ['high availability', 'cluster', 'raft', 'leader'],
  'page:extensions-atw': ['support bundle', 'logs'],
  'page:app-operations-for-mongodb': [
    'om',
    'operations',
    'fleet',
    'install mongodb',
    'replica set',
  ],
  'page:om-clusters': FLEET,
  'page:om-services': FLEET,
  'page:om-nodes': ['hosts', 'machines'],
  'page:om-installs': AUTOMATIONS,
  'page:om-scans': AUTOMATIONS,
  'action:om-install-mongodb': ['bootstrap', 'new replica set'],
  'action:sign-out': ['log out', 'logout'],
  'action:theme-toggle': ['dark mode', 'light mode', 'theme'],
  // The overview child folds into its technology page as a whole phrase, so a bare "overview" still ranks Overview first.
  ...Object.fromEntries(
    TECHNOLOGIES.map((tech) => [
      `page:${tech.id}`,
      [techLabel(tech.text ?? '', 'Overview'), ...TECH_ALIASES[tech.id]],
    ])
  ),
  // MongoDB opens its hub, so "MongoDB overview" names the dashboard, not the page.
  [`page:${NAV_MONGO.id}`]: [
    ...TECH_ALIASES[NAV_MONGO.id],
    'clusters',
    'replica set',
    'sharded',
  ],
};

type SearchItemFields = Omit<SearchItem, 'keywords' | 'dedupeKey' | 'order'> &
  Partial<Pick<SearchItem, 'keywords' | 'dedupeKey'>>;

// Builds an index entry with its curated keywords and its destination as the dedupe key.
export const searchItem = ({
  keywords = [],
  dedupeKey,
  ...fields
}: SearchItemFields): SearchItem => ({
  ...fields,
  keywords: [...keywords, ...(KEYWORDS[fields.id] ?? [])],
  dedupeKey: dedupeKey ?? canonicalKey(fields.to ?? fields.href) ?? fields.id,
  order: 0,
});
