import type { InputHTMLAttributes } from 'react';
import { Messages } from '../../Settings.messages';
import { MetricsResolutions } from 'types/settings.types';

export const RESOLUTION_PRESETS = [
  'rare',
  'standard',
  'frequent',
  'custom',
] as const;
export type ResolutionPreset = (typeof RESOLUTION_PRESETS)[number];

const radioInputProps = (testId: string) =>
  ({ 'data-testid': testId }) as InputHTMLAttributes<HTMLInputElement>;

export const resolutionOptions: {
  value: ResolutionPreset;
  label: string;
  radioProps: { inputProps: InputHTMLAttributes<HTMLInputElement> };
}[] = [
  {
    value: 'rare',
    label: Messages.metrics.options.rare,
    radioProps: {
      inputProps: radioInputProps('metrics-resolution-radio-rare'),
    },
  },
  {
    value: 'standard',
    label: Messages.metrics.options.standard,
    radioProps: {
      inputProps: radioInputProps('metrics-resolution-radio-standard'),
    },
  },
  {
    value: 'frequent',
    label: Messages.metrics.options.frequent,
    radioProps: {
      inputProps: radioInputProps('metrics-resolution-radio-frequent'),
    },
  },
  {
    value: 'custom',
    label: Messages.metrics.options.custom,
    radioProps: {
      inputProps: radioInputProps('metrics-resolution-radio-custom'),
    },
  },
];

export const defaultResolutions: MetricsResolutions[] = [
  { hr: '60s', mr: '180s', lr: '300s' },
  { hr: '5s', mr: '10s', lr: '60s' },
  { hr: '1s', mr: '5s', lr: '30s' },
];

export const RESOLUTION_MIN = 1;
export const RESOLUTION_MAX = 1000000000;

export const DEFAULT_METRICS: MetricsResolutions = {
  hr: '5s',
  mr: '10s',
  lr: '60s',
};
