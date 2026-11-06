import type { FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import type { AlertSeverityBand } from 'types/alerting.types';
import { ALERTING_URL, NO_VALUE } from '../AmazonRds.constants';
import { Messages } from '../AmazonRds.messages';
import type { AlertSummary } from '../AmazonRds.types';

const BAND_COLORS: Record<AlertSeverityBand, string> = {
  critical: 'error.main',
  error: 'error.main',
  warning: 'warning.main',
  info: 'info.main',
};

// Without alert data the cell shows no value, never a "None" it cannot vouch for. The link opens
// in a new tab, like the dashboard links.
export const AlertsBadge: FC<{ alerts?: AlertSummary }> = ({ alerts }) => {
  if (!alerts) {
    return <>{NO_VALUE}</>;
  }

  if (!alerts.count) {
    return (
      <Typography component="span" variant="inherit" color="text.secondary">
        {Messages.noAlerts}
      </Typography>
    );
  }

  const label = Messages.firingAlerts(alerts.count);

  return (
    <Link
      component={RouterLink}
      to={ALERTING_URL}
      target="_blank"
      rel="noopener noreferrer"
      title={label}
      aria-label={label}
      underline="hover"
      color={alerts.band ? BAND_COLORS[alerts.band] : 'text.primary'}
      sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
    >
      <NotificationsOutlined fontSize="small" />
      {alerts.count}
    </Link>
  );
};
