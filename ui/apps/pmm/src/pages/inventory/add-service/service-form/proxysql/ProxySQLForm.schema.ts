import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  requireCredentialsWithoutTls,
} from '../ServiceForm.schema';

export const proxysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...disableCollectorsSchema.shape,
  })
  .superRefine(requireCredentialsWithoutTls);
