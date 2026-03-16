import { Box, Card, CardContent, Link, Typography } from '@mui/material';
import { FC } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

const RENDER_API_PATH = '/v1/grafana/render';

export const PanelBlock: FC<{ block: InvestigationBlock }> = ({ block }) => {
  const config = (block.configJson || {}) as Record<string, unknown> & {
    dashboardUid?: string;
    panelId?: string;
    dashboard_uid?: string;
    panel_id?: string;
    timeFrom?: string;
    timeTo?: string;
    time_from?: string;
    time_to?: string;
    image_url?: string;
    dashboard_url?: string;
  };
  const dashboardUid = config.dashboardUid ?? config.dashboard_uid;
  const panelId = config.panelId ?? config.panel_id;
  const timeFrom = config.timeFrom ?? config.time_from;
  const timeTo = config.timeTo ?? config.time_to;
  const imageUrl = typeof config.image_url === 'string' ? config.image_url : null;
  const dashboardUrl = typeof config.dashboard_url === 'string' ? config.dashboard_url : null;

  const href =
    dashboardUrl ??
    (dashboardUid && panelId
      ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}?viewPanel=${panelId}${timeFrom ? `&from=${encodeURIComponent(timeFrom)}` : ''}${timeTo ? `&to=${encodeURIComponent(timeTo)}` : ''}`
      : dashboardUid
        ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}`
        : null);

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

  const renderImageSrc =
    imageUrl ||
    (dashboardUid && panelId && timeFrom && timeTo
      ? (() => {
          const params = new URLSearchParams({
            dashboard_uid: dashboardUid,
            panel_id: String(panelId),
            from: timeFrom,
            to: timeTo,
            width: '1000',
            height: '500',
          });
          Object.entries(config).forEach(([k, v]) => {
            if ((k.startsWith('var_') || k.startsWith('var-')) && v != null && typeof v === 'string') {
              params.set(k.startsWith('var_') ? `var-${k.slice(4)}` : k, v);
            }
          });
          return `${RENDER_API_PATH}?${params.toString()}`;
        })()
      : null);

  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      <CardContent>
        {block.title && (
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
            {block.title}
          </Typography>
        )}
        {renderImageSrc && (
          <Box sx={{ mb: 1 }}>
            {href ? (
              <Link
                href={href}
                target="_blank"
                rel="noopener noreferrer"
                sx={{ display: 'block' }}
              >
                <Box
                  component="img"
                  src={renderImageSrc}
                  alt={block.title || 'Grafana panel'}
                  loading="lazy"
                  sx={{ maxWidth: '100%', height: 'auto', borderRadius: 1, display: 'block' }}
                />
              </Link>
            ) : (
              <Box
                component="img"
                src={renderImageSrc}
                alt={block.title || 'Grafana panel'}
                loading="lazy"
                sx={{ maxWidth: '100%', height: 'auto', borderRadius: 1, display: 'block' }}
              />
            )}
          </Box>
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
        ) : !embedSrc && !renderImageSrc ? (
          <Typography variant="body2" color="text.secondary">
            Panel (dashboard_uid or panel_id not set)
          </Typography>
        ) : null}
      </CardContent>
    </Card>
  );
};
