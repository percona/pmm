import type { z } from 'zod';
import type { haproxySchema } from './HAProxyForm.schema';

export type HAProxyFormValues = z.infer<typeof haproxySchema>;
