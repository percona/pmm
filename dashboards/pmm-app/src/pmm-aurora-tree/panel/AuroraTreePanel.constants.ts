import { BadgeColor } from '@grafana/ui';
import {
  AuroraTreeOptions,
  EngineFamily,
  EngineId,
  EngineInfo,
  InstanceLabels,
  InstanceMetric,
  InstanceRole,
} from './AuroraTreePanel.types';

export const LABEL_FIELDS: Record<keyof InstanceLabels, string> = {
  serviceName: 'service_name',
  nodeName: 'node_name',
  nodeId: 'node_id',
  serviceType: 'service_type',
  region: 'region',
  awsAccountId: 'aws_account_id',
  clusterId: 'aws_cluster_id',
  awsEngine: 'aws_engine',
  // aws_rds_instance_info carries the DB instance identifier as instance.
  instanceId: 'instance',
  sourceInstanceId: 'aws_source_instance_id',
  sourceRegion: 'aws_source_region',
};

// Prometheus names the value field of instant table query <refId> "Value #<refId>".
export const DEFAULT_OPTIONS: AuroraTreeOptions = {
  upField: 'Value #A',
  innodbReadOnlyField: 'Value #C',
  readOnlyField: 'Value #D',
  inRecoveryField: 'Value #E',
  throughputField: 'Value #F',
  volumeUsedField: 'Value #G',
  freeStorageField: 'Value #H',
};

export const METRIC_OPTIONS: Record<InstanceMetric, keyof AuroraTreeOptions> = {
  up: 'upField',
  innodbReadOnly: 'innodbReadOnlyField',
  readOnly: 'readOnlyField',
  inRecovery: 'inRecoveryField',
  throughput: 'throughputField',
  volumeUsedBytes: 'volumeUsedField',
  freeStorageBytes: 'freeStorageField',
};

export const ENGINES: Record<EngineId, EngineInfo> = {
  'aurora-mysql': { label: 'Aurora MySQL', family: 'mysql', aurora: true },
  'aurora-postgresql': { label: 'Aurora PostgreSQL', family: 'postgresql', aurora: true },
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

export const DETAILS_DASHBOARD_PATH = '/d/mysql-amazonaurora/mysql-amazon-aurora-details';

export const MYSQL_SUMMARY_DASHBOARD_PATH = '/d/mysql-instance-summary/mysql-instance-summary';

export const POSTGRESQL_SUMMARY_DASHBOARD_PATH = '/d/postgresql-instance-summary/postgresql-instance-summary';

export const NODE_SUMMARY_DASHBOARD_PATH = '/d/node-instance-summary/node-summary';

export const NO_VALUE = '-';

export const THROUGHPUT_UNITS: Record<EngineFamily, string> = {
  mysql: 'qps',
  postgresql: 'tps',
};

export const THROUGHPUT_DECIMALS = 2;

export const STORAGE_UNIT = 'bytes';

export const SEARCH_DEBOUNCE_MS = 200;

export const URL_PARAMS = {
  query: 'rds-q',
  regions: 'rds-region',
  accounts: 'rds-account',
  engines: 'rds-engine',
  types: 'rds-type',
  groups: 'rds-id',
};

// Read when rds-id is absent, so links made before the rename still filter.
export const LEGACY_GROUPS_PARAM = 'rds-group';

export const ROLE_ORDER: InstanceRole[] = ['writer', 'reader', 'unknown'];

export const ROLE_COLORS: Record<InstanceRole, BadgeColor> = {
  writer: 'blue',
  reader: 'purple',
  unknown: 'darkgrey',
};
