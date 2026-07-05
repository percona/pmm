import { describe, it, expect } from 'vitest';
import { Settings } from 'types/settings.types';
import { Severity } from 'types/severity.types';
import { toFormValues, toPayload } from './AdvancedSettingsForm.utils';

const TEST_SETTINGS: Settings = {
  updatesEnabled: true,
  telemetryEnabled: true,
  advisorEnabled: true,
  alertingEnabled: true,
  pmmPublicAddress: 'pmm.local',
  backupManagementEnabled: false,
  azurediscoverEnabled: false,
  enableAccessControl: false,
  dataRetention: '2592000s',
  advisorHistoryRetention: '1209600s',
  advisorNotificationsEnabled: true,
  advisorNotificationSeverityThreshold: Severity.warning,
};

describe('toFormValues', () => {
  it('maps advisor notification settings', () => {
    const values = toFormValues(TEST_SETTINGS);

    expect(values.advisorRetention).toBe('14');
    expect(values.advisorNotifications).toBe(true);
    expect(values.advisorSeverityThreshold).toBe(Severity.warning);
  });

  it('falls back to defaults when advisor settings are missing', () => {
    const values = toFormValues({
      ...TEST_SETTINGS,
      advisorHistoryRetention: undefined,
      advisorNotificationsEnabled: undefined,
      advisorNotificationSeverityThreshold: undefined,
    });

    expect(values.advisorRetention).toBe('30');
    expect(values.advisorNotifications).toBe(false);
    expect(values.advisorSeverityThreshold).toBe(Severity.error);
  });

  it('treats unspecified severity threshold as the default', () => {
    const values = toFormValues({
      ...TEST_SETTINGS,
      advisorNotificationSeverityThreshold: Severity.unspecified,
    });

    expect(values.advisorSeverityThreshold).toBe(Severity.error);
  });
});

describe('toPayload', () => {
  it('maps advisor notification settings back to the API shape', () => {
    const payload = toPayload(toFormValues(TEST_SETTINGS));

    expect(payload.advisorHistoryRetention).toBe('1209600s');
    expect(payload.enableAdvisorNotifications).toBe(true);
    expect(payload.advisorNotificationSeverityThreshold).toBe(
      Severity.warning
    );
  });
});
