import { IconName } from 'components/icon/Icon.types';
import { AlertSeverity } from 'types/alerting.types';

export const SEVERITY_ICON_MAP: Partial<Record<AlertSeverity, IconName>> = {
  critical: 'status-down',
  warning: 'status-at-risk',
};
