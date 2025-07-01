import { api } from './api';

export interface QANLabel {
  key: string;
  value: string[];
}

export interface QANReportRequest {
  period_start_from: string; // ISO date string
  period_start_to: string;   // ISO date string
  group_by?: string;         // Default: 'queryid'
  order_by?: string;         // Default: '-load'
  limit?: number;            // Default: 10
  offset?: number;           // Default: 0
  labels?: QANLabel[];
  columns?: string[];
}

export interface QANMetric {
  stats: {
    rate?: number;
    cnt?: number;
    sum?: number;
    sum_per_sec?: number;
    sumPerSec?: number;
    min?: number;
    max?: number;
    avg?: number;
    p99?: number;
  };
}

export interface QANSparklinePoint {
  point?: number;
  timeFrame?: number;
  timestamp?: string;
  load?: number;
  numQueriesPerSec?: number;
  numQueriesWithErrorsPerSec?: number;
  numQueriesWithWarningsPerSec?: number;
  mQueryTimeSumPerSec?: number;
  mLockTimeSumPerSec?: number;
  mRowsSentSumPerSec?: number;
  mRowsExaminedSumPerSec?: number;
  // Add other sparkline fields as needed
  [key: string]: any; // Allow for additional fields
}

export interface QANRow {
  rank: number;
  dimension: string;
  database: string;
  fingerprint: string;
  num_queries: number;
  numQueries: number;
  qps: number;
  load: number;
  metrics: Record<string, QANMetric>;
  sparkline?: QANSparklinePoint[];
}

export interface QANReportResponse {
  total_rows: number;
  offset: number;
  limit: number;
  rows: QANRow[];
  is_total_estimated?: boolean; // Indicates whether total_rows is an estimate or exact count
}

export interface QANMetricsNamesResponse {
  data: Record<string, string>;
}

export interface QANFilterValue {
  value: string;
  main_metric_percent: number;
  main_metric_per_sec: number;
}

export interface QANFilterLabel {
  name: QANFilterValue[];
}

export interface QANFiltersRequest {
  period_start_from: string; // ISO date string
  period_start_to: string;   // ISO date string
  main_metric_name?: string; // Default: 'm_query_time_sum'
  labels?: QANLabel[];
}

export interface QANFiltersResponse {
  labels: Record<string, QANFilterLabel>;
}

export const getQANReport = async (request: QANReportRequest): Promise<QANReportResponse> => {
  try {
    const response = await api.post<any>('/qan/metrics:getReport', request);
    const data = response.data;
    
    // Handle potential field name variations and missing total_rows
    const result: QANReportResponse = {
      // Try both snake_case and camelCase variants, fallback to calculating from rows
      total_rows: data.total_rows ?? data.totalRows ?? data.rows?.length ?? 0,
      offset: data.offset ?? 0,
      limit: data.limit ?? 0,
      rows: data.rows ?? [],
      is_total_estimated: false // Default to exact count if provided by API
    };
    
    // If total_rows is still 0 but we have rows, we need to estimate it
    if (result.total_rows === 0 && result.rows.length > 0) {
      // Filter out TOTAL row (rank=0) and count actual query rows
      const queryRows = result.rows.filter(row => 
        row.fingerprint !== 'TOTAL' && row.dimension !== '' && (row.rank || 0) > 0
      );
      
      // IMPORTANT: This is an estimate based on current page data only
      // We cannot determine the actual total count across all pages from a single page response
      // This estimate assumes there might be more data beyond the current page
      if (queryRows.length === (result.limit || 10) && (result.offset || 0) === 0) {
        // If we got a full page and we're on the first page, estimate there might be more
        result.total_rows = Math.max(queryRows.length * 2, queryRows.length + 10);
      } else {
        // Otherwise, use the current page size as a conservative estimate
        result.total_rows = queryRows.length + (result.offset || 0);
      }
      
      result.is_total_estimated = true; // Mark as estimated since we calculated it
    } else if (data.total_rows === undefined && data.totalRows === undefined) {
      // If API didn't provide total_rows at all, mark our fallback as estimated
      result.is_total_estimated = true;
    }
    
    return result;
  } catch (error) {
    console.error('QAN API error:', error);
    throw error;
  }
};

export const getQANMetricsNames = async (): Promise<QANMetricsNamesResponse> => {
  try {
    const response = await api.post<QANMetricsNamesResponse>('/qan/metrics:getNames', {});
    return response.data;
  } catch (error) {
    console.error('QAN metrics names API error:', error);
    throw error;
  }
};

export const getQANFilters = async (request: QANFiltersRequest): Promise<QANFiltersResponse> => {
  try {
    const response = await api.post<QANFiltersResponse>('/qan/metrics:getFilters', request);
    return response.data;
  } catch (error) {
    console.error('QAN filters API error:', error);
    throw error;
  }
};

// Helper function to get recent QAN data
export const getRecentQANData = async (
  hoursBack: number = 12, 
  limit: number = 10, 
  filters?: QANLabel[],
  orderBy?: string,
  offset?: number
): Promise<QANReportResponse> => {
  const now = new Date();
  const startTime = new Date(now.getTime() - hoursBack * 60 * 60 * 1000);
  
  const request: QANReportRequest = {
    period_start_from: startTime.toISOString(),
    period_start_to: now.toISOString(),
    group_by: 'queryid',
    order_by: orderBy || '-load', // Default to load descending (slowest first)
    limit,
    offset: offset || 0,
    columns: ['query_time', 'lock_time', 'rows_sent', 'rows_examined', 'num_queries', 'docs_returned', 'docs_examined'],
    ...(filters && filters.length > 0 && { labels: filters })
  };
  
  return await getQANReport(request);
}; 