import { LogFilter, LogLine } from 'api/logs';
import {
  PMM_ASSISTANT_PATH,
  PMM_BASE_PATH,
  PMM_LOGS_PATH,
  PMM_NEW_NAV_PATH,
} from 'lib/constants';
import { searchParamsFromFilter } from './logsFilter';

const PMM_QAN_PATH = `${PMM_NEW_NAV_PATH}/qan`;

// The service type each database preset logs for, and that type's summary dashboard.
const SERVICES: Record<string, { type: string; dashboard: string }> = {
  postgres: {
    type: 'postgresql',
    dashboard: 'postgresql-instance-summary/postgresql-instance-summary',
  },
  mongodb: {
    type: 'mongodb',
    dashboard: 'mongodb-instance-summary/mongodb-instance-summary',
  },
  mysql_error: {
    type: 'mysql',
    dashboard: 'mysql-instance-summary/mysql-instance-summary',
  },
  syslog_mysql_systemd: {
    type: 'mysql',
    dashboard: 'mysql-instance-summary/mysql-instance-summary',
  },
};

/** The service type a preset logs for, as /v1/management/services names it. */
export const serviceTypeOf = (preset: string) => SERVICES[preset]?.type;

/** The service's summary dashboard for a database line with a known service, else the node summary. */
export const dashboardUrl = (
  line: LogLine,
  fromMs: number,
  toMs: number,
  serviceName?: string
) => {
  const service = SERVICES[line.preset];
  const [path, variable] =
    service && serviceName
      ? [service.dashboard, { 'var-service_name': serviceName }]
      : [
          'node-instance-summary/node-summary',
          { 'var-node_name': line.nodeName },
        ];
  return `/graph/d/${path}?${new URLSearchParams({
    ...variable,
    from: String(fromMs),
    to: String(toMs),
  })}`;
};

// A PostgreSQL statement or execute line with a statement, as pmm-managed's statementOf reads it.
const QUERY_LINE =
  /^(?:duration: [0-9.]+ ms\s+)?(?:statement|execute [^:\n]*):[\s;]*[^\s;]/;

/** Whether a line logs a query QAN may have: a PostgreSQL statement or a MongoDB slow query, whose attr holds it. */
export const isQueryLine = (line: LogLine) =>
  (line.preset === 'postgres' && QUERY_LINE.test(line.body)) ||
  (line.preset === 'mongodb' &&
    line.body === 'Slow query' &&
    !!line.attributes.attr);

/** What a QAN link opens: a known query, else the queries of a known service, else of the node. */
export interface QanTarget {
  nodeName: string;
  serviceName?: string;
  queryId?: string;
  fingerprint?: string;
}

/** QAN for the window, in the native UI when it is enabled. */
export const qanUrl = (
  { nodeName, serviceName, queryId, fingerprint }: QanTarget,
  fromMs: number,
  toMs: number,
  native: boolean
) => {
  const [label, value] = serviceName
    ? ['service_name', serviceName]
    : ['node_name', nodeName];
  const params = new URLSearchParams({
    [native ? `filter_${label}` : `var-${label}`]: value,
    from: String(fromMs),
    to: String(toMs),
  });
  if (queryId) {
    params.set('filter_by', queryId);
    params.set('query_selected', 'true');
    params.set('details_tab', 'details');
    // Native QAN shows the details of the fingerprint in the URL, as its own rows link them.
    if (native && fingerprint) params.set('fingerprint', fingerprint);
  }
  return native
    ? `${PMM_QAN_PATH}?${params}`
    : `/graph/d/pmm-qan/pmm-query-analytics?${params}`;
};

export const assistantAskUrl = (
  line: LogLine,
  fromMs: number,
  toMs: number
) => {
  const ask =
    `Explain this ${line.severityText || 'log'} line from ${line.source} on node ${line.nodeName} ` +
    `(${line.fileName}, ${line.time}) and look for related problems between ` +
    `${new Date(fromMs).toISOString()} and ${new Date(toMs).toISOString()}:\n\n${line.body}`;
  return `${PMM_ASSISTANT_PATH}?${new URLSearchParams({ ask })}`;
};

const LINE_LINK_WINDOW_MS = 5 * 60_000;

/** An absolute link to the Logs page five minutes either side of a line, with the page's other filters. */
export const lineLink = (filter: LogFilter, line: LogLine, origin: string) => {
  const ms = Date.parse(line.time);
  const params = searchParamsFromFilter({
    ...filter,
    start: new Date(ms - LINE_LINK_WINDOW_MS).toISOString(),
    end: new Date(ms + LINE_LINK_WINDOW_MS).toISOString(),
  });
  return `${origin}${PMM_BASE_PATH}${PMM_LOGS_PATH}/lines?${params}`;
};
