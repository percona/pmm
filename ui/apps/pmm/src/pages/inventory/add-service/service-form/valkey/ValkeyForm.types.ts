import type { z } from 'zod';
import type { valkeySchema } from './ValkeyForm.schema';

export type ValkeyFormValues = z.infer<typeof valkeySchema>;
