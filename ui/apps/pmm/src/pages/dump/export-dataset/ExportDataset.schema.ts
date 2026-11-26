import { z } from 'zod';
import { Messages } from './ExportDataset.messages';

const isStrongPassword = (password: string) =>
  password.length >= 8 &&
  /[A-Za-z]/.test(password) &&
  /\d/.test(password) &&
  /[^A-Za-z0-9]/.test(password);

export const exportDatasetSchema = z
  .object({
    serviceScope: z.enum(['all', 'selected']),
    serviceNames: z.array(z.string()),
    startTime: z.date(),
    endTime: z.date(),
    exportQan: z.boolean(),
    ignoreLoad: z.boolean(),
    enableEncryption: z.boolean(),
    encryptionPassword: z.string(),
  })
  .superRefine((values, context) => {
    const now = new Date();
    if (values.serviceScope === 'selected' && !values.serviceNames.length) {
      context.addIssue({
        code: 'custom',
        path: ['serviceNames'],
        message: Messages.validation.servicesRequired,
      });
    }
    if (values.startTime >= values.endTime) {
      context.addIssue({
        code: 'custom',
        path: ['startTime'],
        message: Messages.validation.validRange,
      });
    }
    if (values.startTime > now) {
      context.addIssue({
        code: 'custom',
        path: ['startTime'],
        message: Messages.validation.futureDate,
      });
    }
    if (values.endTime > now) {
      context.addIssue({
        code: 'custom',
        path: ['endTime'],
        message: Messages.validation.futureDate,
      });
    }
    if (
      values.enableEncryption &&
      !isStrongPassword(values.encryptionPassword)
    ) {
      context.addIssue({
        code: 'custom',
        path: ['encryptionPassword'],
        message: Messages.passwordRules,
      });
    }
  });

export type ExportDatasetFormValues = z.infer<typeof exportDatasetSchema>;
