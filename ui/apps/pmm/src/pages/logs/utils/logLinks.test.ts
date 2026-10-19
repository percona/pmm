import { LogLine } from 'api/logs';
import {
  assistantAskUrl,
  dashboardUrl,
  isQueryLine,
  lineLink,
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
  patternHash: '',
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

  it('opens the query in Grafana QAN, filtered by its service', () => {
    expect(
      qanUrl(
        {
          nodeName: 'pg-node',
          serviceName: 'pg-svc',
          queryId: 'q1',
          fingerprint: 'SELECT $1',
        },
        from,
        to,
        false
      )
    ).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_name=pg-svc&from=1792232100000&to=1792233900000&filter_by=q1&query_selected=true&details_tab=details'
    );
  });

  it('opens the query in native QAN when it is enabled', () => {
    expect(
      qanUrl(
        {
          nodeName: 'pg-node',
          serviceName: 'pg-svc',
          queryId: 'q1',
          fingerprint: 'SELECT $1',
        },
        from,
        to,
        true
      )
    ).toBe(
      '/qan?filter_service_name=pg-svc&from=1792232100000&to=1792233900000&filter_by=q1&query_selected=true&details_tab=details&fingerprint=SELECT+%241'
    );
  });

  it('lists the queries of the service, or of the node, without a known query', () => {
    expect(
      qanUrl({ nodeName: 'pg-node', serviceName: 'pg-svc' }, from, to, false)
    ).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_name=pg-svc&from=1792232100000&to=1792233900000'
    );
    expect(qanUrl({ nodeName: 'pg-node' }, from, to, true)).toBe(
      '/qan?filter_node_name=pg-node&from=1792232100000&to=1792233900000'
    );
  });

  it('knows which lines log a query', () => {
    const logsQuery = (body: string, preset = 'postgres') =>
      isQueryLine({ ...line, body, preset });
    expect(logsQuery('statement: SELECT 1')).toBe(true);
    expect(
      logsQuery('duration: 1003.112 ms  statement: SELECT pg_sleep(1)')
    ).toBe(true);
    expect(logsQuery('execute <unnamed>: SELECT pg_database_size($1)')).toBe(
      true
    );
    expect(logsQuery('statement: \n\t\tSELECT\n\t\t  datid')).toBe(true);
    expect(logsQuery('statement: ;')).toBe(false);
    expect(logsQuery('duration: 0.512 ms')).toBe(false);
    expect(logsQuery('deadlock detected')).toBe(false);
    expect(logsQuery('statement: SELECT 1', 'mysql_error')).toBe(false);

    // A MongoDB slow query logs its operation in attr.
    const slow = { ...line, preset: 'mongodb', body: 'Slow query' };
    expect(
      isQueryLine({ ...slow, attributes: { attr: '{"type":"update"}' } })
    ).toBe(true);
    expect(isQueryLine(slow)).toBe(false);
    expect(
      isQueryLine({
        ...slow,
        body: 'Connection accepted',
        attributes: { attr: '{}' },
      })
    ).toBe(false);
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

  it('links to the Logs page five minutes around a line, keeping the filters', () => {
    expect(
      lineLink(
        {
          start: '2026-10-17T09:00:00.000Z',
          end: '2026-10-17T11:00:00.000Z',
          sources: ['pg-node'],
        },
        line,
        'https://pmm.example'
      )
    ).toBe(
      'https://pmm.example/pmm-ui/logs/lines?from=1792232700000&to=1792233300000&source=pg-node'
    );
  });
});
