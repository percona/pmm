import Link from '@mui/material/Link';
import { FC } from 'react';
import { SessionRow } from '../SessionsTable.types';
import { Link as RouterLink } from 'react-router-dom';
import { getServiceIds } from '../SessionsTable.utils';

interface Props {
  session: SessionRow;
}

export const SessionName: FC<Props> = ({ session }) => {
  const serviceIds = getServiceIds(session);
  const params = new URLSearchParams();
  serviceIds.forEach((serviceId) => params.append('serviceIds', serviceId));

  return (
    <Link component={RouterLink} to={`/rta/overview?${params.toString()}`}>
      {session.sessionName}
    </Link>
  );
};

export default SessionName;
