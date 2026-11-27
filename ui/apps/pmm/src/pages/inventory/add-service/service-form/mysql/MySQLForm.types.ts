import type { z } from 'zod';
import type { mysqlOptionsSchema, mysqlSchema } from './MySQLForm.schema';

export enum TablestatOption {
  disabled = 'disabled',
  default = 'default',
  custom = 'custom',
}

export type MySQLOptionsFormValues = z.infer<typeof mysqlOptionsSchema>;
export type MySQLFormValues = z.infer<typeof mysqlSchema>;
