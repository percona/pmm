import { z } from 'zod';
import { Messages } from './ExportDataset.messages';
import { toWallClock } from './ExportDataset.time';

// Same character classes as PMM Server's validators.ValidatePassword
const isStrongPassword = (password: string) =>
  password.length >= 8 &&
  /\p{L}/u.test(password) &&
  /\p{N}/u.test(password) &&
  /[\p{P}\p{S}]/u.test(password);

const exportDatasetFields = z.object({
  serviceScope: z.enum(['all', 'selected']),
  serviceNames: z.array(z.string()),
  startTime: z.date(),
  endTime: z.date(),
  exportQan: z.boolean(),
  ignoreLoad: z.boolean(),
  enableEncryption: z.boolean(),
  encryptionPassword: z.string(),
});

// Dates hold wall-clock time in the PMM timezone; see ExportDataset.time.ts
export const createExportDatasetSchema = (timezone: string) =>
  exportDatasetFields.superRefine((values, context) => {
    const now = toWallClock(new Date(), timezone);
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

export type ExportDatasetFormValues = z.infer<typeof exportDatasetFields>;
