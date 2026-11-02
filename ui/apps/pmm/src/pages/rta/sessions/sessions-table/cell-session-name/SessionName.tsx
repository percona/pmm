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
  // The overview would silently drop part of a mixed set; its services are still linked
  // individually.
  const isMixedCluster =
    session.type === 'cluster' && session.serviceType === undefined;

  if (isMixedCluster) {
    return (
      <Tooltip title={Messages.mixedClusterTooltip} arrow>
        <Typography
          variant="body2"
          color="text.secondary"
          component="span"
          // Focusable so keyboard users can open the tooltip explaining why this is not a link.
          tabIndex={0}
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
