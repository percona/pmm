import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  getIntegerError,
  maxQueryLengthSchema,
  queryAnalyticsOptionsSchema,
  requireCredentialsWithoutTls,
  requireWhen,
  tlsCertificatesSchema,
} from '../ServiceForm.schema';
import {
  AutoDiscoveryOption,
  MaxConnectionsOption,
  type PostgreSQLOptionsFormValues,
  TrackingOption,
} from './PostgreSQLForm.types';

// The PostgreSQL options, shared by the remote, RDS and Azure forms.
export const postgresqlOptionsSchema = z.object({
  ...tlsCertificatesSchema.shape,
  ...queryAnalyticsOptionsSchema.shape,
  ...disableCollectorsSchema.shape,
  ...maxQueryLengthSchema.shape,
  database: z.string(),
  tracking: z.enum(TrackingOption),
  autoDiscoveryOptions: z.enum(AutoDiscoveryOption),
  autoDiscoveryLimit: z.string(),
  maxConnectionsOptions: z.enum(MaxConnectionsOption),
  maxExporterConnections: z.string(),
});

export const requireAutoDiscoveryLimit =
  requireWhen<PostgreSQLOptionsFormValues>(
    (values) => values.autoDiscoveryOptions === AutoDiscoveryOption.custom,
    'autoDiscoveryLimit',
    (value) => getIntegerError(value, -1)
  );

export const requireMaxExporterConnections =
  requireWhen<PostgreSQLOptionsFormValues>(
    (values) => values.maxConnectionsOptions === MaxConnectionsOption.enabled,
    'maxExporterConnections',
    (value) => getIntegerError(value, 0)
  );

export const postgresqlSchema = z
  .object({ ...databaseServiceSchema.shape, ...postgresqlOptionsSchema.shape })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(requireAutoDiscoveryLimit)
  .superRefine(requireMaxExporterConnections);
