import type { z } from 'zod';
import type { rdsPostgresqlSchema } from './RdsPostgreSQLForm.schema';

export type RdsPostgreSQLFormValues = z.infer<typeof rdsPostgresqlSchema>;
