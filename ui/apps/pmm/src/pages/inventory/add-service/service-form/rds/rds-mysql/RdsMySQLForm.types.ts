import type { z } from 'zod';
import type { rdsMysqlSchema } from './RdsMySQLForm.schema';

export type RdsMySQLFormValues = z.infer<typeof rdsMysqlSchema>;
