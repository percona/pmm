import { z } from 'zod';
import { Messages } from '../../Settings.messages';
import { MAX_DAYS, MIN_DAYS } from './Advanced.constants';

const { required, retentionRange } = Messages.advanced.validation;

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

export const advancedSettingsSchema = z.object({
  retention: retentionField,
  telemetry: z.boolean(),
  updates: z.boolean(),
  alerting: z.boolean(),
  backup: z.boolean(),
  enableInternalPgQan: z.boolean(),
  publicAddress: z.string(),
  azureDiscover: z.boolean(),
  accessControl: z.boolean(),
});

export type AdvancedSettingsFormValues = z.infer<typeof advancedSettingsSchema>;
