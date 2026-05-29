import type { QanMetricPoint } from 'types/qan.types';

const COLUMN_LABELS: Record<string, string> = {
  load: 'Load',
  num_queries: 'Query Count',
  query_time: 'Query Time',
};

export function qanColumnLabel(column: string): string {
  return (
    COLUMN_LABELS[column] ??
    column.replace(/_/g, ' ').replace(/\b\w/g, (c) => c.toUpperCase())
  );
}

export function formatQanMetricFigure(column: string, value: unknown): string {
  if (value == null || Number.isNaN(value)) return '—';
  if (typeof value !== 'number') return String(value);

  const abs = Math.abs(value);
  const digits = abs >= 100 ? 2 : abs >= 1 ? 2 : abs >= 0.01 ? 2 : 4;
  const num = value.toFixed(digits);

  if (column === 'load') return `${num} load`;
  if (column === 'num_queries') return `${num} QPS`;
  if (column === 'query_time' || column.endsWith('_time')) {
    // qan-api2 returns time-metric avg in seconds (Grafana QAN humanize semantics).
    return `${(value * 1000).toFixed(2)} ms`;
  }
  return num;
}

/** Bucket sparkline points into bar segments (Figma “cost graph” style). */
export function bucketSparklineValues(
  points: QanMetricPoint[] | undefined,
  buckets = 6
): number[] {
  if (!points?.length) return [];
  const vals = points.map((p) => p.value);
  const chunk = Math.max(1, Math.ceil(vals.length / buckets));
  const result: number[] = [];
  for (let i = 0; i < buckets; i += 1) {
    const slice = vals.slice(i * chunk, (i + 1) * chunk);
    result.push(slice.length ? Math.max(...slice) : 0);
  }
  return result;
}
