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
    time_from?: string;
    time_to?: string;
  };
  const dashboardUid = config.dashboardUid ?? config.dashboard_uid;
  const panelId = config.panelId ?? config.panel_id;
  const timeFrom = config.timeFrom ?? config.time_from;
  const timeTo = config.timeTo ?? config.time_to;

  const href = dashboardUid && panelId
    ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}?viewPanel=${panelId}`
    : dashboardUid
      ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}`
      : null;

  const embedSrc =
    dashboardUid && panelId
      ? (() => {
          const base = `${PMM_NEW_NAV_GRAFANA_PATH}/d-solo/${dashboardUid}/?panelId=${panelId}`;
          const params = new URLSearchParams();
          if (timeFrom) params.set('from', timeFrom);
          if (timeTo) params.set('to', timeTo);
          const q = params.toString();
          return q ? `${base}&${q}` : base;
        })()
      : null;

  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      <CardContent>
        {block.title && (
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
            {block.title}
          </Typography>
        )}
        {embedSrc && (
          <iframe
            src={embedSrc}
            title={block.title || 'Grafana panel'}
            style={{
              width: '100%',
              height: 250,
              border: 0,
            }}
            loading="lazy"
          />
        )}
        {href ? (
          <Link href={href} target="_blank" rel="noopener noreferrer" sx={{ display: 'block', mt: 1 }}>
            Open panel in Grafana
          </Link>
        ) : !embedSrc ? (
          <Typography variant="body2" color="text.secondary">
            Panel (dashboard_uid or panel_id not set)
          </Typography>
        ) : null}
      </CardContent>
    </Card>
  );
};
