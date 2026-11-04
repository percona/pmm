import { z } from 'zod';
import {
  mysqlOptionsSchema,
  requireTablestatsLimit,
} from '../../mysql/MySQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentialsWithoutTls,
} from '../../ServiceForm.schema';
import { rdsSchema } from '../RdsServiceForm.schema';

export const rdsMysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...mysqlOptionsSchema.shape,
    ...rdsSchema.shape,
  })
  .superRefine(requireCredentialsWithoutTls)
  .superRefine(requireTablestatsLimit);
