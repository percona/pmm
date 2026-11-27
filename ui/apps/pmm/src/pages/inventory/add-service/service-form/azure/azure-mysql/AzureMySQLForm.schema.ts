import { z } from 'zod';
import {
  mysqlOptionsSchema,
  requireTablestatsLimit,
} from '../../mysql/MySQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentials,
} from '../../ServiceForm.schema';
import { azureSchema } from '../AzureServiceForm.schema';

export const azureMysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...mysqlOptionsSchema.shape,
    ...azureSchema.shape,
  })
  .superRefine(requireCredentials)
  .superRefine(requireTablestatsLimit);
