import { Settings, UpdateSettingsPayload } from 'types/settings.types';
import { Severity } from 'types/severity.types';
import { AdvancedSettingsFormValues } from './AdvancedSettingsForm.schema';
import {
  DEFAULT_ADVISOR_RETENTION,
  DEFAULT_DATA_RETENTION,
  SECONDS_IN_DAY,
} from './Advanced.constants';
import {
  convertCheckIntervalsToHours,
  convertHoursStringToSeconds,
  convertSecondsToDays,
} from './Advanced.utils';

export const toFormValues = (
  settings: Settings
): AdvancedSettingsFormValues => ({
  retention: String(
    convertSecondsToDays(settings.dataRetention ?? DEFAULT_DATA_RETENTION) ||
      '1'
  ),
  telemetry: settings.telemetryEnabled,
  updates: settings.updatesEnabled,
  alerting: settings.alertingEnabled,
  backup: settings.backupManagementEnabled,
  enableInternalPgQan: settings.enableInternalPgQan ?? false,
  publicAddress: settings.pmmPublicAddress,
  stt: settings.advisorEnabled,
  ...convertCheckIntervalsToHours(settings.advisorRunIntervals),
  advisorRetention: String(
    convertSecondsToDays(
      settings.advisorHistoryRetention ?? DEFAULT_ADVISOR_RETENTION
    ) || '1'
  ),
  advisorNotifications: settings.advisorNotificationsEnabled ?? false,
  advisorSeverityThreshold:
    settings.advisorNotificationSeverityThreshold &&
    settings.advisorNotificationSeverityThreshold !== Severity.unspecified
      ? settings.advisorNotificationSeverityThreshold
      : Severity.error,
  azureDiscover: settings.azurediscoverEnabled,
  accessControl: settings.enableAccessControl,
});

export const toPayload = (
  values: AdvancedSettingsFormValues
): UpdateSettingsPayload => {
  const dataRetention = `${Math.round(parseFloat(values.retention) * SECONDS_IN_DAY)}s`;
  const advisorRunIntervals = values.stt
    ? {
        rareInterval: `${convertHoursStringToSeconds(values.rareInterval)}s`,
        standardInterval: `${convertHoursStringToSeconds(values.standardInterval)}s`,
        frequentInterval: `${convertHoursStringToSeconds(values.frequentInterval)}s`,
      }
    : undefined;

  return {
    dataRetention,
    pmmPublicAddress: values.publicAddress,
    enableTelemetry: values.telemetry,
    enableUpdates: values.updates,
    enableAlerting: values.alerting,
    enableBackupManagement: values.backup,
    enableInternalPgQan: values.enableInternalPgQan,
    enableAdvisor: values.stt,
    advisorRunIntervals,
    advisorHistoryRetention: `${Math.round(parseFloat(values.advisorRetention)) * SECONDS_IN_DAY}s`,
    enableAdvisorNotifications: values.advisorNotifications,
    advisorNotificationSeverityThreshold: values.advisorSeverityThreshold,
    enableAzurediscover: values.azureDiscover,
    enableAccessControl: values.accessControl,
  };
};
