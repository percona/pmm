import { Settings } from 'types/settings.types';

export interface AdvancedSettingsFormProps {
  settings: Settings;
}

export interface AdvancedSettingsFormValues {
  retention: string;
  telemetry: boolean;
  updates: boolean;
  alerting: boolean;
  backup: boolean;
  enableInternalPgQan: boolean;
  publicAddress: string;
  stt: boolean;
  rareInterval: string;
  standardInterval: string;
  frequentInterval: string;
  azureDiscover: boolean;
  accessControl: boolean;
}

