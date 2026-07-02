import Chip from '@mui/material/Chip';
import { FC } from 'react';
import { HIGH_AVAILABILITY_BADGE_HEALTH } from './HighAvailabilityBadge.constants';
import { useTheme } from '@mui/material/styles';
import { getStyles } from './HighAvailabilityBadge.styles';
import Stack from '@mui/material/Stack';
import { HighAvailabilityBadgeProps } from './HighAvailabilityBadge.types';

const HighAvailabilityBadge: FC<HighAvailabilityBadgeProps> = ({
  health,
  ...props
}) => {
  const theme = useTheme();
  const styles = getStyles(theme);

  if (health === 'unknown') {
    return null;
  }

  return (
    <Stack flex={8} alignItems="flex-start">
      <Chip
        data-testid="ha-badge"
        color="warning"
        variant={
          health === 'unreachable' || health === 'critical'
            ? 'filled'
            : 'outlined'
        }
        label={HIGH_AVAILABILITY_BADGE_HEALTH[health]}
        sx={styles[health]}
        {...props}
      />
    </Stack>
  );
};

export default HighAvailabilityBadge;
