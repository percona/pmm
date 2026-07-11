import { z } from 'zod';
import { Messages } from './EditTemplateModal.messages';

export const editTemplateSchema = z.object({
  yaml: z.string().trim().min(1, { message: Messages.validation.required }),
});

export type EditTemplateFormValues = z.infer<typeof editTemplateSchema>;
