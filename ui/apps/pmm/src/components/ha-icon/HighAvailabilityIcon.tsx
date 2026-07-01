import { useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import { Icon } from 'components/icon';
import { FC } from 'react';
import { getStyles } from './HighAvailabilityIcon.styles';
import { HighAvailabilityIconProps } from './HighAvailabilityIcon.types';
import { HA_ICON_MAP } from './HighAvailabilityIcon.constants';

const HighAvailabilityIcon: FC<HighAvailabilityIconProps> = ({ health }) => {
  const theme = useTheme();
  const styles = getStyles(theme);
  const haIcon = HA_ICON_MAP[health];

  return (
    <Box sx={{ position: 'relative' }}>
      <Icon data-testid="ha-icon" name="cluster" />
      {haIcon && (
        <Box
          data-testid="ha-health-icon"
          sx={{ position: 'absolute', top: -4, right: -7 }}
        >
          <Icon sx={styles.icon} name={haIcon} />
        </Box>
      )}
    </Box>
  );
};

export default HighAvailabilityIcon;
