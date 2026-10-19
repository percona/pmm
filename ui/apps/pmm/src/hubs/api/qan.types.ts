export interface QanReportLabel {
  key: string;
  value: string[];
}

/** `POST /v1/qan/metrics:getReport` body, camelCase here and snake_case on the wire. */
export interface QanReportRequest {
  periodStartFrom: string;
  periodStartTo: string;
  groupBy: 'queryid';
  labels: QanReportLabel[];
  columns: string[];
  orderBy: string;
  mainMetric: string;
  offset: number;
  limit: number;
}

export interface QanMetricStats {
  rate?: number;
  cnt?: number;
  sum?: number;
  min?: number;
  max?: number;
  p99?: number;
  avg?: number;
  sumPerSec?: number;
}

export interface QanReportRow {
  rank?: number;
  /** The query id; empty on the total row. */
  dimension: string;
  database?: string;
  fingerprint?: string;
  numQueries?: number;
  qps?: number;
  load?: number;
  /** Keyed by column, camelCased by the API client: queryTime, numQueries and so on. */
  metrics?: Record<string, { stats?: QanMetricStats } | undefined>;
}

export interface QanReportResponse {
  totalRows?: number;
  offset?: number;
  limit?: number;
  rows?: QanReportRow[];
}
