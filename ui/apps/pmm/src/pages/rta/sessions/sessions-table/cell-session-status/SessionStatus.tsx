import Box from '@mui/material/Box';
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

// The agent sends one finding per line. A MariaDB instance can report several long ones, so
// they are listed apart, and the tooltip scrolls rather than running off a short viewport.
// The cap is just under half the viewport so the tooltip always fits on the side of its anchor
// with more room, which is the side the popper flips to: 60vh did not, and at 1280x800 a row
// in the middle of the list pushed it 20px past the bottom edge. The 64px leave room for the
// row the anchor sits in, the tooltip's padding and its arrow.
const StatusMessageTitle: FC<{ intro?: string; message: string }> = ({
  intro,
  message,
}) => {
  const findings = message.split('\n').filter((line) => line.trim() !== '');

  return (
    <Box
      data-testid="session-status-message-tooltip"
      sx={{ maxHeight: 'calc(50vh - 64px)', overflowY: 'auto', pr: 0.5 }}
    >
      {intro && (
        <Typography variant="inherit" component="p" sx={{ mb: 1 }}>
          {intro}
        </Typography>
      )}
      {findings.length === 1 && !intro ? (
        findings[0]
      ) : (
        <Box component="ul" sx={{ m: 0, pl: 2, '& > li + li': { mt: 1 } }}>
          {findings.map((finding, index) => (
            <li key={index}>{finding}</li>
          ))}
        </Box>
      )}
    </Box>
  );
};

const StatusMessageTooltip: FC<{
  intro?: string;
  message: string;
  children: ReactNode;
}> = ({ intro, message, children }) => (
  <Tooltip
    title={<StatusMessageTitle intro={intro} message={message} />}
    slotProps={{ tooltip: { sx: { maxWidth: 480 } } }}
  >
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
        intro={Messages.runningWithWarnings}
        message={session.statusMessage}
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
      <StatusMessageTooltip message={session.statusMessage}>
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
