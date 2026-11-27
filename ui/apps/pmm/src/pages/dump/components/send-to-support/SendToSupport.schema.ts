import { z } from 'zod';
import { Messages } from './SendToSupport.messages';

export const sendToSupportSchema = z.object({
  address: z.string().trim().min(1, Messages.required),
  user: z.string().trim().min(1, Messages.required),
  password: z.string().min(1, Messages.required),
  directory: z.string(),
});

export type SendToSupportFormValues = z.infer<typeof sendToSupportSchema>;
