import { GRAFANA_SUB_PATH, PMM_BASE_PATH } from 'lib/constants';

/** Strip PMM UI shell prefix so paths are comparable to Grafana routes. */
export function stripPmmUiPrefix(pathname: string): string {
  if (pathname.startsWith(PMM_BASE_PATH)) {
    const rest = pathname.slice(PMM_BASE_PATH.length);
    return rest.startsWith('/') ? rest : `/${rest}`;
  }
  return pathname;
}

export type GrafanaContextKind = 'dashboard' | 'd-solo' | 'explore' | 'other-graph';

export interface ParsedGrafanaLocation {
  kind: GrafanaContextKind;
  /** Dashboard UID when kind is dashboard or d-solo path carries uid */
  dashboardUid: string | null;
  normalizedPath: string;
  searchParams: URLSearchParams;
}

/**
 * Parse PMM shell pathname + search into Grafana-oriented fields.
 * Returns null if the user is not under /graph (after PMM prefix strip).
 */
export function parseGrafanaLocation(pathname: string, search: string): ParsedGrafanaLocation | null {
  const normalizedPath = stripPmmUiPrefix(pathname);
  if (!normalizedPath.startsWith(GRAFANA_SUB_PATH)) {
    return null;
  }

  const searchParams = new URLSearchParams(search.startsWith('?') ? search.slice(1) : search);

  const exploreMatch = normalizedPath.match(/^\/graph\/explore(?:\/|$)/);
  if (exploreMatch) {
    return {
      kind: 'explore',
      dashboardUid: null,
      normalizedPath,
      searchParams,
    };
  }

  const dSoloMatch = normalizedPath.match(/^\/graph\/d-solo\/([^/]+)/);
  if (dSoloMatch) {
    return {
      kind: 'd-solo',
      dashboardUid: dSoloMatch[1] ?? null,
      normalizedPath,
      searchParams,
    };
  }

  const dMatch = normalizedPath.match(/^\/graph\/d\/([^/]+)/);
  if (dMatch) {
    return {
      kind: 'dashboard',
      dashboardUid: dMatch[1] ?? null,
      normalizedPath,
      searchParams,
    };
  }

  return {
    kind: 'other-graph',
    dashboardUid: null,
    normalizedPath,
    searchParams,
  };
}

function collectVarParams(params: URLSearchParams): string[] {
  const lines: string[] = [];
  const keys = [...params.keys()].sort();
  for (const key of keys) {
    if (key.startsWith('var-')) {
      const value = params.get(key) ?? '';
      lines.push(`- ${key}=${value}`);
    }
  }
  return lines;
}

const CONTEXT_RULES = `Rules for this context:
- Treat the URL fields below as the ONLY ground truth for which Grafana page, dashboard UID, focused panel (if any), time range, and template variables the user is viewing.
- If viewPanel is absent, the user is on the dashboard view without a single focused panel encoded in the URL — do NOT claim a specific panel ID or title unless you state you are inferring from the tab title only.
- If the user asks what they are looking at, answer from this context only; do NOT guess from prior tool calls, skills, or unrelated dashboards (e.g. do not invent mysql-innodb / panel IDs).
- Do not mention internal skill names or internal troubleshooting steps when answering "what panel/graph am I viewing?".`;

/**
 * Builds the system-message fragment injected before ADRE chat requests.
 * Empty string when not on a Grafana route.
 */
export function buildGrafanaDashboardContext(
  pathname: string,
  search: string,
  origin: string,
  grafanaDocumentTitle?: string | null,
): string {
  const parsed = parseGrafanaLocation(pathname, search);
  if (!parsed) {
    return '';
  }

  const fullUrl = `${origin}${pathname}${search}`;
  const { kind, dashboardUid, normalizedPath, searchParams } = parsed;
  const viewPanel = searchParams.get('viewPanel') ?? searchParams.get('viewpanel');
  const from = searchParams.get('from') ?? '';
  const to = searchParams.get('to') ?? '';
  const varLines = collectVarParams(searchParams);

  const parts: string[] = [
    'Current Grafana context (from the PMM UI URL synced with the Grafana iframe):',
    `- Full URL: ${fullUrl}`,
    `- Path kind: ${kind}`,
    `- Normalized Grafana path: ${normalizedPath}`,
  ];

  if (dashboardUid) {
    parts.push(`- Dashboard UID: ${dashboardUid}`);
  }

  if (grafanaDocumentTitle?.trim()) {
    parts.push(`- Grafana tab / document title: ${grafanaDocumentTitle.trim()}`);
  }

  if (kind === 'explore') {
    parts.push('- Page: Grafana Explore (not a saved dashboard).');
  }

  if (viewPanel) {
    parts.push(`- Focused panel (viewPanel): ${viewPanel}`);
  } else if (kind === 'dashboard') {
    parts.push(
      '- Focused panel: not set in the URL (full dashboard view). Do not assert a specific panel ID.',
    );
  }

  if (from || to) {
    parts.push(`- Time range: from=${from || '(default)'} to=${to || '(default)'}`);
  }

  if (varLines.length > 0) {
    parts.push('- Template variables:');
    parts.push(...varLines);
  }

  parts.push('');
  parts.push(CONTEXT_RULES);

  return parts.join('\n');
}
