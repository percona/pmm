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
      rows: data.rows ?? []
    };
    
    // If total_rows is still 0 but we have rows, calculate it properly
    if (result.total_rows === 0 && result.rows.length > 0) {
      // Filter out TOTAL row (rank=0) and count actual query rows
      const queryRows = result.rows.filter(row => 
        row.fingerprint !== 'TOTAL' && row.dimension !== '' && (row.rank || 0) > 0
      );
      
      // For real API responses, the total_rows should represent the total count across all pages
      // Since we can't determine this from a single page, we'll use a reasonable estimate
      result.total_rows = Math.max(queryRows.length, result.limit || 10);
      
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

// Helper function to get recent QAN data for demo purposes
export const getRecentQANData = async (
  hoursBack: number = 24, 
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
    columns: ['query_time', 'lock_time', 'rows_sent', 'rows_examined', 'num_queries'],
    ...(filters && filters.length > 0 && { labels: filters })
  };
  

  
  try {
    return await getQANReport(request);
  } catch (error) {
    console.warn('QAN API not available, using mock data for demo:', error);
    
    // Return realistic mock data for development/demo purposes with multiple services
    const mockData = {
      total_rows: 7, // Count of actual query rows (excluding TOTAL row with rank=0)
      offset: offset || 0,
      limit: limit,
      rows: [
        {
          rank: 0,
          dimension: 'TOTAL',
          database: 'ecommerce',
          fingerprint: 'TOTAL',
          num_queries: 15847, // Deprecated field
          qps: 6.6,
          load: 2.45,
          metrics: {
            num_queries: {
              stats: {
                sum: 15847,
                rate: 6.6,
                cnt: 15847
              }
            },
            query_time: {
              stats: {
                sum: 2.45,
                avg: 0.000154,
                max: 0.025,
                min: 0.000001,
                p99: 0.008
              }
            },
            lock_time: {
              stats: {
                sum: 0.123,
                avg: 0.0000078,
                max: 0.002
              }
            },
            rows_examined: {
              stats: {
                sum: 2847392,
                avg: 179.7,
                max: 15000
              }
            },
            rows_sent: {
              stats: {
                sum: 18473,
                avg: 1.16,
                max: 100
              }
            }
          }
        },
        {
          rank: 1,
          dimension: 'A1B2C3D4E5F6',
          database: 'ecommerce',
          fingerprint: 'SELECT p.product_id, p.name, p.price, c.name AS category FROM products p JOIN categories c ON p.category_id = c.category_id WHERE p.price BETWEEN ? AND ? ORDER BY p.price DESC LIMIT ?',
          num_queries: 3247, // Deprecated field
          qps: 1.35,
          load: 0.89,
          metrics: {
            num_queries: {
              stats: {
                sum: 3247,
                rate: 1.35,
                cnt: 3247
              }
            },
            query_time: {
              stats: {
                sum: 0.89,
                avg: 0.000274,
                max: 0.025,
                min: 0.000012,
                p99: 0.008
              }
            },
            lock_time: {
              stats: {
                sum: 0.034,
                avg: 0.0000105,
                max: 0.002
              }
            },
            rows_examined: {
              stats: {
                sum: 487392,
                avg: 150.1,
                max: 15000
              }
            },
            rows_sent: {
              stats: {
                sum: 6494,
                avg: 2.0,
                max: 50
              }
            }
          }
        },
        {
          rank: 2,
          dimension: 'F6E5D4C3B2A1',
          database: 'analytics',
          fingerprint: 'SELECT u.user_id, u.email, u.first_name, u.last_name, COUNT(o.order_id) as order_count FROM users u LEFT JOIN orders o ON u.user_id = o.user_id WHERE u.created_at >= ? GROUP BY u.user_id ORDER BY order_count DESC',
          num_queries: 2156, // Deprecated field
          qps: 0.9,
          load: 0.67,
          metrics: {
            num_queries: {
              stats: {
                sum: 2156,
                rate: 0.9,
                cnt: 2156
              }
            },
            query_time: {
              stats: {
                sum: 0.67,
                avg: 0.000311,
                max: 0.018,
                min: 0.000008,
                p99: 0.006
              }
            },
            lock_time: {
              stats: {
                sum: 0.019,
                avg: 0.0000088,
                max: 0.001
              }
            },
            rows_examined: {
              stats: {
                sum: 324567,
                avg: 150.5,
                max: 8000
              }
            },
            rows_sent: {
              stats: {
                sum: 4312,
                avg: 2.0,
                max: 25
              }
            }
          }
        },
        {
          rank: 3,
          dimension: 'B2A1F6E5D4C3',
          database: 'inventory',
          fingerprint: 'UPDATE inventory SET quantity = quantity - ? WHERE product_id = ? AND quantity >= ?',
          num_queries: 1893, // Deprecated field
          qps: 0.79,
          load: 0.34,
          metrics: {
            num_queries: {
              stats: {
                sum: 1893,
                rate: 0.79,
                cnt: 1893
              }
            },
            query_time: {
              stats: {
                sum: 0.34,
                avg: 0.00018,
                max: 0.012,
                min: 0.000005,
                p99: 0.003
              }
            },
            lock_time: {
              stats: {
                sum: 0.045,
                avg: 0.0000238,
                max: 0.008
              }
            },
            rows_examined: {
              stats: {
                sum: 1893,
                avg: 1.0,
                max: 1
              }
            },
            rows_sent: {
              stats: {
                sum: 0,
                avg: 0,
                max: 0
              }
            }
          }
        },
        {
          rank: 4,
          dimension: 'C3D4E5F6A1B2',
          database: 'ecommerce',
          fingerprint: 'SELECT o.order_id, o.order_date, o.total_amount, u.email FROM orders o JOIN users u ON o.user_id = u.user_id WHERE o.order_date >= ? AND o.status = ? ORDER BY o.order_date DESC LIMIT ?',
          num_queries: 1247, // Deprecated field
          qps: 0.52,
          load: 0.28,
          metrics: {
            num_queries: {
              stats: {
                sum: 1247,
                rate: 0.52,
                cnt: 1247
              }
            },
            query_time: {
              stats: {
                sum: 0.28,
                avg: 0.000225,
                max: 0.015,
                min: 0.000003,
                p99: 0.004
              }
            },
            lock_time: {
              stats: {
                sum: 0.012,
                avg: 0.0000096,
                max: 0.001
              }
            },
            rows_examined: {
              stats: {
                sum: 74820,
                avg: 60.0,
                max: 1000
              }
            },
            rows_sent: {
              stats: {
                sum: 2494,
                avg: 2.0,
                max: 20
              }
            }
          }
        },
        {
          rank: 5,
          dimension: 'D4E5F6A1B2C3',
          database: 'analytics',
          fingerprint: 'SELECT DATE(created_at) as date, COUNT(*) as daily_events, COUNT(DISTINCT user_id) as unique_users FROM events WHERE created_at >= ? GROUP BY DATE(created_at) ORDER BY date DESC',
          num_queries: 892,
          qps: 0.37,
          load: 0.45,
          metrics: {
            num_queries: {
              stats: {
                sum: 892,
                rate: 0.37,
                cnt: 892
              }
            },
            query_time: {
              stats: {
                sum: 0.45,
                avg: 0.000504,
                max: 0.032,
                min: 0.000015,
                p99: 0.012
              }
            },
            lock_time: {
              stats: {
                sum: 0.023,
                avg: 0.0000258,
                max: 0.003
              }
            },
            rows_examined: {
              stats: {
                sum: 445600,
                avg: 499.6,
                max: 25000
              }
            },
            rows_sent: {
              stats: {
                sum: 892,
                avg: 1.0,
                max: 1
              }
            }
          }
        },
        {
          rank: 6,
          dimension: 'E5F6A1B2C3D4',
          database: 'inventory',
          fingerprint: 'SELECT w.name as warehouse, p.name as product, i.quantity, i.reserved_quantity FROM inventory i JOIN warehouses w ON i.warehouse_id = w.id JOIN products p ON i.product_id = p.id WHERE i.quantity < ?',
          num_queries: 634,
          qps: 0.26,
          load: 0.18,
          metrics: {
            num_queries: {
              stats: {
                sum: 634,
                rate: 0.26,
                cnt: 634
              }
            },
            query_time: {
              stats: {
                sum: 0.18,
                avg: 0.000284,
                max: 0.008,
                min: 0.000012,
                p99: 0.002
              }
            },
            lock_time: {
              stats: {
                sum: 0.008,
                avg: 0.0000126,
                max: 0.001
              }
            },
            rows_examined: {
              stats: {
                sum: 31700,
                avg: 50.0,
                max: 500
              }
            },
            rows_sent: {
              stats: {
                sum: 1268,
                avg: 2.0,
                max: 15
              }
            }
          }
        },
        {
          rank: 7,
          dimension: 'F6A1B2C3D4E5',
          database: 'logs',
          fingerprint: 'INSERT INTO access_logs (user_id, endpoint, method, status_code, response_time, created_at) VALUES (?, ?, ?, ?, ?, ?)',
          num_queries: 12456,
          qps: 5.19,
          load: 0.25,
          metrics: {
            num_queries: {
              stats: {
                sum: 12456,
                rate: 5.19,
                cnt: 12456
              }
            },
            query_time: {
              stats: {
                sum: 0.25,
                avg: 0.00002,
                max: 0.003,
                min: 0.000001,
                p99: 0.0005
              }
            },
            lock_time: {
              stats: {
                sum: 0.031,
                avg: 0.0000025,
                max: 0.0005
              }
            },
            rows_examined: {
              stats: {
                sum: 0,
                avg: 0,
                max: 0
              }
            },
            rows_sent: {
              stats: {
                sum: 0,
                avg: 0,
                max: 0
              }
            }
          }
        }
      ]
    };
    
    // Filter mock data based on provided filters
    if (filters && filters.length > 0) {
      const filteredRows = mockData.rows.filter(row => {
        return filters.every(filter => {
          if (filter.key === 'service_name') {
            // For service_name filter, check against database field in mock data
            return filter.value.includes(row.database);
          }
          // For other filters, you could add more logic here
          return true;
        });
      });
      
      // Calculate total_rows as count of query rows (excluding TOTAL row)
      const queryRowsCount = filteredRows.filter(row => 
        row.fingerprint !== 'TOTAL' && row.dimension !== '' && row.rank > 0
      ).length;
      
      const filteredMockData = {
        ...mockData,
        total_rows: queryRowsCount,
        rows: filteredRows
      };
      
      return filteredMockData;
    }
    return mockData;
  }
}; 