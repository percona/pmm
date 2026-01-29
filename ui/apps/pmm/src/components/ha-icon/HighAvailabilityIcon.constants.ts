import { IconName } from 'components/icon/Icon.types';
import { HAHealth } from 'types/ha.types';

export const HA_ICON_MAP: Record<Exclude<HAHealth, 'healthy'>, IconName> = {
  degraded: 'status-at-risk',
  critical: 'status-down',
  down: 'status-down',
};
