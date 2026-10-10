import { describe, it, expect } from 'vitest';
import { Severity } from 'types/severity.types';
import { AdvisorsFormValues, advisorsSchema } from './AdvisorsForm.schema';

const valid: AdvisorsFormValues = {
  stt: true,
  rareInterval: '78',
  standardInterval: '24',
  frequentInterval: '4',
  advisorRetention: '30',
  advisorNotifications: false,
  advisorSeverityThreshold: Severity.error,
  advisorNotificationEmails: '',
};

const errorFor = (values: AdvisorsFormValues, field: string) => {
  const result = advisorsSchema.safeParse(values);
  if (result.success) {
    return undefined;
  }
  return result.error.issues.find((issue) => issue.path[0] === field)?.message;
};

describe('advisorsSchema check run intervals', () => {
  it('accepts whole hours', () => {
    expect(advisorsSchema.safeParse(valid).success).toBe(true);
  });

  it('rejects a fractional interval', () => {
    expect(
      errorFor({ ...valid, standardInterval: '24.5' }, 'standardInterval')
    ).toBe('Use whole hours');
  });

  it('rejects an interval below one hour', () => {
    expect(
      errorFor({ ...valid, frequentInterval: '0.1' }, 'frequentInterval')
    ).toBe('Min 1');
  });

  it('skips interval checks when advisors are off', () => {
    expect(
      advisorsSchema.safeParse({
        ...valid,
        stt: false,
        standardInterval: '24.5',
      }).success
    ).toBe(true);
  });
});
