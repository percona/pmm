import { z } from 'zod';
import { Severity } from 'types/severity.types';
import { Messages } from '../../Settings.messages';
import { MAX_DAYS, MIN_DAYS } from '../advanced/Advanced.constants';
import { MIN_ADVISOR_CHECK_INTERVAL } from './Advisors.constants';
import { splitEmailAddresses } from './Advisors.utils';

const {
  required,
  retentionRange,
  intervalMin,
  emailsRequired,
  emailInvalid,
  emailsMax,
} = Messages.advisors.validation;

// mirrors maxAdvisorNotificationEmailAddresses in managed/models/settings_helpers.go
const MAX_NOTIFICATION_EMAILS = 20;

const intervalFields = [
  'rareInterval',
  'standardInterval',
  'frequentInterval',
] as const;

export const advisorsSchema = z
  .object({
    stt: z.boolean(),
    rareInterval: z.string(),
    standardInterval: z.string(),
    frequentInterval: z.string(),
    advisorRetention: z.string(),
    advisorNotifications: z.boolean(),
    advisorSeverityThreshold: z.nativeEnum(Severity),
    advisorNotificationEmails: z.string(),
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
      } else if (n < MIN_ADVISOR_CHECK_INTERVAL) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: intervalMin(MIN_ADVISOR_CHECK_INTERVAL),
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

    if (!data.advisorNotifications) return;

    // the API rejects notifications with no recipients, so catch it before submitting
    const emails = splitEmailAddresses(data.advisorNotificationEmails);
    if (emails.length === 0) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: emailsRequired,
        path: ['advisorNotificationEmails'],
      });
      return;
    }
    if (emails.length > MAX_NOTIFICATION_EMAILS) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: emailsMax(MAX_NOTIFICATION_EMAILS),
        path: ['advisorNotificationEmails'],
      });
      return;
    }
    const invalid = emails.find(
      (email) => !z.string().email().safeParse(email).success
    );
    if (invalid) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: emailInvalid(invalid),
        path: ['advisorNotificationEmails'],
      });
    }
  });

export type AdvisorsFormValues = z.infer<typeof advisorsSchema>;
