import { z } from 'zod';
import {
  postgresqlOptionsSchema,
  requireAutoDiscoveryLimit,
  requireMaxExporterConnections,
} from '../../postgresql/PostgreSQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentials,
} from '../../ServiceForm.schema';
import { rdsSchema } from '../RdsServiceForm.schema';

export const rdsPostgresqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...postgresqlOptionsSchema.shape,
    ...rdsSchema.shape,
  })
  .superRefine(requireCredentials)
  .superRefine(requireAutoDiscoveryLimit)
  .superRefine(requireMaxExporterConnections);
