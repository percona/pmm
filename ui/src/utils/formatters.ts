import { QANRow } from '../api/qan';

// Number formatting
export const formatNumber = (num: number | undefined | null): string => {
  if (num === undefined || num === null || isNaN(num)) {
    return '0';
  }
  return num.toLocaleString();
};

// Duration formatting
export const formatDuration = (seconds: number | undefined | null): string => {
  if (seconds === undefined || seconds === null || isNaN(seconds)) {
    return '0ms';
  }
  if (seconds < 1) {
    return `${(seconds * 1000).toFixed(0)}ms`;
  }
  return `${seconds.toFixed(3)}s`;
};

// Query truncation
export const truncateQuery = (query: string | undefined | null, maxLength: number = 80): string => {
  if (!query) return 'N/A';
  return query.length > maxLength ? `${query.substring(0, maxLength)}...` : query;
};

// Clipboard utilities
export const copyToClipboard = async (text: string): Promise<void> => {
  try {
    await navigator.clipboard.writeText(text);
  } catch (err) {
    throw new Error('Failed to copy content');
  }
};

// QAN data extractors
export const getQueryCount = (row: QANRow): number => {
  // First try the direct field
  if (row.num_queries !== undefined && row.num_queries !== null) {
    return row.num_queries;
  }
  
  // Then try the metrics object
  if (row.metrics?.num_queries?.stats?.sum !== undefined) {
    return row.metrics.num_queries.stats.sum;
  }
  if (row.metrics?.num_queries?.stats?.cnt !== undefined) {
    return row.metrics.num_queries.stats.cnt;
  }
  
  return 0;
};

export const getLoadValue = (row: QANRow): number => {
  // First try the direct field
  if (row.load !== undefined && row.load !== null) {
    return row.load;
  }
  
  // Then try the metrics object for query_time
  if (row.metrics?.query_time?.stats?.sum !== undefined) {
    return row.metrics.query_time.stats.sum;
  }
  
  return 0;
};

export const getQueryRate = (row: QANRow): number => {
  // QPS can come from metrics or direct field
  const rateFromMetrics = row.metrics?.numQueries?.stats?.sumPerSec || row.metrics?.num_queries?.stats?.sumPerSec;
  if (rateFromMetrics !== undefined && rateFromMetrics !== null && !isNaN(rateFromMetrics)) {
    return rateFromMetrics;
  }
  
  return row.qps || 0;
}; 