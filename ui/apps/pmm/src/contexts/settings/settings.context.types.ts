import { FrontendSettings, Settings } from 'types/settings.types';

export interface CombinedSettings extends Settings {
  frontend: FrontendSettings;
  newUIEnabled?: boolean;
}

export interface SettingsContextProps {
  isLoading: boolean;
  settings: CombinedSettings | null;
}
