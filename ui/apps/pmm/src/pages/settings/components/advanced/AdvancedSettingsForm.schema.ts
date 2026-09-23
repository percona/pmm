import { z } from 'zod';
import { Messages } from '../../Settings.messages';
import {
  MAX_DAYS,
  MIN_DAYS,
  MIN_STT_CHECK_INTERVAL,
} from './Advanced.constants';

const { required, retentionRange, intervalMin } = Messages.advanced.validation;

const retentionField = z
  .string()
  .refine((v) => v !== '' && !isNaN(parseFloat(v)), { message: required })
  .refine(
    (v) => {
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

// While high availability locks retention, the field is not submitted and the chart may hold a
// value outside the range accepted here, so validating it would block the rest of the form.
export const createAdvancedSettingsSchema = (retentionLocked: boolean) =>
  z
    .object({
      retention: retentionLocked ? z.string() : retentionField,
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

export const advancedSettingsSchema = createAdvancedSettingsSchema(false);

export type AdvancedSettingsFormValues = z.infer<typeof advancedSettingsSchema>;
