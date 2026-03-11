import { Card, CardContent, Link, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

export const PanelBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const config = (block.configJson || {}) as {
    dashboardUid?: string;
    panelId?: string;
    dashboard_uid?: string;
    panel_id?: string;
    timeFrom?: string;
    timeTo?: string;
  };
  const dashboardUid = config.dashboardUid ?? config.dashboard_uid;
  const panelId = config.panelId ?? config.panel_id;
  const href = dashboardUid && panelId
    ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}?viewPanel=${panelId}`
    : dashboardUid
      ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}`
      : null;
  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      <CardContent>
        {block.title && (
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
            {block.title}
          </Typography>
        )}
        {href ? (
          <Link href={href} target="_blank" rel="noopener noreferrer">
            Open panel in Grafana
          </Link>
        ) : (
          <Typography variant="body2" color="text.secondary">
            Panel (dashboard_uid or panel_id not set)
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
