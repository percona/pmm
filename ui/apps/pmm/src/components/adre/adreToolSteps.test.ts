import type { ProgressStep } from 'hooks/useAdreChat';
import {
  stepFromToolRow,
  stepLink,
  stepQuery,
  stepSummary,
} from './adreToolSteps';

const step = (
  toolName: string,
  params: Record<string, unknown>
): ProgressStep => ({
  id: 't1',
  toolName,
  status: 'done',
  params,
});

const explorePane = (to: string) => {
  const url = new URL(to, 'https://pmm.example');
  expect(url.pathname).toBe('/graph/explore');
  return Object.values(
    JSON.parse(url.searchParams.get('panes') ?? '{}')
  )[0] as {
    datasource: string;
    queries: Array<Record<string, unknown>>;
    range: { from: string; to: string };
  };
};

describe('adreToolSteps', () => {
  it('opens a ClickHouse query in Explore on the database it ran against', () => {
    const logs = step('pmm_clickhouse_query', {
      database: 'otel',
      query: "SELECT count() FROM logs WHERE ServiceName = 'pg1'",
    });

    expect(stepQuery(logs)).toEqual({
      language: 'sql',
      text: "SELECT count() FROM logs WHERE ServiceName = 'pg1'",
    });
    const link = stepLink(logs, false);
    expect(link?.label).toBe('Open in Explore');
    const pane = explorePane(link!.to);
    expect(pane.datasource).toBe('ClickHouse-OTEL');
    expect(pane.queries[0]).toMatchObject({
      rawSql: "SELECT count() FROM logs WHERE ServiceName = 'pg1'",
      editorType: 'sql',
    });
    expect(
      explorePane(
        stepLink(
          step('pmm_clickhouse_query', { database: 'pmm', query: 'SELECT 1' }),
          false
        )!.to
      ).datasource
    ).toBe('ClickHouse');
  });

  it('opens PromQL in Explore over the range the tool used', () => {
    const relative = step('execute_prometheus_range_query', {
      query: 'up{service_name="rs1"}',
      start: '-7200',
      description: 'Replica set state over the last 2h',
    });

    expect(stepSummary(relative)).toBe('Replica set state over the last 2h');
    expect(stepQuery(relative)).toEqual({
      language: 'promql',
      text: 'up{service_name="rs1"}',
    });
    const pane = explorePane(stepLink(relative, false)!.to);
    expect(pane.datasource).toBe('Metrics');
    expect(pane.queries[0]).toMatchObject({ expr: 'up{service_name="rs1"}' });
    expect(pane.range).toEqual({ from: 'now-7200s', to: 'now' });

    const absolute = step('pmm_metrics_snapshot', {
      query: 'up',
      start: '2026-10-17T20:41:24Z',
      end: '2026-10-17T21:11:25Z',
    });
    expect(explorePane(stepLink(absolute, false)!.to).range).toEqual({
      from: String(Date.parse('2026-10-17T20:41:24Z')),
      to: String(Date.parse('2026-10-17T21:11:25Z')),
    });
  });

  it('opens log patterns on the Logs page with the same filter', () => {
    const link = stepLink(
      step('pmm_logs_patterns', {
        start: '2026-10-17T20:41:24Z',
        end: '2026-10-17T21:11:25Z',
        sources: 'pdpgsql_pmm_17_1',
        node_names: 'n1,n2',
        min_severity: 17,
        text: 'shop_app',
      }),
      false
    );
    const url = new URL(link!.to, 'https://pmm.example');

    expect(link?.label).toBe('Open in Logs');
    expect(url.pathname).toBe('/logs');
    expect(url.searchParams.get('from')).toBe(
      String(Date.parse('2026-10-17T20:41:24Z'))
    );
    expect(url.searchParams.getAll('node')).toEqual(['n1', 'n2']);
    expect(url.searchParams.get('source')).toBe('pdpgsql_pmm_17_1');
    expect(url.searchParams.get('severity')).toBe('17');
    expect(url.searchParams.get('q')).toBe('shop_app');
  });

  it('opens an explained query in QAN and a rendered panel on its dashboard', () => {
    const explain = step('pmm_db_action', {
      action: 'mysql_explain',
      service_id: '4ddf1174',
      query_id: 'b589a24c',
    });

    expect(stepLink(explain, true)?.to).toBe(
      '/qan?filter_service_id=4ddf1174&filter_by=b589a24c&query_selected=true&details_tab=details'
    );
    expect(stepLink(explain, false)?.to).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?filter_by=b589a24c&query_selected=true&details_tab=details'
    );
    expect(
      stepLink(
        step('pmm_db_action', {
          action: 'mysql_show_index',
          table_name: 'orders',
        }),
        false
      )
    ).toBeUndefined();

    expect(
      stepLink(
        step('pmm_render_grafana_panel', {
          dashboard_uid: 'mysql-instance-summary',
          panel_id: '92',
          from: 'now-1h',
          to: 'now',
        }),
        false
      )
    ).toEqual({
      label: 'Open panel',
      to: '/graph/d/mysql-instance-summary?viewPanel=92&from=now-1h&to=now',
    });
  });

  it('filters panels and Grafana QAN by the service the tool looked at', () => {
    const panel = step('pmm_render_grafana_panel', {
      dashboard_uid: 'mysql-instance-summary',
      panel_id: '92',
      service_id: '4ddf1174',
    });
    const explain = step('pmm_mysql_explain', {
      service_id: '4ddf1174',
      query_id: 'b589a24c',
    });

    expect(stepLink(panel, false, 'ps-8.4')?.to).toBe(
      '/graph/d/mysql-instance-summary?viewPanel=92&var-service_name=ps-8.4'
    );
    expect(stepLink(explain, false, 'ps-8.4')?.to).toBe(
      '/graph/d/pmm-qan/pmm-query-analytics?var-service_name=ps-8.4&filter_by=b589a24c&query_selected=true&details_tab=details'
    );
  });

  it('shows the other tools by their parameters', () => {
    const list = step('pmm_list_services', {
      service_type: 'mysql',
      description: 'x',
    });

    expect(stepQuery(list)).toEqual({
      language: 'json',
      text: '{\n  "service_type": "mysql"\n}',
    });
    expect(stepLink(list, false)).toBeUndefined();
    expect(stepQuery(step('pmm_list_nodes', {}))).toBeUndefined();
  });

  it('reads a saved tool row, whose keys came back camelCased', () => {
    expect(
      stepFromToolRow({
        id: 150,
        conversationId: 1,
        role: 'tool',
        content: '',
        createdAt: '2026-10-17T21:00:00Z',
        toolName: 'pmm_logs_patterns',
        toolResultJson: {
          toolCallId: 'call_1',
          result: {
            status: 'error',
            error: 'ClickHouse timed out',
            params: { minSeverity: 17, nodeNames: 'n1' },
          },
        },
      })
    ).toEqual({
      id: 'call_1',
      toolName: 'pmm_logs_patterns',
      status: 'done',
      failed: true,
      error: 'ClickHouse timed out',
      params: { min_severity: 17, node_names: 'n1' },
    });
  });
});
