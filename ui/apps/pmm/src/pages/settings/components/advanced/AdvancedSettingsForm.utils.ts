import { Settings, UpdateSettingsPayload } from 'types/settings.types';
import { AdvancedSettingsFormValues } from './AdvancedSettingsForm.schema';
import { DEFAULT_DATA_RETENTION, SECONDS_IN_DAY } from './Advanced.constants';
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
  azureDiscover: settings.azurediscoverEnabled,
  accessControl: settings.enableAccessControl,
});

// Retention is sent only when it differs from the value loaded from the server. Echoing an
// unchanged value back can be stale, and in high availability the server refuses any retention
// that differs from the stored one, whether or not the HA status has reached the form yet.
export const toPayload = (
  values: AdvancedSettingsFormValues,
  loadedRetention?: string
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
    ...(values.retention === loadedRetention ? {} : { dataRetention }),
    pmmPublicAddress: values.publicAddress,
    enableTelemetry: values.telemetry,
    enableUpdates: values.updates,
    enableAlerting: values.alerting,
    enableBackupManagement: values.backup,
    enableInternalPgQan: values.enableInternalPgQan,
    enableAdvisor: values.stt,
    advisorRunIntervals,
    enableAzurediscover: values.azureDiscover,
    enableAccessControl: values.accessControl,
  };
};
