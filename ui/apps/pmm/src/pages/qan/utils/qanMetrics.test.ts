import {
  qanMetricCell,
  qanMetricDisplayValue,
  qanMetricKey,
} from './qanMetrics';
import type { QanReportRow } from 'types/qan.types';

describe('qanMetrics', () => {
  const row: QanReportRow = {
    load: 0.05,
    numQueries: 100,
    metrics: {
      numQueries: {
        stats: { sum: 500, sumPerSec: 1.2 },
      },
      queryTime: {
        stats: { avg: 0.33, sumPerSec: 0.05 },
      },
      load: {
        stats: { sumPerSec: 0.05 },
      },
    },
  };

  it('reads camelCase metric keys from axios responses', () => {
    expect(qanMetricCell(row.metrics, 'num_queries')?.stats?.sum).toBe(500);
    expect(qanMetricCell(row.metrics, 'query_time')?.stats?.avg).toBe(0.33);
  });

  it('displays rate metrics for counters and avg for time metrics', () => {
    expect(qanMetricDisplayValue('num_queries', row)).toBe(1.2);
    expect(qanMetricDisplayValue('query_time', row)).toBe(0.33);
    expect(qanMetricDisplayValue('load', row)).toBe(0.05);
  });

  it('converts snake_case column ids to camelCase metric keys', () => {
    expect(qanMetricKey('num_queries')).toBe('numQueries');
  });
});
