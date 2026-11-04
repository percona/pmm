import {
  DATABASE_SERVICE_DEFAULTS,
  DISABLE_COLLECTORS_DEFAULTS,
  MAX_QUERY_LENGTH_DEFAULTS,
  QUERY_ANALYTICS_OPTIONS_DEFAULTS,
  TLS_CERTIFICATES_DEFAULTS,
} from '../ServiceForm.constants';
import { toRadioOptions } from '../ServiceForm.utils';
import { Messages } from './PostgreSQLForm.messages';
import {
  AutoDiscoveryOption,
  MaxConnectionsOption,
  type PostgreSQLFormValues,
  type PostgreSQLOptionsFormValues,
  TrackingOption,
} from './PostgreSQLForm.types';

export const TRACKING_OPTIONS = toRadioOptions(
  Object.values(TrackingOption),
  Messages.tracking.options
);

export const AUTO_DISCOVERY_OPTIONS = toRadioOptions(
  Object.values(AutoDiscoveryOption),
  Messages.autoDiscovery.options
);

export const AUTO_DISCOVERY_LIMITS = {
  [AutoDiscoveryOption.enabled]: 10,
  [AutoDiscoveryOption.disabled]: -1,
};

export const MAX_CONNECTIONS_OPTIONS = toRadioOptions(
  Object.values(MaxConnectionsOption),
  Messages.maxConnections.options
);

export const POSTGRESQL_PORT = '5432';

// RDS and Azure collect query analytics through pg_stat_statements only.
export const DISCOVERED_TRACKING_OPTIONS = TRACKING_OPTIONS.filter(
  ({ value }) => value !== TrackingOption.pgMonitor
);

export const POSTGRESQL_OPTIONS_DEFAULTS: PostgreSQLOptionsFormValues = {
  ...TLS_CERTIFICATES_DEFAULTS,
  ...QUERY_ANALYTICS_OPTIONS_DEFAULTS,
  disableCommentsParsing: true,
  ...DISABLE_COLLECTORS_DEFAULTS,
  ...MAX_QUERY_LENGTH_DEFAULTS,
  database: '',
  tracking: TrackingOption.pgMonitor,
  autoDiscoveryOptions: AutoDiscoveryOption.enabled,
  autoDiscoveryLimit: '10',
  maxConnectionsOptions: MaxConnectionsOption.disabled,
  maxExporterConnections: '5',
};

export const POSTGRESQL_DEFAULTS: PostgreSQLFormValues = {
  ...DATABASE_SERVICE_DEFAULTS,
  port: POSTGRESQL_PORT,
  ...POSTGRESQL_OPTIONS_DEFAULTS,
};
