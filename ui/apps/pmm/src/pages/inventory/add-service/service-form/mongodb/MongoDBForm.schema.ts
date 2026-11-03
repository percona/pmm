import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  maxQueryLengthSchema,
  mongodbTlsCertificatesSchema,
  requireCredentialsWithoutTls,
} from '../ServiceForm.schema';

export const mongodbSchema = z
  .object({
    ...databaseServiceSchema.shape,
    ...mongodbTlsCertificatesSchema.shape,
    ...disableCollectorsSchema.shape,
    ...maxQueryLengthSchema.shape,
    qanMongodbProfiler: z.boolean(),
  })
  .superRefine(requireCredentialsWithoutTls);
