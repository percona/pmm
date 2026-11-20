import type { z } from 'zod';
import type { proxysqlSchema } from './ProxySQLForm.schema';

export type ProxySQLFormValues = z.infer<typeof proxysqlSchema>;
