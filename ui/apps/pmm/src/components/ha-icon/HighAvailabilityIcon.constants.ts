import { IconName } from 'components/icon/Icon.types';
import { HAHealth } from 'types/ha.types';

export const HA_ICON_MAP: Partial<Record<HAHealth, IconName>> = {
  degraded: 'status-at-risk',
  critical: 'status-down',
  unreachable: 'status-down',
};
