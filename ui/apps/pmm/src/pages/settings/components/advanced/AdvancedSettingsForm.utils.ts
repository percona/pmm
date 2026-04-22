import { Settings } from 'types/settings.types';
import { AdvancedSettingsFormValues } from './AdvancedSettingsForm.schema';
import { DEFAULT_DATA_RETENTION } from './Advanced.constants';
import { convertCheckIntervalsToHours, convertSecondsToDays } from './Advanced.utils';

export const toFormValues = (settings: Settings): AdvancedSettingsFormValues => ({
  retention: String(convertSecondsToDays(settings.dataRetention ?? DEFAULT_DATA_RETENTION) || '1'),
  telemetry: settings.telemetryEnabled,
  updates: settings.updatesEnabled,
  alerting: settings.alertingEnabled,
  backup: settings.backupManagementEnabled,
  enableInternalPgQan: settings.enableInternalPgQan,
  publicAddress: settings.pmmPublicAddress,
  stt: settings.advisorEnabled,
  ...convertCheckIntervalsToHours(settings.advisorRunIntervals),
  azureDiscover: settings.azurediscoverEnabled,
  accessControl: settings.enableAccessControl,
});
