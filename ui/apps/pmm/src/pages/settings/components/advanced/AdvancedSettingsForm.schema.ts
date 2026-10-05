import { z } from 'zod';
import { Messages } from '../../Settings.messages';
import {
  MAX_DAYS,
  MIN_DAYS,
  MIN_STT_CHECK_INTERVAL,
} from './Advanced.constants';

const { required, retentionRange, intervalMin } = Messages.advanced.validation;

// The value loaded from the server is accepted as it is: it is only sent once the user changes it,
// and the pmm-ha chart may set one outside the range a user can type here.
const retentionField = (loaded?: string) =>
  z
    .string()
    .refine((v) => v === loaded || (v !== '' && !isNaN(parseFloat(v))), {
      message: required,
    })
    .refine(
      (v) => {
        if (v === loaded) return true;
        const n = parseFloat(v);
        return n >= MIN_DAYS && n <= MAX_DAYS;
      },
      { message: retentionRange(MIN_DAYS, MAX_DAYS) }
    );

const intervalFields = [
  'rareInterval',
  'standardInterval',
  'frequentInterval',
] as const;

export const createAdvancedSettingsSchema = (loadedRetention?: string) =>
  z
    .object({
      retention: retentionField(loadedRetention),
      telemetry: z.boolean(),
      updates: z.boolean(),
      alerting: z.boolean(),
      backup: z.boolean(),
      enableInternalPgQan: z.boolean(),
      publicAddress: z.string(),
      stt: z.boolean(),
      rareInterval: z.string(),
      standardInterval: z.string(),
      frequentInterval: z.string(),
      azureDiscover: z.boolean(),
      accessControl: z.boolean(),
    })
    .superRefine((data, ctx) => {
      if (!data.stt) return;
      for (const field of intervalFields) {
        const v = data[field];
        const n = parseFloat(v);
        if (v === '' || isNaN(n)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: required,
            path: [field],
          });
        } else if (n < MIN_STT_CHECK_INTERVAL) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: intervalMin(MIN_STT_CHECK_INTERVAL),
            path: [field],
          });
        }
      }
    });

export type AdvancedSettingsFormValues = z.infer<
  ReturnType<typeof createAdvancedSettingsSchema>
>;
