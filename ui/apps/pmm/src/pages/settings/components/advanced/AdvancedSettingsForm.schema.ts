import { z } from 'zod';
import { Messages } from '../../Settings.messages';
import { MAX_DAYS, MIN_DAYS } from './Advanced.constants';

const { required, retentionRange } = Messages.advanced.validation;

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

export const createAdvancedSettingsSchema = (loadedRetention?: string) =>
  z.object({
    retention: retentionField(loadedRetention),
    telemetry: z.boolean(),
    updates: z.boolean(),
    alerting: z.boolean(),
    backup: z.boolean(),
    enableInternalPgQan: z.boolean(),
    publicAddress: z.string(),
    azureDiscover: z.boolean(),
    accessControl: z.boolean(),
  });

export type AdvancedSettingsFormValues = z.infer<
  ReturnType<typeof createAdvancedSettingsSchema>
>;
