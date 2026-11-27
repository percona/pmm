import type { z } from 'zod';
import type { azurePostgresqlSchema } from './AzurePostgreSQLForm.schema';

export type AzurePostgreSQLFormValues = z.infer<typeof azurePostgresqlSchema>;
