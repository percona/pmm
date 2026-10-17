import { LogLine } from 'api/logs';
import {
  assistantAskUrl,
  DATABASE_PRESETS,
  dashboardUrl,
  qanUrl,
  serviceTypeOf,
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
  it('links a database line to its service dashboard', () => {
    expect(dashboardUrl(line, from, to, 'pg-svc')).toBe(
      '/graph/d/postgresql-instance-summary/postgresql-instance-summary?var-service_name=pg-svc&from=1792232100000&to=1792233900000'
    );
  });

  it('links to the node dashboard without a service', () => {
    expect(dashboardUrl(line, from, to)).toBe(
      '/graph/d/node-instance-summary/node-summary?var-node_name=pg-node&from=1792232100000&to=1792233900000'
    );
    expect(
      dashboardUrl({ ...line, preset: 'nginx_error' }, from, to, 'pg-svc')
    ).toContain('/graph/d/node-instance-summary/');
  });

  it('knows which service type a preset logs for', () => {
    expect(serviceTypeOf('postgres')).toBe('postgresql');
    expect(serviceTypeOf('mongodb')).toBe('mongodb');
    expect(serviceTypeOf('mysql_error')).toBe('mysql');
    expect(serviceTypeOf('nginx_error')).toBeUndefined();
  });

  it('links to Grafana QAN filtered by node for the window', () => {
    expect(qanUrl({ ...line, body: 'checkpoint starting' }, from, to)).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-node_name=pg-node&from=1792232100000&to=1792233900000'
    );
  });

  it('searches QAN for the start of a logged statement', () => {
    const search = (body: string) =>
      new URLSearchParams(
        qanUrl({ ...line, body }, from, to).split('?')[1]
      ).get('dimensionSearchText');
    expect(
      search(
        'statement: SELECT id, customer_id FROM orders WHERE id = 1098292;'
      )
    ).toBe('SELECT id, customer_id FROM orders WHERE id');
    expect(search('execute <unnamed>: SELECT pg_database_size($1)')).toBe(
      'SELECT pg_database_size'
    );
    expect(search('statement: ;')).toBeNull();
    // QAN keeps a query's line breaks, so a query starting on the next line is not searched.
    expect(search('statement: \n\t\tSELECT\n\t\t  datid')).toBeNull();
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
