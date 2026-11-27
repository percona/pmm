import type { z } from 'zod';
import type { azureMysqlSchema } from './AzureMySQLForm.schema';

export type AzureMySQLFormValues = z.infer<typeof azureMysqlSchema>;
