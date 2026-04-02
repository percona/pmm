import { Settings } from 'types/settings.types';

export interface MetricsResolutionFormProps {
  settings: Settings;
}

export interface MetricsResolutionFormValues {
  preset: 'rare' | 'standard' | 'frequent' | 'custom';
  lr: string;
  mr: string;
  hr: string;
}
