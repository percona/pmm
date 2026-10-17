import { LogLine } from 'api/logs';
import { PMM_ASSISTANT_PATH } from 'lib/constants';

/** Presets of database logs, the ones QAN has queries for. */
export const DATABASE_PRESETS = [
  'mysql_error',
  'postgres',
  'mongodb',
  'syslog_mysql_systemd',
];

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

const MIN_SEARCH = 10;
const MAX_SEARCH = 80;

/** The start of a logged PostgreSQL statement up to its first literal, which QAN fingerprints replace. */
const statementSearch = (body: string) => {
  const query = body.match(/^(?:statement|execute [^:]*):[ \t]*([^\n]*)/)?.[1];
  const start = query
    ?.split(/['"0-9]/, 1)[0]
    .replace(/\W+$/, '')
    .slice(0, MAX_SEARCH);
  return start && start.length >= MIN_SEARCH ? start : undefined;
};

/** Grafana QAN for the line's node and window, searched for the line's statement when it logs one. */
export const qanUrl = (line: LogLine, fromMs: number, toMs: number) => {
  const params = new URLSearchParams({
    'var-node_name': line.nodeName,
    from: String(fromMs),
    to: String(toMs),
  });
  const search = statementSearch(line.body);
  if (search) params.set('dimensionSearchText', search);
  return `/graph/d/pmm-qan/pmm-query-analytics?${params}`;
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
