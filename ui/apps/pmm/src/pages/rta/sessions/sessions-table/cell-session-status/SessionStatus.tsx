import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { getStyles } from 'components/ha-icon/HighAvailabilityIcon.styles';
import { Icon } from 'components/icon';
import { FC, ReactNode } from 'react';
import { RealtimeSessionStatus } from 'types/rta.types';
import { diffFromNow, formatDuration } from 'utils/datetime.utils';
import { Messages } from './SessionStatus.messages';
import { getSessionStatusText } from 'utils/status.utils';
import { SessionRow } from '../SessionsTable.types';
import { useLiveTimestamp } from 'hooks/useLiveTimestamp';

interface Props {
  session: SessionRow;
}

// The agent's messages can span several lines, one per finding.
const StatusMessageTooltip: FC<{ title: ReactNode; children: ReactNode }> = ({
  title,
  children,
}) => (
  <Tooltip title={<span style={{ whiteSpace: 'pre-line' }}>{title}</span>}>
    <Stack
      direction="row"
      alignItems="center"
      gap={0.5}
      data-testid="session-status-message"
    >
      {children}
    </Stack>
  </Tooltip>
);

const SessionStatus: FC<Props> = ({ session }) => {
  const theme = useTheme();
  const styles = getStyles(theme);
  useLiveTimestamp(30_000);

  if (session.status === RealtimeSessionStatus.running) {
    const runningFor = Messages.runningFor(
      formatDuration(diffFromNow(session.startTime))
    );

    if (!session.statusMessage) {
      return runningFor;
    }

    return (
      <StatusMessageTooltip
        title={`${Messages.runningWithWarnings}\n${session.statusMessage}`}
      >
        {runningFor}
        <Icon name="status-at-risk" sx={styles.icon} />
      </StatusMessageTooltip>
    );
  }

  const status = (
    <>
      <Icon name="status-at-risk" sx={styles.icon} />
      <Typography variant="body2" color="text.secondary">
        {getSessionStatusText(session.status)}
      </Typography>
    </>
  );

  if (session.statusMessage) {
    return (
      <StatusMessageTooltip title={session.statusMessage}>
        {status}
      </StatusMessageTooltip>
    );
  }

  return (
    <Stack direction="row" alignItems="center" gap={0.5}>
      {status}
    </Stack>
  );
};

export default SessionStatus;
