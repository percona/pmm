import { toSameOriginUrl } from 'components/adre/adre-chat-markdown.utils';

function pmmRenderPath(url: string): string {
  if (!url) return '';
  if (url.startsWith('/')) return url.split('?')[0] ?? url;
  try {
    return new URL(url).pathname;
  } catch {
    return '';
  }
}

/** True when URL points at PMM Grafana render API (blob cache or live render). */
export function isAllowedPMMImageUrl(url: string): boolean {
  const normalized = toSameOriginUrl(url.trim());
  const path = pmmRenderPath(normalized);
  return path.startsWith('/v1/grafana/render') || path.startsWith('/graph/render');
}
