import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { Tooltip } from '@percona/peak-ui';
import { FC } from 'react';
import { SessionRow } from '../SessionsTable.types';
import { Link as RouterLink } from 'react-router-dom';
import { getServiceIds } from '../SessionsTable.utils';
import { createRealtimeOverviewUrl } from 'utils/link.utils';
import { Messages } from './SessionName.messages';

interface Props {
  session: SessionRow;
}

export const SessionName: FC<Props> = ({ session }) => {
  const serviceIds = getServiceIds(session);
  // A cluster row carries the technology its services share, so an undefined one spans more
  // than one. Linking it would hand the overview a mixed set, which shows a single technology
  // and silently drops the rest -- the outcome the selection screen deliberately avoids by
  // sending mixed selections here in the first place. Its services are still linked
  // individually, so the reader picks the one they want to watch.
  const isMixedCluster =
    session.type === 'cluster' && session.serviceType === undefined;

  if (isMixedCluster) {
    return (
      <Tooltip title={Messages.mixedClusterTooltip} arrow>
        <Typography
          variant="body2"
          color="text.secondary"
          component="span"
          data-testid={`session-${session.sessionId}-mixed-cluster`}
        >
          {session.sessionName}
        </Typography>
      </Tooltip>
    );
  }

  return (
    <Link component={RouterLink} to={createRealtimeOverviewUrl(serviceIds)}>
      {session.sessionName}
    </Link>
  );
};

export default SessionName;
