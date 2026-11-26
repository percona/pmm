import type { AlertSeverity, AlertSeverityBand } from 'types/alerting.types';
import DangerousIcon from '@mui/icons-material/Dangerous';
import WarningIcon from '@mui/icons-material/Warning';
import { ReactElement } from 'react';
import { Icon } from 'components/icon';

export const SEVERITY_BAND_MAP: Record<
  Exclude<AlertSeverity, 'unknown'>,
  AlertSeverityBand
> = {
  emergency: 'critical',
  alert: 'critical',
  critical: 'critical',
  error: 'error',
  warning: 'warning',
  notice: 'info',
  info: 'info',
  informational: 'info',
  debug: 'info',
};

// Key order is the severity order, most severe first.
export const SEVERITY_BAND_CONFIG: Record<
  AlertSeverityBand,
  { label: string; icon: ReactElement }
> = {
  critical: { label: 'Critical', icon: <DangerousIcon color="error" /> },
  error: { label: 'Error', icon: <Icon name="emergency-home" color="error" /> },
  warning: { label: 'Warning', icon: <WarningIcon color="warning" /> },
  info: { label: 'Info', icon: <Icon name="chat-info-outlined" /> },
};

export const SEVERITY_BAND_ORDER = Object.keys(
  SEVERITY_BAND_CONFIG
) as AlertSeverityBand[];
