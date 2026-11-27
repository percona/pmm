import type { AdreMessageRow } from 'api/adre';
import type { ProgressStep } from 'hooks/useAdreChat';
import {
  PMM_LOGS_PATH,
  PMM_NEW_NAV_GRAFANA_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { searchParamsFromFilter } from 'pages/logs/utils/logsFilter';
import { stripQanServiceId } from 'utils/qanServiceId';

export interface StepQuery {
  language: 'sql' | 'promql' | 'json';
  text: string;
}

/** A page that shows what a step looked at, as a path inside the PMM UI. */
export interface StepLink {
  label: string;
  to: string;
}

const PROMQL_TOOLS = [
  'execute_prometheus_range_query',
  'execute_prometheus_instant_query',
  'pmm_metrics_snapshot',
];
const QUERY_ID_TOOLS = ['pmm_db_action', 'pmm_mysql_explain'];

const str = (v: unknown) =>
  typeof v === 'string' || typeof v === 'number' ? String(v).trim() : '';

const list = (v: unknown) =>
  str(v)
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean);

const oneLine = (s: string, max = 120) => {
  const line = s.replace(/\s+/g, ' ').trim();
  return line.length > max ? `${line.slice(0, max - 1)}…` : line;
};

/** A Prometheus tool time as an Explore time: relative seconds, unix seconds or RFC3339. */
const exploreTime = (v: unknown, fallback: string) => {
  const s = str(v);
  if (!s) return fallback;
  if (s.startsWith('now')) return s;
  if (/^-\d+(\.\d+)?$/.test(s)) return `now${Math.round(Number(s))}s`;
  if (/^\d+(\.\d+)?$/.test(s)) return String(Math.round(Number(s) * 1000));
  const ms = Date.parse(s);
  return Number.isNaN(ms) ? fallback : String(ms);
};

/** A Grafana Explore link that runs one query on the named datasource over the range. */
export const explore = (
  datasource: string,
  query: Record<string, unknown>,
  range: { from: string; to: string }
) => {
  const panes = {
    a: { datasource, queries: [{ refId: 'A', ...query }], range },
  };
  const params = new URLSearchParams({
    schemaVersion: '1',
    orgId: '1',
    panes: JSON.stringify(panes),
  });
  return `${PMM_NEW_NAV_GRAFANA_PATH}/explore?${params}`;
};

/** The exact query a step ran: its SQL or PromQL, or else its parameters. */
export const stepQuery = (step: ProgressStep): StepQuery | undefined => {
  const p = step.params ?? {};
  if (step.toolName === 'pmm_clickhouse_query' && str(p.query)) {
    return { language: 'sql', text: str(p.query) };
  }
  if (PROMQL_TOOLS.includes(step.toolName) && str(p.query)) {
    return { language: 'promql', text: str(p.query) };
  }
  // The model's description of the call is shown as its summary instead.
  const shown = Object.fromEntries(
    Object.entries(p).filter(
      ([k, v]) => k !== 'description' && v != null && v !== ''
    )
  );
  if (!Object.keys(shown).length) return undefined;
  return { language: 'json', text: JSON.stringify(shown, null, 2) };
};

/** One line saying what a step did. */
export const stepSummary = (step: ProgressStep): string | undefined => {
  const p = step.params ?? {};
  if (str(p.description)) return oneLine(str(p.description));
  if (str(p.query)) return oneLine(str(p.query));
  if (step.toolName === 'pmm_logs_patterns' && str(p.start)) {
    return oneLine(
      [str(p.sources) || str(p.node_names), str(p.text) && `"${str(p.text)}"`]
        .filter(Boolean)
        .join(' · ') || 'All logs'
    );
  }
  if (str(p.action))
    return oneLine(
      [str(p.action), str(p.table_name)].filter(Boolean).join(' ')
    );
  return step.description ? oneLine(step.description) : undefined;
};

/** Where to see what a step looked at, when PMM has a page for it; dashboards filter by service name. */
export const stepLink = (
  step: ProgressStep,
  nativeQan: boolean,
  serviceName?: string
): StepLink | undefined => {
  const p = step.params ?? {};
  if (step.toolName === 'pmm_clickhouse_query' && str(p.query)) {
    return {
      label: 'Open in Explore',
      to: explore(
        str(p.database) === 'otel' ? 'ClickHouse-OTEL' : 'ClickHouse',
        {
          rawSql: str(p.query),
          editorType: 'sql',
          format: 1,
          queryType: 'table',
        },
        { from: 'now-1h', to: 'now' }
      ),
    };
  }
  if (PROMQL_TOOLS.includes(step.toolName) && str(p.query)) {
    return {
      label: 'Open in Explore',
      to: explore(
        'Metrics',
        { expr: str(p.query), range: true },
        {
          from: exploreTime(p.start, 'now-1h'),
          to: exploreTime(p.end, 'now'),
        }
      ),
    };
  }
  if (step.toolName === 'pmm_logs_patterns') {
    const start = Date.parse(str(p.start));
    const end = Date.parse(str(p.end));
    if (Number.isNaN(start) || Number.isNaN(end)) return undefined;
    const severity = Number(p.min_severity);
    const params = searchParamsFromFilter({
      start: new Date(start).toISOString(),
      end: new Date(end).toISOString(),
      ...(list(p.node_names).length && { nodeNames: list(p.node_names) }),
      ...(list(p.sources).length && { sources: list(p.sources) }),
      ...(severity > 0 && { minSeverity: severity }),
      ...(str(p.text) && { text: str(p.text) }),
    });
    return { label: 'Open in Logs', to: `${PMM_LOGS_PATH}?${params}` };
  }
  if (QUERY_ID_TOOLS.includes(step.toolName) && str(p.query_id)) {
    const params = new URLSearchParams();
    if (nativeQan && str(p.service_id)) {
      params.set('filter_service_id', stripQanServiceId(str(p.service_id)));
    } else if (!nativeQan && serviceName) {
      params.set('var-service_name', serviceName);
    }
    params.set('filter_by', str(p.query_id));
    params.set('query_selected', 'true');
    params.set('details_tab', 'details');
    return {
      label: 'Open in QAN',
      to: nativeQan
        ? `${PMM_NEW_NAV_PATH}/qan?${params}`
        : `${PMM_NEW_NAV_GRAFANA_PATH}/d/pmm-qan/pmm-query-analytics?${params}`,
    };
  }
  if (step.toolName === 'pmm_render_grafana_panel' && str(p.dashboard_uid)) {
    const params = new URLSearchParams();
    if (str(p.panel_id)) params.set('viewPanel', str(p.panel_id));
    if (str(p.from)) params.set('from', str(p.from));
    if (str(p.to)) params.set('to', str(p.to));
    if (serviceName) params.set('var-service_name', serviceName);
    return {
      label: 'Open panel',
      to: `${PMM_NEW_NAV_GRAFANA_PATH}/d/${encodeURIComponent(str(p.dashboard_uid))}?${params}`,
    };
  }
  return undefined;
};

const snakeCase = (key: string) =>
  key.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`);

/** A step from a tool result saved with a chat; the API camelCases its keys. */
export const stepFromToolRow = (
  row: AdreMessageRow
): ProgressStep | undefined => {
  const json = row.toolResultJson as
    | {
        toolCallId?: string;
        result?: {
          status?: string;
          error?: string | null;
          params?: Record<string, unknown>;
        };
      }
    | undefined;
  if (!row.toolName && !json) return undefined;
  const result = json?.result;
  const params = result?.params
    ? Object.fromEntries(
        Object.entries(result.params).map(([k, v]) => [snakeCase(k), v])
      )
    : undefined;
  return {
    id: json?.toolCallId || String(row.id),
    toolName: row.toolName ?? '',
    status: 'done',
    ...(result?.status === 'error' && { failed: true }),
    ...(result?.error && { error: String(result.error) }),
    ...(params && { params }),
  };
};
