import type { AddServicePayload } from 'types/services.types';
import {
  toCollectors,
  toDatabaseService,
  toLimit,
  toOptionalNumber,
  toOptionalString,
  toTlsCertificates,
} from '../ServiceForm.utils';
import { AUTO_DISCOVERY_LIMITS } from './PostgreSQLForm.constants';
import {
  AutoDiscoveryOption,
  MaxConnectionsOption,
  type PostgreSQLFormValues,
  TrackingOption,
} from './PostgreSQLForm.types';

export const toPostgreSQLPayload = (
  values: PostgreSQLFormValues
): AddServicePayload => ({
  postgresql: {
    ...toDatabaseService(values),
    ...toTlsCertificates(values),
    database: toOptionalString(values.database),
    qanPostgresqlPgstatementsAgent:
      values.tracking === TrackingOption.pgStatements,
    qanPostgresqlPgstatmonitorAgent:
      values.tracking === TrackingOption.pgMonitor,
    disableQueryExamples: values.disableQueryExamples,
    disableCommentsParsing: values.disableCommentsParsing,
    maxQueryLength: toOptionalNumber(values.maxQueryLength),
    disableCollectors: toCollectors(values.disableCollectors),
    autoDiscoveryLimit: toLimit(
      values.autoDiscoveryOptions,
      AutoDiscoveryOption.custom,
      values.autoDiscoveryLimit,
      AUTO_DISCOVERY_LIMITS
    ),
    maxExporterConnections: toLimit(
      values.maxConnectionsOptions,
      MaxConnectionsOption.enabled,
      values.maxExporterConnections,
      {}
    ),
  },
});
