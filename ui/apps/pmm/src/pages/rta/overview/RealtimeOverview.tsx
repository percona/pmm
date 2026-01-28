import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { FC } from 'react';
import { Link as RouterLink, useSearchParams } from 'react-router-dom';
import { RealtimePage } from '../components/rta-page';

/**
 * Currently just a temporary page to test linking to and from sessions page.
 */
const RealtimeOverviewPage: FC = () => {
  const [searchParams] = useSearchParams();
  const serviceIds = searchParams.getAll('serviceIds');

  return (
    <RealtimePage>
      <Link component={RouterLink} to="/rta/sessions?fromOverview=true">
        Back to sessions
      </Link>
      <Typography>Service IDs: [{serviceIds.join(', ') || 'N/A'}]</Typography>
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
