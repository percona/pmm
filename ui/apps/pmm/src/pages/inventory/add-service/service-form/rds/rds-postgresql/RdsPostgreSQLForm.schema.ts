import { z } from 'zod';
import {
  postgresqlOptionsSchema,
  requireAutoDiscoveryLimit,
  requireMaxExporterConnections,
} from '../../postgresql/PostgreSQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentialsWithoutTls,
} from '../../ServiceForm.schema';
import { rdsSchema } from '../RdsServiceForm.schema';

export const rdsPostgresqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...postgresqlOptionsSchema.shape,
    ...rdsSchema.shape,
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(requireAutoDiscoveryLimit)
  .superRefine(requireMaxExporterConnections);
