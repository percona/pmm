import { z } from 'zod';
import {
  postgresqlOptionsSchema,
  requireAutoDiscoveryLimit,
} from '../../postgresql/PostgreSQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentials,
} from '../../ServiceForm.schema';
import { azureSchema } from '../AzureServiceForm.schema';

export const azurePostgresqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...postgresqlOptionsSchema.shape,
    ...azureSchema.shape,
  })
  .superRefine(requireCredentials)
  .superRefine(requireAutoDiscoveryLimit);
