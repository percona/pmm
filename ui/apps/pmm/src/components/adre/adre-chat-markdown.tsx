import { Box, Link, Typography } from '@mui/material';
import { FC, useState, useEffect, ReactNode, memo } from 'react';
import { CodeBlock } from 'pages/updates/change-log/code-block';
import { PMM_BASE_PATH, PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

const GRAFANA_RENDER_PATH = '/v1/grafana/render';
const GRAFANA_RENDER_D_SOLO = '/graph/render/d-solo/';
const RENDER_IMAGE_TIMEOUT_MS = 60000;

function toEpochMsOrOriginal(s: string): string {
  if (!s) return s;
  const date = new Date(s);
  if (Number.isNaN(date.getTime())) return s;

  return String(date.getTime());
}

export function toSameOriginUrl(url: string): string {
  if (!url || url.startsWith('/')) return url;
  try {
    const u = new URL(url, window.location.origin);
    if (u.origin === window.location.origin) return url;
    const path = u.pathname + u.search;
    if (path.startsWith('/v1/grafana/render') || path.startsWith('/graph/')) {
      return window.location.origin + path;
    }

    return url;
  } catch {
    return url;
  }
}

export function toGrafanaDashboardLink(href: string): string {
  if (!href || href === '#') return href;
  const sameOrigin = toSameOriginUrl(href);
  try {
    const u = new URL(sameOrigin, window.location.origin);
    if (!u.pathname.startsWith('/graph/d/')) return sameOrigin;

    return PMM_BASE_PATH + u.pathname + u.search;
  } catch {
    return sameOrigin;
  }
}

export function dashboardUrlFromRenderUrl(renderSrc: string): string | null {
  try {
    let pathOnly: string;
    let params: URLSearchParams;
    if (renderSrc.includes('://')) {
      const u = new URL(renderSrc);
      pathOnly = u.pathname;
      params = u.searchParams;
    } else {
      const path = renderSrc.startsWith('/') ? renderSrc : `/${renderSrc}`;
      const searchStart = path.indexOf('?');
      pathOnly = searchStart === -1 ? path : path.slice(0, searchStart);
      params = new URLSearchParams(searchStart === -1 ? '' : path.slice(searchStart + 1));
    }

    let uid: string | null = null;
    let panelId: string | null = null;

    if (pathOnly.includes(GRAFANA_RENDER_D_SOLO)) {
      const match = pathOnly.match(/\/graph\/render\/d-solo\/([^/]+)/);
      uid = match ? match[1] : null;
      panelId = params.get('panelId');
    } else {
      uid = params.get('dashboard_uid');
      panelId = params.get('panel_id');
    }

    const from = params.get('from');
    const to = params.get('to');
    if (!uid) return null;
    const base = `${PMM_BASE_PATH}${PMM_NEW_NAV_GRAFANA_PATH}/d/${uid}`;
    const q = new URLSearchParams();
    if (panelId) q.set('viewPanel', panelId);
    if (from) q.set('from', toEpochMsOrOriginal(from));
    if (to) q.set('to', toEpochMsOrOriginal(to));
    params.forEach((v, k) => {
      if (k.startsWith('var-')) q.set(k, v);
    });
    const qs = q.toString();

    return qs ? `${base}?${qs}` : base;
  } catch {
    return null;
  }
}

export function isGrafanaRenderImageSrc(src: string): boolean {
  if (src.includes(GRAFANA_RENDER_PATH) && src.includes('dashboard_uid=') && src.includes('panel_id=')) return true;

  return src.includes(GRAFANA_RENDER_D_SOLO) && src.includes('panelId=');
}

function normalizePanelId(panelId: string | null): string {
  if (!panelId) return '';
  const s = panelId.trim();

  return s.startsWith('panel-') ? s.slice(6) : s;
}

export function getRenderImageUrlsInContent(content: string): string[] {
  if (!content) return [];
  const urls: string[] = [];
  const re = /!\[[^\]]*\]\((.*?)\)/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(content)) !== null) {
    const url = m[1]?.trim();
    if (url && isGrafanaRenderImageSrc(url)) urls.push(url);
  }

  return urls;
}

export function parseRenderImageUrlToPanelKey(url: string): string | null {
  try {
    let pathOnly: string;
    let params: URLSearchParams;
    if (url.includes('://')) {
      const u = new URL(url);
      pathOnly = u.pathname;
      params = u.searchParams;
    } else {
      const path = url.startsWith('/') ? url : `/${url}`;
      const searchStart = path.indexOf('?');
      pathOnly = searchStart === -1 ? path : path.slice(0, searchStart);
      params = new URLSearchParams(searchStart === -1 ? '' : path.slice(searchStart + 1));
    }
    let uid: string | null = null;
    let panelId: string | null = null;
    if (pathOnly.includes(GRAFANA_RENDER_D_SOLO)) {
      const match = pathOnly.match(/\/graph\/render\/d-solo\/([^/]+)/);
      uid = match ? match[1] : null;
      panelId = params.get('panelId');
    } else {
      uid = params.get('dashboard_uid');
      panelId = params.get('panel_id');
    }
    if (!uid) return null;

    return `${uid}|${normalizePanelId(panelId)}`;
  } catch {
    return null;
  }
}

export function parseDashboardLinkToPanelKey(href: string): string | null {
  if (!href || href === '#') return null;
  try {
    const sameOrigin = toSameOriginUrl(href);
    const u = new URL(sameOrigin, window.location.origin);
    if (!u.pathname.startsWith('/graph/d/')) return null;
    const match = u.pathname.match(/\/graph\/d\/([^/]+)/);
    const uid = match ? match[1] : null;
    const viewPanel = u.searchParams.get('viewPanel');
    if (!uid) return null;

    return `${uid}|${normalizePanelId(viewPanel)}`;
  } catch {
    return null;
  }
}

export function withRenderCacheParam(src: string): string {
  if (!src || !src.includes(GRAFANA_RENDER_PATH)) return src;
  if (/[?&]cache=1(?=&|$)/.test(src)) return src;
  try {
    const u = new URL(src, window.location.origin);
    u.searchParams.set('cache', '1');

    return u.toString();
  } catch {
    return src.includes('?') ? `${src}&cache=1` : `${src}?cache=1`;
  }
}

const PANEL_IMAGE_CACHE_MAX = 50;
const panelImageCache = new Map<string, string>();

function panelImageCacheSet(key: string, value: string) {
  if (panelImageCache.size >= PANEL_IMAGE_CACHE_MAX) {
    const oldest = panelImageCache.keys().next().value;
    if (oldest !== undefined) {
      const url = panelImageCache.get(oldest);
      if (url) URL.revokeObjectURL(url);
      panelImageCache.delete(oldest);
    }
  }
  panelImageCache.set(key, value);
}

export function clearPanelImageCache() {
  panelImageCache.forEach((url) => URL.revokeObjectURL(url));
  panelImageCache.clear();
}

const GrafanaPanelImageInner: FC<{
  src: string;
  alt: string;
  dashboardHref: string | null;
}> = ({ src, alt, dashboardHref }) => {
  const [state, setState] = useState<'loading' | { status: 'success'; url: string } | { status: 'error'; detail?: string }>('loading');

  useEffect(() => {
    const cached = panelImageCache.get(src);
    if (cached) {
      setState({ status: 'success', url: cached });
      return;
    }
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
        panelImageCacheSet(src, objectUrl);
        setState({ status: 'success', url: objectUrl });
      })
      .catch((err) => setState({ status: 'error', detail: err instanceof Error ? err.message : undefined }))
      .finally(() => clearTimeout(timeoutId));

    return () => {
      clearTimeout(timeoutId);
    };
  }, [src]);

  if (state === 'loading') {
    return (
      <Box sx={{ my: 1, minHeight: 500, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: 'rgba(255,255,255,0.03)', borderRadius: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Loading panel image…
        </Typography>
      </Box>
    );
  }
  if (state.status === 'error') {
    const friendlyDetail = state.detail && (state.detail.includes('<!DOCTYPE') || state.detail.length > 200)
      ? 'Panel render timed out — try opening in Grafana directly'
      : state.detail;
    return (
      <Box sx={{ my: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Image failed to load{friendlyDetail ? ` (${friendlyDetail})` : ''}
        </Typography>
        {dashboardHref && (
          <Link
            href={dashboardHref}
            target="_blank"
            rel="noopener noreferrer"
            sx={{
              display: 'inline-block',
              mt: 0.5,
              fontSize: '0.8125rem',
              color: 'primary.light',
              '&:hover': { color: 'primary.main' },
            }}
          >
            Open in Grafana
          </Link>
        )}
      </Box>
    );
  }

  return (
    <Box sx={{ my: 1 }}>
      <Box
        component="img"
        src={state.url}
        alt={alt}
        loading="lazy"
        sx={{ maxWidth: '100%', height: 'auto', borderRadius: 1, display: 'block' }}
      />
      {dashboardHref && (
        <Link
          href={dashboardHref}
          target="_blank"
          rel="noopener noreferrer"
          sx={{
            display: 'inline-block',
            mt: 0.5,
            fontSize: '0.8125rem',
            color: 'primary.light',
            '&:hover': { color: 'primary.main' },
          }}
        >
          Open in Grafana
        </Link>
      )}
    </Box>
  );
};

export const GrafanaPanelImage = memo(GrafanaPanelImageInner);

/** Returns markdown component overrides for rendering Grafana panel images, code blocks, and dashboard links within chat messages. */
export function getMarkdownComponents(content: string) {
  const panelKeysFromImages = new Set(
    getRenderImageUrlsInContent(content).map(parseRenderImageUrlToPanelKey).filter(Boolean)
  );

  return {
    code: ({
      inline,
      children,
    }: {
      inline?: boolean;
      children?: ReactNode;
    }) => {
      if (inline) {
        return (
          <Box
            component="code"
            sx={{
              px: 0.5,
              borderRadius: 0.5,
              fontFamily: 'Roboto Mono, monospace',
              bgcolor: 'action.hover',
            }}
          >
            {children}
          </Box>
        );
      }

      return <CodeBlock>{children}</CodeBlock>;
    },
    table: ({ children }: { children?: ReactNode }) => (
      <Box sx={{ width: '100%', overflowX: 'auto', my: 1 }}>
        <Box component="table" sx={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
          {children}
        </Box>
      </Box>
    ),
    th: ({ children }: { children?: ReactNode }) => (
      <Box
        component="th"
        sx={{
          textAlign: 'left',
          px: 1,
          py: 0.5,
          border: 1,
          borderColor: 'divider',
          bgcolor: 'action.hover',
          fontWeight: 600,
        }}
      >
        {children}
      </Box>
    ),
    td: ({ children }: { children?: ReactNode }) => (
      <Box
        component="td"
        sx={{
          px: 1,
          py: 0.5,
          border: 1,
          borderColor: 'divider',
          verticalAlign: 'top',
        }}
      >
        {children}
      </Box>
    ),
    a: ({ href, children }: { href?: string; children?: ReactNode }) => {
      const panelKey = href ? parseDashboardLinkToPanelKey(href) : null;
      if (panelKey !== null && panelKeysFromImages.has(panelKey)) return null;

      return (
        <Link
          href={href ? toGrafanaDashboardLink(href) : '#'}
          target="_blank"
          rel="noopener noreferrer"
          sx={{
            fontSize: '0.8125rem',
            color: 'primary.light',
            '&:hover': { color: 'primary.main' },
          }}
        >
          {children}
        </Link>
      );
    },
    img: ({ src, alt }: { src?: string; alt?: string }) => {
      if (src && isGrafanaRenderImageSrc(src)) {
        const imageSrc = toSameOriginUrl(withRenderCacheParam(src));
        const dashboardHref = dashboardUrlFromRenderUrl(src);

        return (
          <GrafanaPanelImage
            src={imageSrc}
            alt={alt ?? 'Grafana panel'}
            dashboardHref={dashboardHref}
          />
        );
      }

      return <Box component="img" src={src ? toSameOriginUrl(src) : undefined} alt={alt ?? ''} />;
    },
  };
}
