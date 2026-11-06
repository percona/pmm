import { useState, type FC } from 'react';
import { Link as RouterLink } from 'react-router-dom';
import Box from '@mui/material/Box';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Popover from '@mui/material/Popover';
import Typography from '@mui/material/Typography';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import type { AlertRow } from 'pages/alerting/status/AlertsPage.types';
import {
  getSeverityBand,
  SEVERITY_BAND_CONFIG,
} from 'pages/alerting/status/alert-severity';
import type { AlertSeverityBand } from 'types/alerting.types';
import { NO_VALUE } from '../AmazonRds.constants';
import { Messages } from '../AmazonRds.messages';
import type { RdsInstance } from '../AmazonRds.types';
import {
  getAlertGroupsUrl,
  getAlertRuleUrl,
  getAlertsBand,
} from '../AmazonRds.utils';

const BAND_COLORS: Record<AlertSeverityBand, string> = {
  critical: 'error.main',
  error: 'error.main',
  warning: 'warning.main',
  info: 'info.main',
};

// Links open in a new tab, like the dashboard links.
const NewTabLink: FC<{ to: string; children: string }> = ({ to, children }) => (
  <Link
    component={RouterLink}
    to={to}
    target="_blank"
    rel="noopener noreferrer"
  >
    {children}
  </Link>
);

const AlertItem: FC<{ alert: AlertRow; showInstance: boolean }> = ({
  alert,
  showInstance,
}) => {
  const band = getSeverityBand(alert.labels.severity);
  const url = getAlertRuleUrl(alert);
  const details = [
    showInstance ? alert.serviceName || alert.nodeId : undefined,
    alert.activeAt ? Messages.firingFor(alert.age) : undefined,
  ].filter(Boolean);

  return (
    <ListItem>
      <ListItemIcon sx={{ minWidth: 32, '& svg': { fontSize: 20 } }}>
        {band ? SEVERITY_BAND_CONFIG[band].icon : <NotificationsOutlined />}
      </ListItemIcon>
      <ListItemText
        primary={
          url ? (
            <NewTabLink to={url}>{alert.alertName}</NewTabLink>
          ) : (
            alert.alertName
          )
        }
        secondary={details.join(' · ')}
      />
    </ListItem>
  );
};

interface AlertsBadgeProps {
  // Undefined while the alerts are unknown.
  alerts?: AlertRow[];
  instances: RdsInstance[];
}

// Without alert data the cell shows no value, never a "None" it cannot vouch for.
export const AlertsBadge: FC<AlertsBadgeProps> = ({ alerts, instances }) => {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null);

  if (!alerts) {
    return <>{NO_VALUE}</>;
  }

  if (!alerts.length) {
    return (
      <Typography component="span" variant="inherit" color="text.secondary">
        {Messages.noAlerts}
      </Typography>
    );
  }

  const band = getAlertsBand(alerts);
  const label = Messages.firingAlerts(alerts.length);
  const groupsUrl = getAlertGroupsUrl(instances);

  return (
    <>
      <Link
        component="button"
        type="button"
        aria-label={label}
        aria-haspopup="dialog"
        aria-expanded={!!anchor}
        title={label}
        underline="hover"
        color={band ? BAND_COLORS[band] : 'text.primary'}
        onClick={(event) => setAnchor(event.currentTarget)}
        sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.5 }}
      >
        <NotificationsOutlined fontSize="small" />
        {alerts.length}
      </Link>
      <Popover
        open={!!anchor}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }}
        slotProps={{
          paper: {
            role: 'dialog',
            'aria-label': label,
            sx: { maxWidth: 440, maxHeight: 400 },
          },
        }}
      >
        <List dense>
          {alerts.map((alert) => (
            <AlertItem
              key={alert.id}
              alert={alert}
              showInstance={instances.length > 1}
            />
          ))}
        </List>
        {groupsUrl && (
          <Box sx={{ px: 2, pb: 1.5 }}>
            <NewTabLink to={groupsUrl}>{Messages.showAllAlerts}</NewTabLink>
          </Box>
        )}
      </Popover>
    </>
  );
};
