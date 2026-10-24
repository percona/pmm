import { describe, it, expect } from 'vitest';
import { Severity } from 'types/severity.types';
import { AdvisorsFormValues, advisorsSchema } from './AdvisorsForm.schema';

const valid: AdvisorsFormValues = {
  stt: true,
  advisorRetention: '30',
  advisorNotifications: false,
  advisorSeverities: [Severity.error],
  advisorNotificationEmails: '',
};

const errorFor = (values: AdvisorsFormValues, field: string) => {
  const result = advisorsSchema.safeParse(values);
  if (result.success) {
    return undefined;
  }
  return result.error.issues.find((issue) => issue.path[0] === field)?.message;
};

describe('advisorsSchema notification severities', () => {
  const notifying = {
    ...valid,
    advisorNotifications: true,
    advisorNotificationEmails: 'dba@example.com',
  };

  it('requires at least one severity while notifications are on', () => {
    expect(
      errorFor({ ...notifying, advisorSeverities: [] }, 'advisorSeverities')
    ).toBe('Select at least one severity, or turn Advisor notifications off');
  });

  it('allows no severity when notifications are off', () => {
    expect(
      advisorsSchema.safeParse({ ...valid, advisorSeverities: [] }).success
    ).toBe(true);
  });
});
