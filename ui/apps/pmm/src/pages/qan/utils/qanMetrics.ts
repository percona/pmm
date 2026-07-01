import type { QanMetricCell, QanMetricPoint, QanReportRow } from 'types/qan.types';

/** axios-case-converter: API metric keys become camelCase (`numQueries`). */
export function qanMetricKey(column: string): string {
  return column.replace(/_([a-z])/g, (_, c: string) => c.toUpperCase());
}

type MetricStats = QanMetricCell['stats'] & {
  sumPerSec?: number;
  sum_per_sec?: number;
  qps?: number;
  cnt?: number;
};

export function qanMetricCell(
  metrics: QanReportRow['metrics'] | undefined,
  column: string
): QanMetricCell | undefined {
  if (!metrics) return undefined;
  return metrics[column] ?? metrics[qanMetricKey(column)];
}

/** Display value for overview columns (Grafana QAN semantics). */
export function qanMetricDisplayValue(column: string, row: QanReportRow): number | undefined {
  const stats = qanMetricCell(row.metrics, column)?.stats as MetricStats | undefined;
  const isTime = column.endsWith('_time') || column.endsWith('Time');

  if (isTime) {
    return stats?.avg ?? stats?.sumPerSec ?? stats?.sum_per_sec;
  }

  const rate =
    stats?.qps ?? stats?.sumPerSec ?? stats?.sum_per_sec ?? stats?.sum ?? stats?.rate;

  if (rate != null && !Number.isNaN(rate)) return rate;

  if (column === 'load' || qanMetricKey(column) === 'load') return row.load;
  if (column === 'num_queries' || qanMetricKey(column) === 'numQueries') {
    return row.numQueries ?? row.qps;
  }

  return undefined;
}

export function qanMetricSparkline(
  metrics: QanReportRow['metrics'] | undefined,
  column: string
): QanMetricPoint[] | undefined {
  return qanMetricCell(metrics, column)?.sparkline;
}
