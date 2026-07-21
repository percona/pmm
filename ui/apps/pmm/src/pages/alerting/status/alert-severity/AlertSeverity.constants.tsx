import type { TypographyProps } from '@mui/material/Typography';
import type { AlertSeverity } from 'types/alerting.types';
import DangerousIcon from '@mui/icons-material/Dangerous';
import WarningIcon from '@mui/icons-material/Warning';
import { ReactElement } from 'react';
import { Icon } from 'components/icon';

export const SEVERITY_COLOR_MAP: Record<
  AlertSeverity,
  TypographyProps['color']
> = {
  error: undefined,
  warning: undefined,
  unknown: undefined,
  emergency: 'error',
  alert: 'error',
  critical: 'error',
  notice: undefined,
  info: undefined,
  debug: undefined,
};

export const SEVERITY_ICON_MAP: Partial<Record<AlertSeverity, ReactElement>> = {
  emergency: <DangerousIcon color="error" />,
  alert: <DangerousIcon color="error" />,
  critical: <DangerousIcon color="error" />,
  error: <Icon name="emergency-home" color="error" />,
  warning: <WarningIcon color="warning" />,
  notice: <Icon name="chat-info-outlined" />,
  info: <Icon name="chat-info-outlined" />,
  debug: <Icon name="chat-info-outlined" />,
};
