import Chip, { ChipProps } from '@mui/material/Chip';
import { FC } from 'react';
import { HIGH_AVAILABILITY_BADGE_HEALTH } from './HighAvailabilityBadge.constants';
import { useTheme } from '@mui/material/styles';
import { getStyles } from './HighAvailabilityBadge.styles';
import { HighAvailabilityHealth } from 'types/high-availability.types';
import Stack from '@mui/material/Stack';

interface Props extends ChipProps {
  health: HighAvailabilityHealth;
}

const HighAvailabilityBadge: FC<Props> = ({ health, ...props }) => {
  const theme = useTheme();
  const styles = getStyles(theme);

  return (
    <Stack flex={8} alignItems="flex-start">
      <Chip
        color="warning"
        variant={health !== 'down' ? 'outlined' : 'filled'}
        label={HIGH_AVAILABILITY_BADGE_HEALTH[health]}
        sx={styles[health]}
        {...props}
      />
    </Stack>
  );
};

export default HighAvailabilityBadge;
