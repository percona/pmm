import { z } from 'zod';
import {
  databaseServiceSchema,
  disableCollectorsSchema,
  maxQueryLengthSchema,
  mongodbTlsCertificatesSchema,
} from '../ServiceForm.schema';

// MongoDB may run without authentication, so the credentials stay optional.
export const mongodbSchema = z.object({
  ...databaseServiceSchema.shape,
  ...mongodbTlsCertificatesSchema.shape,
  ...disableCollectorsSchema.shape,
  ...maxQueryLengthSchema.shape,
  qanMongodbProfiler: z.boolean(),
});
