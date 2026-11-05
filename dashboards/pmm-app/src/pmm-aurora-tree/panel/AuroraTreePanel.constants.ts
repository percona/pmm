import { BadgeColor } from '@grafana/ui';
import { AuroraTreeOptions, InstanceRole } from './AuroraTreePanel.types';

export const LABEL_FIELDS = {
  region: 'region',
  awsAccountId: 'aws_account_id',
  clusterId: 'aws_cluster_id',
  serviceName: 'service_name',
  nodeName: 'node_name',
};

// Prometheus names the value field of instant table query <refId> "Value #<refId>".
export const DEFAULT_OPTIONS: AuroraTreeOptions = {
  upField: 'Value #A',
  readOnlyField: 'Value #B',
  qpsField: 'Value #C',
  storageField: 'Value #D',
};

export const UNKNOWN_CLUSTER = 'AWS cluster unknown';

export const DETAILS_DASHBOARD_PATH = '/d/mysql-amazonaurora/mysql-amazon-aurora-details';

export const MYSQL_SUMMARY_DASHBOARD_PATH = '/d/mysql-instance-summary/mysql-instance-summary';

export const NODE_SUMMARY_DASHBOARD_PATH = '/d/node-instance-summary/node-summary';

export const NO_VALUE = '-';

export const QPS_UNIT = 'qps';

export const QPS_DECIMALS = 2;

export const STORAGE_UNIT = 'bytes';

export const SEARCH_DEBOUNCE_MS = 200;

export const URL_PARAMS = {
  query: 'aurora-q',
  regions: 'aurora-region',
  accounts: 'aurora-account',
  clusters: 'aurora-cluster',
};

export const ROLE_ORDER: InstanceRole[] = ['writer', 'reader', 'unknown'];

export const ROLE_COLORS: Record<InstanceRole, BadgeColor> = {
  writer: 'blue',
  reader: 'purple',
  unknown: 'darkgrey',
};
