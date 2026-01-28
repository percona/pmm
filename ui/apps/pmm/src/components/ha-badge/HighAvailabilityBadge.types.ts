import { ChipProps } from '@mui/material';
import { HAHealth } from 'types/ha.types';

export interface HighAvailabilityBadgeProps extends ChipProps {
  health: HAHealth;
}
