import { z } from 'zod';
import {
  mysqlOptionsSchema,
  requireTablestatsLimit,
} from '../../mysql/MySQLForm.schema';
import {
  databaseServiceSchema,
  requireCredentials,
} from '../../ServiceForm.schema';
import { rdsSchema } from '../RdsServiceForm.schema';

export const rdsMysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...mysqlOptionsSchema.shape,
    ...rdsSchema.shape,
  })
  .superRefine(requireCredentials)
  .superRefine(requireTablestatsLimit);
