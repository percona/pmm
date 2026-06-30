import { MetricsResolutions } from 'types/settings.types';
import {
  defaultResolutions,
  resolutionOptions,
  ResolutionPreset,
} from './MetricsResolution.constants';

const replaceS = (r: string) => r.replace(/s$/, '');

export const removeUnits = (
  r: MetricsResolutions
): { lr: string; mr: string; hr: string } => ({
  lr: replaceS(r.lr),
  mr: replaceS(r.mr),
  hr: replaceS(r.hr),
});

export const addUnits = (r: {
  lr: string;
  mr: string;
  hr: string;
}): MetricsResolutions => ({
  lr: `${r.lr}s`,
  mr: `${r.mr}s`,
  hr: `${r.hr}s`,
});

const resolutionsEqual = (a: MetricsResolutions, b: MetricsResolutions) =>
  a.hr === b.hr && a.mr === b.mr && a.lr === b.lr;

export const getResolutionPreset = (
  metricsResolutions: MetricsResolutions | undefined
): ResolutionPreset => {
  if (!metricsResolutions) return 'custom';
  const index = defaultResolutions.findIndex((r) =>
    resolutionsEqual(r, metricsResolutions)
  );
  return index !== -1
    ? (resolutionOptions[index].value as ResolutionPreset)
    : 'custom';
};
