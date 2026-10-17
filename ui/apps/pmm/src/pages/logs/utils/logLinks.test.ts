import { LogLine } from 'api/logs';
import {
  assistantAskUrl,
  DATABASE_PRESETS,
  nodeDashboardUrl,
  qanUrl,
} from './logLinks';

const line: LogLine = {
  time: '2026-10-17T10:30:00.000Z',
  nodeName: 'pg-node',
  source: 'pg-node',
  preset: 'postgres',
  fileName: 'postgresql-17-main.log',
  severityNumber: 17,
  severityText: 'ERROR',
  body: 'deadlock detected',
  bodyTruncated: false,
  attributes: {},
};
const from = 1792232100000;
const to = 1792233900000;

describe('logLinks', () => {
  it('links to the node dashboard for the window', () => {
    expect(nodeDashboardUrl('pg-node', from, to)).toBe(
      '/graph/d/node-instance-summary/node-summary?var-node_name=pg-node&from=1792232100000&to=1792233900000'
    );
  });

  it('links to QAN filtered by node for the window', () => {
    expect(qanUrl('pg-node', from, to)).toBe(
      '/pmm-ui/qan?from=1792232100000&to=1792233900000&filter_node_name=pg-node'
    );
  });

  it('asks the Assistant about the line', () => {
    const url = assistantAskUrl(line, from, to);
    expect(url.startsWith('/assistant?ask=')).toBe(true);
    const ask = new URLSearchParams(url.split('?')[1]).get('ask') ?? '';
    expect(ask).toContain('deadlock detected');
    expect(ask).toContain('pg-node');
    expect(ask).toContain('2026-10-17T10:15:00.000Z');
    expect(ask).toContain('2026-10-17T10:45:00.000Z');
  });

  it('lists the database presets', () => {
    expect(DATABASE_PRESETS).toEqual([
      'mysql_error',
      'postgres',
      'mongodb',
      'syslog_mysql_systemd',
    ]);
  });
});
