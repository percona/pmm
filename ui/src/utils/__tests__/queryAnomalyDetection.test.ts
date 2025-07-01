import { 
  detectQueryAnomalies, 
  analyzeQANReport, 
  AnomalyType, 
  AnomalySeverity 
} from '../queryAnomalyDetection';
import { QANRow, QANReportResponse } from '../../api/qan';

// Mock QAN data for testing
const createMockQANRow = (overrides: Partial<QANRow>): QANRow => ({
  rank: 1,
  dimension: 'test-dimension',
  database: 'test_db',
  fingerprint: 'SELECT * FROM users WHERE id = ?',
  num_queries: 1000,
  numQueries: 1000,
  qps: 10,
  load: 1.5,
  metrics: {
    query_time: {
      stats: {
        avg: 0.1,
        max: 0.5,
        sum: 100,
        min: 0.01,
        cnt: 1000,
        p99: 0.3
      }
    }
  },
  ...overrides
});

describe('Query Anomaly Detection', () => {
  describe('detectQueryAnomalies', () => {
    const mockContext = {
      totalQueries: [createMockQANRow({})],
      avgMetrics: {
        avgTime: 0.1,
        avgLoad: 1.0,
        avgQueryRate: 5.0
      },
      rank: 1
    };

    it('should detect high execution time anomalies', () => {
      const slowQuery = createMockQANRow({
        metrics: {
          query_time: {
            stats: {
              avg: 3.0, // 3 seconds - very slow
              max: 5.0,
              sum: 3000,
              min: 1.0,
              cnt: 1000,
              p99: 4.0
            }
          }
        }
      });

      const result = detectQueryAnomalies(slowQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies).toHaveLength(1);
      expect(result.anomalies[0].type).toBe(AnomalyType.HIGH_EXECUTION_TIME);
      expect(result.overallSeverity).toBe(AnomalySeverity.HIGH);
    });

    it('should detect excessive rows examined anomalies', () => {
      const inefficientQuery = createMockQANRow({
        metrics: {
          rows_examined: {
            stats: {
              avg: 100000, // Examining 100k rows
              sum: 100000000,
              cnt: 1000
            }
          },
          rows_sent: {
            stats: {
              avg: 10, // But only sending 10 rows
              sum: 10000,
              cnt: 1000
            }
          }
        }
      });

      const result = detectQueryAnomalies(inefficientQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies.some(a => a.type === AnomalyType.EXCESSIVE_ROWS_EXAMINED)).toBe(true);
    });

    it('should detect high lock time anomalies', () => {
      const lockQuery = createMockQANRow({
        metrics: {
          lock_time: {
            stats: {
              avg: 2.0, // 2 seconds lock time
              sum: 2000,
              cnt: 1000
            }
          }
        }
      });

      const result = detectQueryAnomalies(lockQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies.some(a => a.type === AnomalyType.HIGH_LOCK_TIME)).toBe(true);
    });

    it('should detect full table scan patterns', () => {
      const fullScanQuery = createMockQANRow({
        fingerprint: 'SELECT * FROM large_table ORDER BY created_at' // No WHERE clause
      });

      const result = detectQueryAnomalies(fullScanQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies.some(a => a.type === AnomalyType.FULL_TABLE_SCAN)).toBe(true);
    });

    it('should detect MongoDB collection scans', () => {
      const mongoQuery = createMockQANRow({
        fingerprint: 'db.users.find({})', // Empty filter - scans entire collection
        database: 'mongodb_db'
      });

      const result = detectQueryAnomalies(mongoQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies.some(a => a.type === AnomalyType.FULL_TABLE_SCAN)).toBe(true);
    });

    it('should detect high frequency slow queries', () => {
      const highFreqSlowQuery = createMockQANRow({
        qps: 50, // 50 queries per second
        metrics: {
          query_time: {
            stats: {
              avg: 1.0, // 1 second average - slow
              sum: 50000,
              cnt: 50000
            }
          }
        }
      });

      const result = detectQueryAnomalies(highFreqSlowQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies.some(a => a.type === AnomalyType.HIGH_FREQUENCY_SLOW)).toBe(true);
      expect(result.overallSeverity).toBe(AnomalySeverity.HIGH);
    });

    it('should detect resource intensive queries', () => {
      const resourceIntensiveQuery = createMockQANRow({
        load: 75.0 // Very high load
      });

      const result = detectQueryAnomalies(resourceIntensiveQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.anomalies.some(a => a.type === AnomalyType.RESOURCE_INTENSIVE)).toBe(true);
      expect(result.overallSeverity).toBe(AnomalySeverity.CRITICAL);
    });

    it('should not detect anomalies for well-performing queries', () => {
      const goodQuery = createMockQANRow({
        fingerprint: 'SELECT id, name FROM users WHERE status = ? LIMIT 10',
        metrics: {
          query_time: {
            stats: {
              avg: 0.005, // 5ms - fast
              max: 0.02,
              sum: 5,
              cnt: 1000
            }
          },
          rows_examined: {
            stats: {
              avg: 10,
              sum: 10000,
              cnt: 1000
            }
          },
          rows_sent: {
            stats: {
              avg: 8,
              sum: 8000,
              cnt: 1000
            }
          }
        },
        qps: 5, // Reasonable frequency
        load: 0.1 // Low load
      });

      const result = detectQueryAnomalies(goodQuery, mockContext);

      expect(result.hasAnomalies).toBe(false);
      expect(result.anomalies).toHaveLength(0);
    });

    it('should generate AI analysis prompts for anomalous queries', () => {
      const anomalousQuery = createMockQANRow({
        metrics: {
          query_time: {
            stats: {
              avg: 2.5 // Slow query
            }
          }
        }
      });

      const result = detectQueryAnomalies(anomalousQuery, mockContext);

      expect(result.hasAnomalies).toBe(true);
      expect(result.aiAnalysisPrompt).toBeDefined();
      expect(result.aiAnalysisPrompt).toContain('Query Anomaly Analysis Request');
      expect(result.aiAnalysisPrompt).toContain('Root cause analysis');
    });
  });

  describe('analyzeQANReport', () => {
    it('should analyze entire QAN report and return statistics', () => {
      const mockReport: QANReportResponse = {
        total_rows: 3,
        offset: 0,
        limit: 10,
        is_total_estimated: false,
        rows: [
          createMockQANRow({ 
            fingerprint: 'TOTAL',
            rank: 0,
            dimension: ''
          }),
          createMockQANRow({
            rank: 1,
            dimension: 'query1',
            fingerprint: 'SELECT * FROM large_table', // Problematic
            metrics: {
              query_time: {
                stats: { avg: 3.0 } // Slow
              }
            }
          }),
          createMockQANRow({
            rank: 2,
            dimension: 'query2',
            fingerprint: 'SELECT id FROM users WHERE status = ?', // Good
            metrics: {
              query_time: {
                stats: { avg: 0.01 } // Fast
              }
            }
          }),
          createMockQANRow({
            rank: 3,
            dimension: 'query3',
            fingerprint: 'SELECT * FROM orders', // Problematic
            load: 60.0 // High load - critical
          })
        ]
      };

      const analysis = analyzeQANReport(mockReport);

      expect(analysis.totalQueries).toBe(3); // Excludes TOTAL row
      expect(analysis.anomalousQueries).toBeGreaterThan(0);
      expect(analysis.criticalAnomalies).toBe(1); // High load query
      expect(analysis.topAnomalies.length).toBeGreaterThan(0);
    });

    it('should handle empty QAN reports', () => {
      const emptyReport: QANReportResponse = {
        total_rows: 0,
        offset: 0,
        limit: 10,
        is_total_estimated: false,
        rows: []
      };

      const analysis = analyzeQANReport(emptyReport);

      expect(analysis.totalQueries).toBe(0);
      expect(analysis.anomalousQueries).toBe(0);
      expect(analysis.criticalAnomalies).toBe(0);
      expect(analysis.topAnomalies).toHaveLength(0);
    });
  });
}); 