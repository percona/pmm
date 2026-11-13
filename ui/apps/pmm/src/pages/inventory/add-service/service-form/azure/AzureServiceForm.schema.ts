import { z } from 'zod';

export const azureSchema = z.object({
  azureDatabaseExporter: z.boolean(),
});
