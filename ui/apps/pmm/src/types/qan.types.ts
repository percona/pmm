export type QanGroupBy = 'queryid' | 'service_name' | 'database' | 'schema' | 'username' | 'client_host';

export type QanDetailsTab =
  | 'details'
  | 'examples'
  | 'explainPlan'
  | 'tables'
  | 'aiInsights';

export type QanDatabaseType = 'mysql' | 'postgresql' | 'mongodb' | 'unknown';

export interface QanLabelFilter {
  key: string;
  value: string[];
}

export interface QanLabelsMap {
  [key: string]: string[];
}

export interface QanFilterLabelValue {
  value: string;
  mainMetricPercent?: number;
  mainMetricPerSec?: number;
  checked?: boolean;
}

export interface QanFilterLabelGroup {
  name: QanFilterLabelValue[];
}

export interface QanFiltersResponse {
  labels: Record<string, QanFilterLabelGroup>;
}

export interface QanMetricPoint {
  timestamp: string;
  value: number;
}

export interface QanMetricCell {
  stats?: {
    avg?: number;
    sum?: number;
    min?: number;
    max?: number;
    rate?: number;
    sumPerSec?: number;
    sum_per_sec?: number;
    qps?: number;
  };
  sparkline?: QanMetricPoint[];
}

export interface QanReportRow {
  rank?: number;
  dimension?: string;
  database?: string;
  metrics?: Record<string, QanMetricCell>;
  sparkline?: QanMetricPoint[];
  fingerprint?: string;
  numQueries?: number;
  qps?: number;
  load?: number;
}

export interface QanGetReportRequest {
  columns: string[];
  groupBy: QanGroupBy;
  labels: QanLabelFilter[];
  limit: number;
  offset: number;
  orderBy: string;
  mainMetric: string;
  periodStartFrom: string;
  periodStartTo: string;
  search?: string;
}

export interface QanGetReportResponse {
  totalRows: number;
  offset: number;
  limit: number;
  rows: QanReportRow[];
}

export interface QanGetMetricsRequest {
  filterBy: string;
  groupBy: QanGroupBy;
  labels: QanLabelFilter[];
  periodStartFrom: string;
  periodStartTo: string;
  tables?: string[];
  totals?: boolean;
}

export interface QanMetricValues {
  data?: QanMetricPoint[];
  stats?: Record<string, number>;
}

export interface QanGetMetricsResponse {
  metrics?: Record<string, QanMetricValues>;
  textMetrics?: Record<string, string>;
  sparkline?: QanMetricPoint[];
  totals?: Record<string, QanMetricValues>;
  fingerprint?: string;
  metadata?: Record<string, unknown>;
}

export interface QanQueryExample {
  example?: string;
  exampleType?: string;
  exampleMetrics?: Record<string, unknown>;
  serviceId?: string;
  tables?: string[];
}

export interface QanGetExampleResponse {
  examples?: QanQueryExample[];
}

export interface QanExplainResponse {
  json?: string;
  classic?: string;
  visual?: string;
}

export interface QanGetSchemaResponse {
  tables?: Record<string, unknown>;
}

export interface QanHistogramItem {
  range: string;
  frequency: number;
}

export interface QanGetHistogramResponse {
  histogramItems?: QanHistogramItem[];
}

export interface QanMetricName {
  name: string;
  type: string;
}

export interface QanGetMetricNamesResponse {
  /** Metric key → human-readable label (qan-api2 map). Legacy array shape tolerated in UI. */
  data?: Record<string, string> | QanMetricName[];
}

export interface QanPanelState {
  from: string;
  to: string;
  columns: string[];
  labels: QanLabelsMap;
  pageNumber: number;
  pageSize: number;
  orderBy: string;
  queryId?: string;
  totals: boolean;
  querySelected: boolean;
  groupBy: QanGroupBy;
  openDetailsTab: QanDetailsTab;
  fingerprint?: string;
  database?: string;
  dimensionSearchText?: string;
  databaseType?: QanDatabaseType;
}
