import { z } from 'zod';
import {
  commonOptionsSchema,
  labelsSchema,
  portText,
  requiredText,
} from '../ServiceForm.schema';

export const haproxySchema = z.object({
  serviceName: z.string(),
  address: requiredText,
  port: portText,
  username: z.string(),
  password: z.string(),
  ...labelsSchema.shape,
  ...commonOptionsSchema.shape,
});
