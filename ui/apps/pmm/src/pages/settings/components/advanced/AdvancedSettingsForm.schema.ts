import { z } from 'zod';
import { Severity } from 'types/severity.types';
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

export const advancedSettingsSchema = z
  .object({
    retention: retentionField,
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
    advisorRetention: z.string(),
    advisorNotifications: z.boolean(),
    advisorSeverityThreshold: z.nativeEnum(Severity),
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

    // the API requires a multiple of 24h, so whole days only
    const retention = data.advisorRetention;
    const days = parseFloat(retention);
    if (retention === '' || isNaN(days) || !Number.isInteger(days)) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: required,
        path: ['advisorRetention'],
      });
    } else if (days < MIN_DAYS || days > MAX_DAYS) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: retentionRange(MIN_DAYS, MAX_DAYS),
        path: ['advisorRetention'],
      });
    }
  });

export type AdvancedSettingsFormValues = z.infer<typeof advancedSettingsSchema>;
