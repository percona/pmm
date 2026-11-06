import { PMM_NEW_NAV_GRAFANA_PATH, PMM_NEW_NAV_PATH } from 'lib/constants';
import type {
  EngineFamily,
  EngineId,
  EngineInfo,
  GroupFilters,
  InstanceRole,
} from './AmazonRds.types';

export const ENGINES: Record<EngineId, EngineInfo> = {
  'aurora-mysql': { label: 'Aurora MySQL', family: 'mysql', aurora: true },
  'aurora-postgresql': {
    label: 'Aurora PostgreSQL',
    family: 'postgresql',
    aurora: true,
  },
  mysql: { label: 'MySQL', family: 'mysql', aurora: false },
  mariadb: { label: 'MariaDB', family: 'mysql', aurora: false },
  postgres: { label: 'PostgreSQL', family: 'postgresql', aurora: false },
};

// aws_engine of Aurora MySQL 1 (MySQL 5.6 compatible).
export const LEGACY_AURORA_ENGINE = 'aurora';

// Without AWS details the engine is the PMM service type's plain engine.
export const SERVICE_TYPE_ENGINES: Record<string, EngineId> = {
  mysql: 'mysql',
  postgresql: 'postgres',
};

export const UNKNOWN_GROUP = 'AWS info unavailable';

export const ALERTING_URL = `${PMM_NEW_NAV_PATH}/alerting/status`;

export const ADVISORS_URL = `${PMM_NEW_NAV_GRAFANA_PATH}/advisors/insights`;

export const DASHBOARD_URLS = {
  auroraDetails: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-amazonaurora/mysql-amazon-aurora-details`,
  mysqlSummary: `${PMM_NEW_NAV_GRAFANA_PATH}/d/mysql-instance-summary/mysql-instance-summary`,
  postgresqlSummary: `${PMM_NEW_NAV_GRAFANA_PATH}/d/postgresql-instance-summary/postgresql-instance-summary`,
  nodeSummary: `${PMM_NEW_NAV_GRAFANA_PATH}/d/node-instance-summary/node-summary`,
};

export const NO_VALUE = '-';

export const THROUGHPUT_UNITS: Record<EngineFamily, string> = {
  mysql: 'qps',
  postgresql: 'tps',
};

export const SEARCH_DEBOUNCE_MS = 200;

export const URL_PARAMS: Record<keyof GroupFilters, string> = {
  query: 'rds-q',
  regions: 'rds-region',
  accounts: 'rds-account',
  engines: 'rds-engine',
  types: 'rds-type',
  groups: 'rds-id',
};

export const NO_FILTERS: GroupFilters = {
  query: '',
  regions: [],
  accounts: [],
  engines: [],
  types: [],
  groups: [],
};

export const ROLE_ORDER: InstanceRole[] = ['writer', 'reader', 'unknown'];
