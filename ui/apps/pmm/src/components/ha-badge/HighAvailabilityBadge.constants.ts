import { HAHealth } from 'types/ha.types';

export const HIGH_AVAILABILITY_BADGE_HEALTH: Record<HAHealth, string> = {
  healthy: 'Healthy',
  degraded: 'Degraded',
  critical: 'Critical',
  unreachable: 'Unreachable',
  unknown: 'Unknown',
};
