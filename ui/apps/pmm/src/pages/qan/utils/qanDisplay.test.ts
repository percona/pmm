import {
  bucketSparklineValues,
  formatQanMetricFigure,
  qanColumnLabel,
} from './qanDisplay';

describe('qanDisplay', () => {
  it('maps default column ids to Figma labels', () => {
    expect(qanColumnLabel('load')).toBe('Load');
    expect(qanColumnLabel('num_queries')).toBe('Query Count');
    expect(qanColumnLabel('query_time')).toBe('Query Time');
  });

  it('formats metric figures with units', () => {
    expect(formatQanMetricFigure('load', 8.5)).toBe('8.50 load');
    expect(formatQanMetricFigure('num_queries', 0.05)).toBe('0.05 QPS');
    expect(formatQanMetricFigure('query_time', 2.8)).toBe('2800.00 ms');
  });

  it('formats query_time avg in seconds as milliseconds', () => {
    expect(formatQanMetricFigure('query_time', 0.33)).toBe('330.00 ms');
    expect(formatQanMetricFigure('query_time', 0.001)).toBe('1.00 ms');
  });

  it('buckets sparkline points for bar segments', () => {
    const buckets = bucketSparklineValues(
      [
        { timestamp: '1', value: 1 },
        { timestamp: '2', value: 2 },
        { timestamp: '3', value: 9 },
        { timestamp: '4', value: 3 },
      ],
      2
    );
    expect(buckets).toEqual([2, 9]);
  });
});
