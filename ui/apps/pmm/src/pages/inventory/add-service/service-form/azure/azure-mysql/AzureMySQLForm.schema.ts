import { z } from 'zod';
import {
  mysqlOptionsSchema,
  requireTablestatsLimit,
} from '../../mysql/MySQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentialsWithoutTls,
} from '../../ServiceForm.schema';
import { azureSchema } from '../AzureServiceForm.schema';

export const azureMysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...mysqlOptionsSchema.shape,
    ...azureSchema.shape,
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(requireTablestatsLimit);
