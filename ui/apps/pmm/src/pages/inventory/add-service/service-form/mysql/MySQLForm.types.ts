import type { z } from 'zod';
import type { mysqlSchema } from './MySQLForm.schema';

export enum TablestatOption {
  disabled = 'disabled',
  default = 'default',
  custom = 'custom',
}

export type MySQLFormValues = z.infer<typeof mysqlSchema>;
