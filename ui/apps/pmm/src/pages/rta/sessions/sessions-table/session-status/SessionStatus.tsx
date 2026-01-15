import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import { getStyles } from 'components/ha-icon/HighAvailabilityIcon.styles';
import { Icon } from 'components/icon';
import { FC } from 'react';
import { AgentStatus } from 'types/agent.types';
import { RealTimeSession } from 'types/rta.types';
import { getStatusText } from 'utils/agents.utils';
import { diffFromNow, formatDuration } from 'utils/datetime.utils';
import { Messages } from './SessionStatus.messages';

interface Props {
  session: RealTimeSession;
}

const SessionStatus: FC<Props> = ({ session }) => {
  const theme = useTheme();
  const styles = getStyles(theme);

  if (session.status === AgentStatus.RUNNING) {
    return Messages.runningFor(formatDuration(diffFromNow(session.startedAt)));
  }

  return (
    <Stack direction="row" alignItems="center" gap={0.5}>
      <Icon name="status-at-risk" sx={styles.icon} />
      <Typography variant="body2" color="text.secondary">
        {getStatusText(session.status)}
      </Typography>
    </Stack>
  );
};

export default SessionStatus;
