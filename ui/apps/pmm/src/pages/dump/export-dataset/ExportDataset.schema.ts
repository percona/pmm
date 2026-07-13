import { z } from 'zod';
import { Messages } from './ExportDataset.messages';

export const exportDatasetSchema = z
  .object({
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
    if (!values.enableEncryption) {
      return;
    }

    const password = values.encryptionPassword;
    const validations = [
      [password.length > 0, Messages.validation.passwordRequired],
      [password.length >= 8, Messages.validation.passwordLength],
      [/[A-Za-z]/.test(password), Messages.validation.passwordLetter],
      [/\d/.test(password), Messages.validation.passwordNumber],
      [/[^A-Za-z0-9]/.test(password), Messages.validation.passwordSpecial],
    ] as const;

    const failed = validations.find(([valid]) => !valid);
    if (failed) {
      context.addIssue({
        code: 'custom',
        path: ['encryptionPassword'],
        message: failed[1],
      });
    }
  });

export type ExportDatasetFormValues = z.infer<typeof exportDatasetSchema>;
