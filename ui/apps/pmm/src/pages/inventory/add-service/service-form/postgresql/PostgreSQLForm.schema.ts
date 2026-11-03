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
  TrackingOption,
} from './PostgreSQLForm.types';

export const postgresqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
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
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(
    requireWhen(
      (values) => values.autoDiscoveryOptions === AutoDiscoveryOption.custom,
      'autoDiscoveryLimit',
      (value) => getIntegerError(value, -1)
    )
  )
  .superRefine(
    requireWhen(
      (values) => values.maxConnectionsOptions === MaxConnectionsOption.enabled,
      'maxExporterConnections',
      (value) => getIntegerError(value, 0)
    )
  );
