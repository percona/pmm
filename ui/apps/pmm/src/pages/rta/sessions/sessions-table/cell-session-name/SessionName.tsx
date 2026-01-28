import Link from '@mui/material/Link';
import { FC } from 'react';
import { SessionRow } from '../SessionsTable.types';
import { Link as RouterLink } from 'react-router-dom';
import { getServiceIds } from '../SessionsTable.utils';
import { createRealtimeOverviewUrl } from 'utils/link.utils';

interface Props {
  session: SessionRow;
}

export const SessionName: FC<Props> = ({ session }) => {
  const serviceIds = getServiceIds(session);

  return (
    <Link component={RouterLink} to={createRealtimeOverviewUrl(serviceIds)}>
      {session.sessionName}
    </Link>
  );
};

export default SessionName;
