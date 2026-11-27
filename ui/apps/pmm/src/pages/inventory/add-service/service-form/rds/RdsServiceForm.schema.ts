import { z } from 'zod';
import { requiredText } from '../ServiceForm.schema';

export const rdsSchema = z.object({
  instanceId: requiredText,
  disableBasicMetrics: z.boolean(),
  disableEnhancedMetrics: z.boolean(),
});
