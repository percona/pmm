import { GRAFANA_DIRECT_PATH_PATTERN } from './constants';

export const isRenderingServer = (): boolean => {
  const params = new URLSearchParams(window.location.search);

  return params.get('render') === '1';
};

/**
 * Normalises the way nginx normalises $uri before testing its exclusions: percent-decode, then
 * collapse duplicate slashes. window.location.pathname does neither, so without this /graph/%6Cogin
 * is exempt server-side yet unrecognised here. '..' needs no handling - the browser resolves it
 * before it reaches location.pathname.
 */
export const isGrafanaDirectPath = (pathname: string) => {
  let normalized: string;

  try {
    normalized = decodeURIComponent(pathname);
  } catch {
    // nginx answers malformed escapes with 400, so they never reach the app.
    normalized = pathname;
  }

  return GRAFANA_DIRECT_PATH_PATTERN.test(normalized.replace(/\/{2,}/g, '/'));
};
