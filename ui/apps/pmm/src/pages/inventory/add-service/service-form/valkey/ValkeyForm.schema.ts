import { z } from 'zod';
import {
  databaseServiceSchema,
  tlsCertificatesSchema,
} from '../ServiceForm.schema';

// Valkey may run without authentication, so the credentials stay optional.
export const valkeySchema = z.object({
  ...databaseServiceSchema.shape,
  ...tlsCertificatesSchema.shape,
});
