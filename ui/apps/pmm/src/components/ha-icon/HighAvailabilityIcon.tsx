import { useTheme } from '@mui/material';
import Box from '@mui/material/Box';
import { Icon } from 'components/icon';
import { IconName } from 'components/icon/Icon.types';
import { FC } from 'react';
import { HighAvailabilityHealth } from 'types/high-availability.types';
import { getStyles } from './HighAvailabilityIcon.styles';

interface Props {
  health: HighAvailabilityHealth;
}

const HighAvailabilityIcon: FC<Props> = ({ health }) => {
  const theme = useTheme();
  const styles = getStyles(theme);

  return (
    <Box sx={{ position: 'relative' }}>
      <Icon name="cluster" />
      {health !== 'healthy' && (
        <Box sx={{ position: 'absolute', top: -4, right: -7 }}>
          <Icon sx={styles.icon} name={ICON_MAP[health]} />
        </Box>
      )}
    </Box>
  );
};

const ICON_MAP: Record<Exclude<HighAvailabilityHealth, 'healthy'>, IconName> = {
  'at-risk': 'status-at-risk',
  down: 'status-down',
  updating: 'status-updating',
};

export default HighAvailabilityIcon;
