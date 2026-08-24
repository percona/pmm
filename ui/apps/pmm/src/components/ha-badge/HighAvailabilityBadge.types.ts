import type { ChipProps } from '@mui/material/Chip';
import type { HAHealth } from 'types/ha.types';

export interface HighAvailabilityBadgeProps extends ChipProps {
  health: HAHealth;
}
