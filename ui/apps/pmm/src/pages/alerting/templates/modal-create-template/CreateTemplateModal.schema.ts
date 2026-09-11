import { z } from 'zod';
import { Messages } from './CreateTemplateModal.messages';

export const createTemplateSchema = z.object({
  yaml: z.string().trim().min(1, { message: Messages.validation.required }),
});

export type CreateTemplateFormValues = z.infer<typeof createTemplateSchema>;
