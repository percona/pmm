import type { AlertSeverityBand } from 'types/alerting.types';
import { SEVERITY_BAND_MAP } from './AlertSeverity.constants';

export const getSeverityBand = (
  severity?: string
): AlertSeverityBand | undefined => {
  const level = severity?.trim().toLowerCase();

  // Own keys only, so a label like 'constructor' does not hit the prototype.
  if (
    !level ||
    !Object.prototype.hasOwnProperty.call(SEVERITY_BAND_MAP, level)
  ) {
    return undefined;
  }

  return SEVERITY_BAND_MAP[level as keyof typeof SEVERITY_BAND_MAP];
};
