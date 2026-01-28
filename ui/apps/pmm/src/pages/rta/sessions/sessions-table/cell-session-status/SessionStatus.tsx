import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import { getStyles } from 'components/ha-icon/HighAvailabilityIcon.styles';
import { Icon } from 'components/icon';
import { FC } from 'react';
import { RealtimeSessionStatus } from 'types/rta.types';
import { diffFromNow, formatDuration } from 'utils/datetime.utils';
import { Messages } from './SessionStatus.messages';
import { getSessionStatusText } from 'utils/status.utils';
import { SessionRow } from '../SessionsTable.types';

interface Props {
  session: SessionRow;
}

const SessionStatus: FC<Props> = ({ session }) => {
  const theme = useTheme();
  const styles = getStyles(theme);

  if (session.status === RealtimeSessionStatus.running) {
    return Messages.runningFor(formatDuration(diffFromNow(session.startTime)));
  }

  return (
    <Stack direction="row" alignItems="center" gap={0.5}>
      <Icon name="status-at-risk" sx={styles.icon} />
      <Typography variant="body2" color="text.secondary">
        {getSessionStatusText(session.status)}
      </Typography>
    </Stack>
  );
};

export default SessionStatus;
