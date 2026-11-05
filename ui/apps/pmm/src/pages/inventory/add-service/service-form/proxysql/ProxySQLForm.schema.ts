import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  requireCredentials,
} from '../ServiceForm.schema';

export const proxysqlSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...disableCollectorsSchema.shape,
  })
  .superRefine(requireCredentials);
