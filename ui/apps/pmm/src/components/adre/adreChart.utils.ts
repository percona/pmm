import { GRAFANA_SUB_PATH } from 'lib/constants';

/** A chart the Assistant asked for: the query, which datasource runs it and over which window. */
export interface ChartSpec {
  ds: string;
  title?: string;
  from: string;
  to: string;
  unit?: string;
  legend?: string;
  query: string;
}

const ATTR_RE = /(\w+)=(?:"([^"]*)"|(\S+))/g;

/** The spec of a ```chart block, from its first-line attributes and its body; undefined without a query. */
export const chartSpecFromBlock = (
  meta: string | undefined,
  code: string
): ChartSpec | undefined => {
  const query = code.trim();
  if (!query) return undefined;
  const attrs: Record<string, string> = {};
  for (const m of (meta ?? '').matchAll(ATTR_RE)) {
    attrs[m[1]] = m[2] ?? m[3];
  }
  return {
    ds: attrs.ds || 'Metrics',
    ...(attrs.title && { title: attrs.title }),
    from: attrs.from || 'now-1h',
    to: attrs.to || 'now',
    ...(attrs.unit && { unit: attrs.unit }),
    ...(attrs.legend && { legend: attrs.legend }),
    query,
  };
};

const UNIT_MS: Record<string, number> = {
  s: 1000,
  m: 60_000,
  h: 3600_000,
  d: 86_400_000,
  w: 604_800_000,
};

const timeMs = (t: string, now: number) => {
  const rel = /^now(?:-(\d+)([smhdw]))?$/.exec(t.trim());
  if (rel) return rel[1] ? now - Number(rel[1]) * UNIT_MS[rel[2]] : now;
  const ms = Date.parse(t);
  return Number.isNaN(ms) ? undefined : ms;
};

/** The window as epoch milliseconds: RFC3339 or now-<n><s|m|h|d|w>; undefined when it does not parse or is empty. */
export const rangeMs = (from: string, to: string, now = Date.now()) => {
  const f = timeMs(from, now);
  const t = timeMs(to, now);
  if (f === undefined || t === undefined || f >= t) return undefined;
  return { from: f, to: t };
};

// Assistant_Chart.json has one panel per unit, since Grafana does not read a unit from a variable:
// Metrics panels 1–7 and ClickHouse panels 11–17, in this order.
const CHART_UNITS = [
  'short',
  'percent',
  'percentunit',
  'bytes',
  's',
  'ms',
  'ops',
];
const CHART_DASHBOARD = `${GRAFANA_SUB_PATH}/d-solo/pmm-assistant-chart/assistant-chart`;

const isClickHouse = (ds: string) => ds.startsWith('ClickHouse');

/** The query fields Explore takes for the chart's datasource. */
export const chartExploreQuery = (spec: ChartSpec) =>
  isClickHouse(spec.ds)
    ? { rawSql: spec.query, editorType: 'sql', format: 0 }
    : { expr: spec.query, range: true };

/** The embedded Assistant Chart panel that draws spec over range. */
export const chartEmbedUrl = (
  spec: ChartSpec,
  range: { from: number; to: number }
) => {
  const clickHouse = isClickHouse(spec.ds);
  const unit = Math.max(0, CHART_UNITS.indexOf(spec.unit ?? 'short'));
  const q = new URLSearchParams({
    orgId: '1',
    panelId: String((clickHouse ? 11 : 1) + unit),
    from: String(range.from),
    to: String(range.to),
    theme: 'dark',
    [clickHouse ? 'var-chds' : 'var-ds']: spec.ds,
    'var-title': spec.title ?? 'Chart',
    'var-query': spec.query,
  });
  if (spec.legend) q.set('var-legend', spec.legend);
  return `${CHART_DASHBOARD}?${q}`;
};

/** The embedded panel an Open in Grafana link points at (/graph/d/<uid>?viewPanel=N…), or undefined. */
export const panelEmbedUrl = (href: string) => {
  let u: URL;
  try {
    u = new URL(href, window.location.origin);
  } catch {
    return undefined;
  }
  const uid = /^\/graph\/d\/([^/]+)/.exec(u.pathname)?.[1];
  const panel = u.searchParams.get('viewPanel')?.replace(/^panel-/, '');
  if (!uid || !panel) return undefined;
  const q = new URLSearchParams(u.searchParams);
  q.delete('viewPanel');
  q.set('panelId', panel);
  q.set('orgId', q.get('orgId') ?? '1');
  q.set('theme', 'dark');
  return `${GRAFANA_SUB_PATH}/d-solo/${uid}?${q}`;
};

const IMAGE_THEN_LINK_RE =
  /!\[[^\]]*\]\(([^)\s]+)\)\s*\[[^\]]*\]\(([^)\s]+)\)/g;

/** Panel images and the Open in Grafana link that follows each, so the page can show the panel live instead. */
export const panelLinksByImage = (content: string) => {
  const pairs = new Map<string, string>();
  for (const [, image, link] of content.matchAll(IMAGE_THEN_LINK_RE)) {
    if (panelEmbedUrl(link)) pairs.set(image, link);
  }
  return pairs;
};
