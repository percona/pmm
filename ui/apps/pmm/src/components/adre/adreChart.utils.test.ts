import {
  chartSpecFromBlock,
  formatValue,
  framesToSeries,
  rangeMs,
  type DataFrameJSON,
} from './adreChart.utils';

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

const timeField = { name: 'Time', type: 'time' };

describe('framesToSeries', () => {
  it('lines up Prometheus frames, one series per frame, named by labels', () => {
    const frames: DataFrameJSON[] = [
      {
        schema: {
          fields: [
            timeField,
            { name: 'Value', type: 'number', labels: { node_name: 'ps-1' } },
          ],
        },
        data: {
          values: [
            [1000, 2000],
            [1, 2],
          ],
        },
      },
      {
        schema: {
          fields: [
            timeField,
            { name: 'Value', type: 'number', labels: { node_name: 'ps-2' } },
          ],
        },
        data: {
          values: [
            [2000, 3000],
            [5, 6],
          ],
        },
      },
    ];

    expect(framesToSeries(frames, 10)).toEqual({
      times: [1000, 2000, 3000],
      series: [
        { label: 'ps-2', data: [null, 5, 6] },
        { label: 'ps-1', data: [1, 2, null] },
      ],
      hidden: 0,
    });
  });

  it('splits a wide ClickHouse frame into its value columns and keeps the largest', () => {
    const frames: DataFrameJSON[] = [
      {
        schema: {
          fields: [
            { name: 't', type: 'time' },
            {
              name: 'lines',
              type: 'number',
              labels: { SeverityText: 'ERROR' },
            },
            { name: 'lines', type: 'number', labels: { SeverityText: 'INFO' } },
            { name: 'lines', type: 'number', labels: { SeverityText: 'WARN' } },
          ],
        },
        data: {
          values: [
            [1000, 2000],
            [3, 9],
            [50, 40],
            [1, 0],
          ],
        },
      },
    ];

    expect(framesToSeries(frames, 2)).toEqual({
      times: [1000, 2000],
      series: [
        { label: 'INFO', data: [50, 40] },
        { label: 'ERROR', data: [3, 9] },
      ],
      hidden: 1,
    });
  });

  it('names a series by all its labels when there are several, else by the field', () => {
    const frames: DataFrameJSON[] = [
      {
        schema: {
          fields: [
            timeField,
            {
              name: 'Value',
              type: 'number',
              labels: { node_name: 'ps-1', mode: 'user' },
            },
            { name: 'total', type: 'number' },
          ],
        },
        data: {
          values: [[1000], [1], [2]],
        },
      },
    ];

    expect(framesToSeries(frames, 10).series.map((s) => s.label)).toEqual([
      'total',
      'node_name=ps-1, mode=user',
    ]);
  });
});

describe('formatValue', () => {
  it('formats each unit the Assistant may pick', () => {
    expect(formatValue(42.345, 'percent')).toBe('42.3%');
    expect(formatValue(0.5, 'percentunit')).toBe('50.0%');
    expect(formatValue(1536, 'bytes')).toBe('1.5 KiB');
    expect(formatValue(3 * 1024 ** 3, 'bytes')).toBe('3.0 GiB');
    expect(formatValue(0.25, 's')).toBe('250 ms');
    expect(formatValue(12.5, 's')).toBe('12.5 s');
    expect(formatValue(2500, 'ms')).toBe('2.5 s');
    expect(formatValue(1200, 'ops')).toBe('1.2K/s');
    expect(formatValue(1234567, undefined)).toBe('1.23M');
    expect(formatValue(null, 's')).toBe('');
  });
});
