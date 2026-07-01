import { Box, Card, CardContent, Link, Typography } from '@mui/material';
import { FC, useEffect, useState } from 'react';
import type { InvestigationBlock } from 'api/investigations';
import { PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';
import { toSameOriginUrl, withRenderCacheParam } from 'components/adre/adre-chat-markdown.utils';

const RENDER_IMAGE_TIMEOUT_MS = 60000;

/** Fetches panel image with credentials and long timeout so the image loads in reports. */
const PanelImageWithFetch: FC<{
  src: string;
  alt: string;
  href: string | null;
}> = ({ src, alt, href }) => {
  const [state, setState] = useState<'loading' | { status: 'success'; url: string } | { status: 'error'; detail?: string }>('loading');

  useEffect(() => {
    let objectUrl: string | null = null;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), RENDER_IMAGE_TIMEOUT_MS);

    fetch(src, { credentials: 'include', signal: controller.signal })
      .then(async (res) => {
        const contentType = res.headers.get('Content-Type') ?? '';
        if (!res.ok) {
          let detail = `HTTP ${res.status}`;
          if (contentType.includes('application/json')) {
            try {
              const json = await res.json();
              if (json.error) detail += `: ${json.error}`;
            } catch { /* ignore */ }
          }
          throw new Error(detail);
        }
        if (!contentType.includes('image/')) {
          let detail = `Unexpected content type: ${contentType}`;
          if (contentType.includes('application/json')) {
            try {
              const json = await res.json();
              if (json.error) detail = json.error;
            } catch { /* ignore */ }
          }
          throw new Error(detail);
        }
        return res.blob();
      })
      .then((blob) => {
        objectUrl = URL.createObjectURL(blob);
        setState({ status: 'success', url: objectUrl });
      })
      .catch((err) => setState({ status: 'error', detail: err instanceof Error ? err.message : undefined }))
      .finally(() => clearTimeout(timeoutId));

    return () => {
      clearTimeout(timeoutId);
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [src]);

  if (state === 'loading') {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>
        Loading panel image…
      </Typography>
    );
  }
  if (state.status === 'error') {
    return (
      <Box sx={{ mb: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Image failed to load{(() => {
            if (!state.detail) return '';
            if (state.detail.includes('<!DOCTYPE') || state.detail.length > 200) return ' (Panel render timed out — try opening in Grafana directly)';
            return ` (${state.detail})`;
          })()}
        </Typography>
        {href && (
          <Link href={href} target="_blank" rel="noopener noreferrer" sx={{ display: 'inline-block', mt: 0.5 }}>
            Open panel in Grafana
          </Link>
        )}
      </Box>
    );
  }
  const img = (
    <Box
      component="img"
      src={state.url}
      alt={alt}
      loading="lazy"
      sx={{ maxWidth: '100%', height: 'auto', borderRadius: 1, display: 'block' }}
    />
  );
  return (
    <Box sx={{ mb: 1 }}>
      {href ? (
        <Link href={href} target="_blank" rel="noopener noreferrer" sx={{ display: 'block' }}>
          {img}
        </Link>
      ) : (
        img
      )}
    </Box>
  );
};

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
  const imageUrlRaw = typeof config.image_url === 'string' ? config.image_url : null;
  const imageUrl = imageUrlRaw ? toSameOriginUrl(withRenderCacheParam(imageUrlRaw)) : null;
  const dashboardUrl = typeof config.dashboard_url === 'string' ? config.dashboard_url : null;

  const toEpochMsOrOriginal = (s: string) => {
    if (!s) return s;
    const date = new Date(s);
    return Number.isNaN(date.getTime()) ? s : String(date.getTime());
  };
  const href =
    dashboardUrl ??
    (dashboardUid && panelId
      ? `${PMM_NEW_NAV_GRAFANA_PATH}/d/${dashboardUid}?viewPanel=${panelId}${timeFrom ? `&from=${encodeURIComponent(toEpochMsOrOriginal(timeFrom))}` : ''}${timeTo ? `&to=${encodeURIComponent(toEpochMsOrOriginal(timeTo))}` : ''}`
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

  const renderImageSrc = imageUrl ?? null;

  return (
    <Card variant="outlined" sx={{ mb: 2 }}>
      <CardContent>
        {block.title && (
          <Typography variant="subtitle1" fontWeight={600} sx={{ mb: 1 }}>
            {block.title}
          </Typography>
        )}
        {renderImageSrc && (
          <PanelImageWithFetch
            src={renderImageSrc}
            alt={block.title || 'Grafana panel'}
            href={href}
          />
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
