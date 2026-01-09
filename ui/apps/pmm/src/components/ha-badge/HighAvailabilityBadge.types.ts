import { ChipProps } from '@mui/material/Chip';
import { HAHealth } from 'types/ha.types';

export interface HighAvailabilityBadgeProps extends ChipProps {
  health: HAHealth;
}
