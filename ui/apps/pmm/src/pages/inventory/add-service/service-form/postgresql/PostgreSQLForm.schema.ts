import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  maxQueryLengthSchema,
  optionalIntegerText,
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
    autoDiscoveryLimit: optionalIntegerText(-1),
    maxConnectionsOptions: z.enum(MaxConnectionsOption),
    maxExporterConnections: optionalIntegerText(0),
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(
    requireWhen(
      (values) => values.autoDiscoveryOptions === AutoDiscoveryOption.custom,
      'autoDiscoveryLimit'
    )
  )
  .superRefine(
    requireWhen(
      (values) => values.maxConnectionsOptions === MaxConnectionsOption.enabled,
      'maxExporterConnections'
    )
  );
