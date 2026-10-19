import {
  chartEmbedUrl,
  chartSpecFromBlock,
  panelEmbedUrl,
  panelLinksByImage,
  rangeMs,
} from './adreChart.utils';

const params = (url: string) =>
  Object.fromEntries(new URL(url, 'https://pmm.test').searchParams);

describe('chartSpecFromBlock', () => {
  it('reads the attributes and the query', () => {
    expect(
      chartSpecFromBlock(
        'ds="ClickHouse" title="Query time by service" from="2026-10-19T20:00:00Z" to="2026-10-19T21:00:00Z" unit=s',
        'SELECT $__timeInterval(period_start) AS t, service_name, sum(m_query_time_sum) AS q\nFROM pmm.metrics\n'
      )
    ).toEqual({
      ds: 'ClickHouse',
      title: 'Query time by service',
      from: '2026-10-19T20:00:00Z',
      to: '2026-10-19T21:00:00Z',
      unit: 's',
      query:
        'SELECT $__timeInterval(period_start) AS t, service_name, sum(m_query_time_sum) AS q\nFROM pmm.metrics',
    });
  });

  it('defaults to Metrics over the last hour and needs a query', () => {
    expect(chartSpecFromBlock(undefined, 'up')).toEqual({
      ds: 'Metrics',
      from: 'now-1h',
      to: 'now',
      query: 'up',
    });
    expect(chartSpecFromBlock('ds="Metrics"', '  \n')).toBeUndefined();
  });
});

describe('rangeMs', () => {
  const now = Date.parse('2026-10-19T21:00:00Z');

  it('takes absolute and relative times', () => {
    expect(rangeMs('2026-10-19T20:00:00Z', 'now', now)).toEqual({
      from: Date.parse('2026-10-19T20:00:00Z'),
      to: now,
    });
    expect(rangeMs('now-6h', 'now-30m', now)).toEqual({
      from: now - 6 * 3600_000,
      to: now - 30 * 60_000,
    });
  });

  it('refuses a range that ends before it starts or does not parse', () => {
    expect(rangeMs('now', 'now-1h', now)).toBeUndefined();
    expect(rangeMs('yesterday', 'now', now)).toBeUndefined();
  });
});

describe('chartEmbedUrl', () => {
  const range = { from: 1000, to: 2000 };

  it('passes a PromQL chart to the Metrics panel for its unit', () => {
    const url = chartEmbedUrl(
      {
        ds: 'Metrics',
        title: 'Handlers',
        from: '',
        to: '',
        unit: 'ops',
        legend: '{{service_name}} · {{handler}}',
        query: 'sum by (handler) (rate(x{a=~"b|c\\\\d"}[5m]))',
      },
      range
    );
    expect(url.startsWith('/graph/d-solo/pmm-assistant-chart/')).toBe(true);
    expect(params(url)).toEqual({
      orgId: '1',
      panelId: '7',
      from: '1000',
      to: '2000',
      theme: 'dark',
      'var-ds': 'Metrics',
      'var-title': 'Handlers',
      'var-legend': '{{service_name}} · {{handler}}',
      'var-query': 'sum by (handler) (rate(x{a=~"b|c\\\\d"}[5m]))',
    });
  });

  it('passes SQL to the ClickHouse panel, and an unknown unit to the plain one', () => {
    const p = params(
      chartEmbedUrl(
        {
          ds: 'ClickHouse-OTEL',
          from: '',
          to: '',
          unit: 'furlongs',
          query: 'SELECT 1',
        },
        range
      )
    );
    expect(p.panelId).toBe('11');
    expect(p['var-chds']).toBe('ClickHouse-OTEL');
    expect(p['var-ds']).toBeUndefined();
    expect(p['var-title']).toBe('Chart');
  });
});

const handlersLink =
  'https://pmm.test/graph/d/mysql-instance-summary?from=1&to=2&var-service_name=ps-1&viewPanel=8';

describe('panelEmbedUrl', () => {
  it('turns a dashboard panel link into its embedded panel', () => {
    const url = panelEmbedUrl(handlersLink);
    expect(url?.startsWith('/graph/d-solo/mysql-instance-summary?')).toBe(true);
    expect(params(url!)).toEqual({
      from: '1',
      to: '2',
      'var-service_name': 'ps-1',
      panelId: '8',
      orgId: '1',
      theme: 'dark',
    });
  });

  it('needs a dashboard link that names a panel', () => {
    expect(
      panelEmbedUrl('https://pmm.test/graph/d/mysql-instance-summary?from=1')
    ).toBeUndefined();
    expect(
      panelEmbedUrl('https://pmm.test/graph/explore?left=x')
    ).toBeUndefined();
  });
});

describe('panelLinksByImage', () => {
  it('pairs each panel image with the dashboard link right after it', () => {
    const blob =
      'https://pmm.test/v1/grafana/render/blob/' + 'a'.repeat(64) + '.png';
    const md = `### Handlers\n![MySQL Handlers](${blob})\n[Open in Grafana](${handlersLink})\n\nOther ![x](/img.png) [link](https://pmm.test/graph/d/home)`;
    expect(panelLinksByImage(md)).toEqual(new Map([[blob, handlersLink]]));
  });
});
