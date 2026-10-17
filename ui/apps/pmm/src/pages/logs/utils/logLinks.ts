import { LogLine } from 'api/logs';
import { PMM_ASSISTANT_PATH } from 'lib/constants';
import { buildNativeQanPath } from 'utils/nativeQanNav';

/** Presets of database logs, the ones QAN has queries for. */
export const DATABASE_PRESETS = [
  'mysql_error',
  'postgres',
  'mongodb',
  'syslog_mysql_systemd',
];

export const nodeDashboardUrl = (
  nodeName: string,
  fromMs: number,
  toMs: number
) =>
  `/graph/d/node-instance-summary/node-summary?${new URLSearchParams({
    'var-node_name': nodeName,
    from: String(fromMs),
    to: String(toMs),
  })}`;

export const qanUrl = (nodeName: string, fromMs: number, toMs: number) =>
  buildNativeQanPath({
    from: String(fromMs),
    to: String(toMs),
    filter_node_name: nodeName,
  });

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
