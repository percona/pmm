import { createContext, useContext } from 'react';
import { PMM_BASE_PATH, PMM_NEW_NAV_GRAFANA_PATH } from 'lib/constants';

export const GRAFANA_RENDER_PATH = '/v1/grafana/render';
export const GRAFANA_RENDER_D_SOLO = '/graph/render/d-solo/';
export const RENDER_IMAGE_TIMEOUT_MS = 60000;
export const PANEL_IMAGE_MAX_CONCURRENT = 3;
export const PANEL_IMAGE_ROOT_MARGIN = '300px';
export const PLACEHOLDER_MIN_HEIGHT = 500;

/** Scroll root for IntersectionObserver (chat scroll container). When null, viewport is used. */
export const PanelScrollRootContext = createContext<HTMLElement | null>(null);

export function usePanelScrollRoot(): HTMLElement | null {
  return useContext(PanelScrollRootContext);
}

/** Acquire a slot for panel fetch; returns release function. Resolves when a slot is free. */
function createPanelFetchQueue(maxConcurrent: number) {
  let inFlight = 0;
  const waiters: Array<() => void> = [];

  function release() {
    inFlight = Math.max(0, inFlight - 1);
    if (waiters.length > 0 && inFlight < maxConcurrent) {
      const next = waiters.shift();
      if (next) next();
    }
  }

  function acquire(): Promise<() => void> {
    if (inFlight < maxConcurrent) {
      inFlight += 1;
      return Promise.resolve(release);
    }
    return new Promise<() => void>((resolve) => {
      waiters.push(() => {
        inFlight += 1;
        resolve(release);
      });
    });
  }

  return { acquire, release };
}

export const panelFetchQueue = createPanelFetchQueue(PANEL_IMAGE_MAX_CONCURRENT);

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
export const panelImageCache = new Map<string, string>();

export function panelImageCacheSet(key: string, value: string) {
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
