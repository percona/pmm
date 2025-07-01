import { QANRow, QANReportResponse } from '../api/qan';
import { getLoadValue, getQueryRate } from './formatters';
import { 
  analyzeQueryWithAI, 
  analyzeBatchWithAI, 
  convertAIAnalysisToUIFormat,
  AIAnomalyBatchAnalysis 
} from '../services/aiAnomalyDetection';

export enum AnomalyType {
  HIGH_EXECUTION_TIME = 'high_execution_time',
  EXCESSIVE_ROWS_EXAMINED = 'excessive_rows_examined',
  HIGH_LOCK_TIME = 'high_lock_time',
  MISSING_INDEX = 'missing_index',
  FULL_TABLE_SCAN = 'full_table_scan',
  CARTESIAN_JOIN = 'cartesian_join',
  TEMP_TABLE_DISK = 'temp_table_disk',
  HIGH_FREQUENCY_SLOW = 'high_frequency_slow',
  OUTLIER_PATTERN = 'outlier_pattern',
  RESOURCE_INTENSIVE = 'resource_intensive'
}

export enum AnomalySeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical'
}

export interface QueryAnomaly {
  type: AnomalyType;
  severity: AnomalySeverity;
  description: string;
  recommendation: string;
  confidence: number; // 0-1 score
  metrics?: {
    threshold?: number;
    actual?: number;
    impact?: string;
  };
}

export interface AnomalyDetectionResult {
  queryId: string;
  hasAnomalies: boolean;
  anomalies: QueryAnomaly[];
  overallSeverity: AnomalySeverity;
  aiAnalysisPrompt?: string;
}

/**
 * Detects anomalies in a single query based on its metrics and patterns
 */
export const detectQueryAnomalies = (
  query: QANRow,
  context: {
    totalQueries: QANRow[];
    avgMetrics: Record<string, number>;
    rank: number;
  }
): AnomalyDetectionResult => {
  const anomalies: QueryAnomaly[] = [];
  const fingerprint = query.fingerprint?.toLowerCase() || '';
  
  // Extract metrics safely
  const avgTime = query.metrics?.queryTime?.stats?.avg || query.metrics?.query_time?.stats?.avg || 0;
  const lockTime = query.metrics?.lockTime?.stats?.avg || query.metrics?.lock_time?.stats?.avg || 0;
  const rowsExamined = query.metrics?.rowsExamined?.stats?.avg || query.metrics?.rows_examined?.stats?.avg || 
        query.metrics?.docsExamined?.stats?.avg || query.metrics?.docs_examined?.stats?.avg || 0;
  const rowsSent = query.metrics?.rowsSent?.stats?.avg || query.metrics?.rows_sent?.stats?.avg || 
        query.metrics?.docsReturned?.stats?.avg || query.metrics?.docs_returned?.stats?.avg || 0;
  const queryRate = getQueryRate(query);
  const load = getLoadValue(query);

  // 1. High Execution Time Detection
  if (avgTime > 1.0) { // More than 1 second average
    anomalies.push({
      type: AnomalyType.HIGH_EXECUTION_TIME,
      severity: avgTime > 5.0 ? AnomalySeverity.CRITICAL : 
                avgTime > 2.0 ? AnomalySeverity.HIGH : AnomalySeverity.MEDIUM,
      description: `Query has high execution time: ${avgTime.toFixed(3)}s average`,
      recommendation: 'Consider query optimization, adding indexes, or reviewing query logic.',
      confidence: Math.min(avgTime / 5.0, 1.0),
      metrics: {
        threshold: 1.0,
        actual: avgTime,
        impact: 'Performance degradation'
      }
    });
  }

  // 2. Excessive Rows Examined vs Rows Sent (Efficiency Detection)
  if (rowsExamined > 0 && rowsSent > 0) {
    const efficiency = rowsSent / rowsExamined;
    if (efficiency < 0.01 && rowsExamined > 1000) { // Less than 1% efficiency with significant rows
      anomalies.push({
        type: AnomalyType.EXCESSIVE_ROWS_EXAMINED,
        severity: efficiency < 0.001 ? AnomalySeverity.HIGH : AnomalySeverity.MEDIUM,
        description: `Poor query efficiency: examining ${rowsExamined.toLocaleString()} rows but returning only ${rowsSent.toLocaleString()}`,
        recommendation: 'Add appropriate indexes or refine WHERE clauses to reduce rows examined.',
        confidence: Math.min((1 - efficiency) * 10, 1.0),
        metrics: {
          threshold: 0.01,
          actual: efficiency,
          impact: 'Resource waste and slow performance'
        }
      });
    }
  }

  // 3. High Lock Time Detection
  if (lockTime > 0.1) { // More than 100ms lock time
    anomalies.push({
      type: AnomalyType.HIGH_LOCK_TIME,
      severity: lockTime > 1.0 ? AnomalySeverity.HIGH : AnomalySeverity.MEDIUM,
      description: `High lock time detected: ${lockTime.toFixed(3)}s average`,
      recommendation: 'Review transaction isolation levels, query timing, or consider query optimization.',
      confidence: Math.min(lockTime / 2.0, 1.0),
      metrics: {
        threshold: 0.1,
        actual: lockTime,
        impact: 'Concurrency issues and blocking'
      }
    });
  }

  // 4. Pattern-based Detection (SQL/MongoDB)
  const patternAnomalies = detectPatternAnomalies(fingerprint);
  anomalies.push(...patternAnomalies);

  // 5. High Frequency + Slow Query Detection
  if (queryRate > 10 && avgTime > 0.5) { // More than 10 QPS and > 500ms avg time
    anomalies.push({
      type: AnomalyType.HIGH_FREQUENCY_SLOW,
      severity: AnomalySeverity.HIGH,
      description: `High-frequency slow query: ${queryRate.toFixed(1)} QPS with ${avgTime.toFixed(3)}s avg time`,
      recommendation: 'This query has high impact due to frequency. Prioritize optimization.',
      confidence: Math.min((queryRate * avgTime) / 50, 1.0),
      metrics: {
        threshold: 5.0, // QPS * avg_time threshold
        actual: queryRate * avgTime,
        impact: 'High load and resource consumption'
      }
    });
  }

  // 6. Outlier Detection (compared to other queries)
  if (context.rank <= 5 && context.totalQueries.length > 10) {
    const isOutlier = detectOutlierPattern(query, context.avgMetrics);
    if (isOutlier) {
      anomalies.push({
        type: AnomalyType.OUTLIER_PATTERN,
        severity: AnomalySeverity.MEDIUM,
        description: `Query metrics significantly deviate from typical patterns`,
        recommendation: 'Investigate why this query behaves differently from others.',
        confidence: 0.7,
        metrics: {
          impact: 'Potential optimization opportunity'
        }
      });
    }
  }

  // 7. Resource Intensive Detection
  if (load > 10.0) { // High load impact
    anomalies.push({
      type: AnomalyType.RESOURCE_INTENSIVE,
      severity: load > 50.0 ? AnomalySeverity.CRITICAL : AnomalySeverity.HIGH,
      description: `Resource intensive query with load impact of ${load.toFixed(2)}s`,
      recommendation: 'High priority for optimization due to resource consumption.',
      confidence: Math.min(load / 100, 1.0),
      metrics: {
        threshold: 10.0,
        actual: load,
        impact: 'System resource strain'
      }
    });
  }

  // Determine overall severity
  const overallSeverity = determineOverallSeverity(anomalies);
  
  // Generate AI analysis prompt if anomalies found
  const aiAnalysisPrompt = anomalies.length > 0 ? generateAIAnalysisPrompt(query, anomalies) : undefined;

  return {
    queryId: query.dimension || '',
    hasAnomalies: anomalies.length > 0,
    anomalies,
    overallSeverity,
    aiAnalysisPrompt
  };
};

/**
 * Detects pattern-based anomalies from query fingerprints
 */
const detectPatternAnomalies = (fingerprint: string): QueryAnomaly[] => {
  const anomalies: QueryAnomaly[] = [];
  
  // SQL Pattern Detection
  if (fingerprint.includes('select')) {
    // Missing WHERE clause in SELECT
    if (!fingerprint.includes('where') && !fingerprint.includes('limit')) {
      anomalies.push({
        type: AnomalyType.FULL_TABLE_SCAN,
        severity: AnomalySeverity.HIGH,
        description: 'SELECT query without WHERE clause may cause full table scan',
        recommendation: 'Add appropriate WHERE conditions or LIMIT clauses.',
        confidence: 0.8
      });
    }

    // Cartesian join detection
    if ((fingerprint.match(/join/g) || []).length > 1 && !fingerprint.includes('on')) {
      anomalies.push({
        type: AnomalyType.CARTESIAN_JOIN,
        severity: AnomalySeverity.CRITICAL,
        description: 'Potential cartesian join detected - multiple JOINs without proper ON conditions',
        recommendation: 'Review JOIN conditions to ensure proper relationships.',
        confidence: 0.9
      });
    }

    // Missing index indicators
    if (fingerprint.includes('order by') && !fingerprint.includes('limit')) {
      anomalies.push({
        type: AnomalyType.MISSING_INDEX,
        severity: AnomalySeverity.MEDIUM,
        description: 'ORDER BY without LIMIT may indicate missing index',
        recommendation: 'Consider adding index on ORDER BY columns or add LIMIT clause.',
        confidence: 0.6
      });
    }
  }

  // MongoDB Pattern Detection
  if (fingerprint.includes('db.') || fingerprint.includes('find(') || fingerprint.includes('aggregate(')) {
    // Full collection scan
    if (fingerprint.includes('find({})') || fingerprint.includes('find()')) {
      anomalies.push({
        type: AnomalyType.FULL_TABLE_SCAN,
        severity: AnomalySeverity.HIGH,
        description: 'MongoDB query scanning entire collection without filters',
        recommendation: 'Add appropriate query filters and ensure indexes exist.',
        confidence: 0.9
      });
    }

    // Complex aggregation without indexes
    if (fingerprint.includes('$lookup') || fingerprint.includes('$group')) {
      anomalies.push({
        type: AnomalyType.RESOURCE_INTENSIVE,
        severity: AnomalySeverity.MEDIUM,
        description: 'Complex aggregation pipeline may benefit from optimization',
        recommendation: 'Review aggregation stages and ensure appropriate indexes exist.',
        confidence: 0.7
      });
    }
  }

  return anomalies;
};

/**
 * Detects if a query is an outlier compared to average metrics
 */
const detectOutlierPattern = (query: QANRow, avgMetrics: Record<string, number>): boolean => {
  const avgTime = query.metrics?.queryTime?.stats?.avg || query.metrics?.query_time?.stats?.avg || 0;
  const load = getLoadValue(query);
  
  // Compare with average metrics (if available)
  const avgAvgTime = avgMetrics.avgTime || 0.1;
  const avgLoad = avgMetrics.avgLoad || 1.0;
  
  // Consider outlier if 5x worse than average
  return (avgTime > avgAvgTime * 5) || (load > avgLoad * 5);
};

/**
 * Determines overall severity from individual anomalies
 */
const determineOverallSeverity = (anomalies: QueryAnomaly[]): AnomalySeverity => {
  if (anomalies.length === 0) return AnomalySeverity.LOW;
  
  const severityScores = {
    [AnomalySeverity.LOW]: 1,
    [AnomalySeverity.MEDIUM]: 2,
    [AnomalySeverity.HIGH]: 3,
    [AnomalySeverity.CRITICAL]: 4
  };
  
  const maxScore = Math.max(...anomalies.map(a => severityScores[a.severity]));
  
  if (maxScore >= 4) return AnomalySeverity.CRITICAL;
  if (maxScore >= 3) return AnomalySeverity.HIGH;
  if (maxScore >= 2) return AnomalySeverity.MEDIUM;
  return AnomalySeverity.LOW;
};

/**
 * Generates AI analysis prompt for anomalous queries
 */
const generateAIAnalysisPrompt = (query: QANRow, anomalies: QueryAnomaly[]): string => {
  const anomalyDescriptions = anomalies.map(a => `- ${a.description}`).join('\n');
  
  return `**Query Anomaly Analysis Request**

**Anomalous Query Detected:**
Database: ${query.database}
Rank: #${query.rank}
Query: \`${query.fingerprint}\`

**Detected Anomalies:**
${anomalyDescriptions}

**Request:** Please analyze this query and provide:
1. Root cause analysis for the detected performance issues
2. Specific optimization recommendations
3. Index suggestions if applicable
4. Query rewriting suggestions
5. Risk assessment and priority level

Focus on actionable insights to resolve these performance anomalies.`;
};

/**
 * Analyzes all queries in a QAN report and returns anomaly statistics
 */
export const analyzeQANReport = (qanData: QANReportResponse): {
  totalQueries: number;
  anomalousQueries: number;
  criticalAnomalies: number;
  topAnomalies: Array<{ query: QANRow; result: AnomalyDetectionResult }>;
} => {
  if (!qanData.rows || qanData.rows.length === 0) {
    return {
      totalQueries: 0,
      anomalousQueries: 0,
      criticalAnomalies: 0,
      topAnomalies: []
    };
  }

  // Filter out TOTAL row and prepare query rows
  const queryRows = qanData.rows.filter(row => 
    row.fingerprint !== 'TOTAL' && row.dimension !== '' && (row.rank || 0) > 0
  );

  // Calculate average metrics for context
  const avgMetrics = calculateAverageMetrics(queryRows);
  
  // Analyze each query
  const results = queryRows.map(query => ({
    query,
    result: detectQueryAnomalies(query, {
      totalQueries: queryRows,
      avgMetrics,
      rank: query.rank || 0
    })
  }));

  // Filter anomalous queries
  const anomalousResults = results.filter(r => r.result.hasAnomalies);
  const criticalResults = anomalousResults.filter(r => 
    r.result.overallSeverity === AnomalySeverity.CRITICAL
  );

  // Sort by severity and confidence
  const topAnomalies = anomalousResults
    .sort((a, b) => {
      const severityOrder = {
        [AnomalySeverity.CRITICAL]: 4,
        [AnomalySeverity.HIGH]: 3,
        [AnomalySeverity.MEDIUM]: 2,
        [AnomalySeverity.LOW]: 1
      };
      
      const aSeverity = severityOrder[a.result.overallSeverity];
      const bSeverity = severityOrder[b.result.overallSeverity];
      
      if (aSeverity !== bSeverity) return bSeverity - aSeverity;
      
      // If same severity, sort by highest confidence
      const aMaxConfidence = Math.max(...a.result.anomalies.map(an => an.confidence));
      const bMaxConfidence = Math.max(...b.result.anomalies.map(an => an.confidence));
      return bMaxConfidence - aMaxConfidence;
    })
    .slice(0, 10); // Top 10 anomalies

  return {
    totalQueries: queryRows.length,
    anomalousQueries: anomalousResults.length,
    criticalAnomalies: criticalResults.length,
    topAnomalies
  };
};

/**
 * Calculates average metrics across all queries for context
 */
const calculateAverageMetrics = (queries: QANRow[]): Record<string, number> => {
  if (queries.length === 0) return {};
  
  const totals = queries.reduce((acc, query) => {
    const avgTime = query.metrics?.queryTime?.stats?.avg || query.metrics?.query_time?.stats?.avg || 0;
    const load = getLoadValue(query);
    const queryRate = getQueryRate(query);
    
    acc.avgTime += avgTime;
    acc.avgLoad += load;
    acc.avgQueryRate += queryRate;
    
    return acc;
  }, { avgTime: 0, avgLoad: 0, avgQueryRate: 0 });
  
  const count = queries.length;
  return {
    avgTime: totals.avgTime / count,
    avgLoad: totals.avgLoad / count,
    avgQueryRate: totals.avgQueryRate / count
  };
};

/**
 * AI-powered anomaly detection for a single query
 */
export const detectQueryAnomaliesWithAI = async (
  query: QANRow,
  context: {
    totalQueries: QANRow[];
    avgMetrics: Record<string, number>;
    rank: number;
  }
): Promise<AnomalyDetectionResult> => {
  try {
    // Call AI analysis
    const aiAnalysis = await analyzeQueryWithAI(query, {
      avgMetrics: context.avgMetrics,
      totalQueries: context.totalQueries.length
    });
    
    // Convert AI response to our UI format
    return convertAIAnalysisToUIFormat(aiAnalysis);
  } catch (error) {
    console.warn('AI anomaly detection failed, falling back to rule-based detection:', error);
    // Fallback to rule-based detection
    return detectQueryAnomalies(query, context);
  }
};

/**
 * AI-powered batch analysis for QAN reports
 */
export const analyzeQANReportWithAI = async (qanData: QANReportResponse): Promise<{
  totalQueries: number;
  anomalousQueries: number;
  criticalAnomalies: number;
  highAnomalies: number;
  mediumAnomalies: number;
  lowAnomalies: number;
  overallHealthScore: number;
  topAnomalies: Array<{ query: QANRow; result: AnomalyDetectionResult }>;
  batchAnalysis: AIAnomalyBatchAnalysis;
}> => {
  try {
    // Get AI batch analysis
    const batchAnalysis = await analyzeBatchWithAI(qanData);
    
    // Filter valid queries
    const queryRows = qanData.rows?.filter(row => 
      row.fingerprint !== 'TOTAL' && 
      row.dimension !== '' && 
      (row.rank || 0) > 0
    ) || [];
    
    // Process individual queries with AI analysis results
    const topAnomalies: Array<{ query: QANRow; result: AnomalyDetectionResult }> = [];
    
    for (const query of queryRows.slice(0, 20)) { // Analyze top 20 queries
      const queryAnalysis = batchAnalysis.analyses[query.dimension];
      
      if (queryAnalysis) {
        // Use AI analysis if available
        const result = convertAIAnalysisToUIFormat(queryAnalysis);
        if (result.hasAnomalies) {
          topAnomalies.push({ query, result });
        }
      } else {
        // Fallback to rule-based for queries not analyzed by AI
        const avgMetrics = calculateAverageMetrics(queryRows);
        const result = detectQueryAnomalies(query, {
          totalQueries: queryRows,
          avgMetrics,
          rank: query.rank || 0
        });
        
        if (result.hasAnomalies) {
          topAnomalies.push({ query, result });
        }
      }
    }
    
    // Sort by severity and confidence
    topAnomalies.sort((a, b) => {
      const severityWeight = { critical: 4, high: 3, medium: 2, low: 1 };
      const weightA = severityWeight[a.result.overallSeverity];
      const weightB = severityWeight[b.result.overallSeverity];
      
      if (weightA !== weightB) {
        return weightB - weightA; // Higher severity first
      }
      
      // If same severity, sort by confidence
      const confidenceA = a.result.anomalies.reduce((sum, a) => sum + a.confidence, 0) / a.result.anomalies.length;
      const confidenceB = b.result.anomalies.reduce((sum, a) => sum + a.confidence, 0) / b.result.anomalies.length;
      return confidenceB - confidenceA;
    });
    
    return {
      totalQueries: batchAnalysis.totalQueries,
      anomalousQueries: batchAnalysis.anomalousQueries,
      criticalAnomalies: batchAnalysis.criticalCount,
      highAnomalies: batchAnalysis.highCount,
      mediumAnomalies: batchAnalysis.mediumCount,
      lowAnomalies: batchAnalysis.lowCount,
      overallHealthScore: batchAnalysis.overallHealthScore,
      topAnomalies: topAnomalies.slice(0, 10), // Return top 10
      batchAnalysis
    };
  } catch (error) {
    console.warn('AI batch analysis failed, falling back to rule-based analysis:', error);
    // Fallback to rule-based analysis
    const fallbackResult = analyzeQANReport(qanData);
    return {
      ...fallbackResult,
      highAnomalies: 0,
      mediumAnomalies: 0, 
      lowAnomalies: 0,
      overallHealthScore: 50,
      batchAnalysis: {
        totalQueries: fallbackResult.totalQueries,
        anomalousQueries: fallbackResult.anomalousQueries,
        criticalCount: fallbackResult.criticalAnomalies,
        highCount: 0,
        mediumCount: 0,
        lowCount: 0,
        overallHealthScore: 50,
        topCriticalQueries: [],
        analysisTimestamp: new Date().toISOString(),
        analyses: {}
      }
    };
  }
};

/**
 * Hybrid approach: Use AI for initial analysis with rule-based fallback
 */
export const detectQueryAnomaliesHybrid = async (
  query: QANRow,
  context: {
    totalQueries: QANRow[];
    avgMetrics: Record<string, number>;
    rank: number;
  },
  useAI: boolean = true
): Promise<AnomalyDetectionResult> => {
  if (useAI) {
    try {
      return await detectQueryAnomaliesWithAI(query, context);
    } catch (error) {
      console.warn('Falling back to rule-based detection:', error);
    }
  }
  
  // Rule-based fallback
  return detectQueryAnomalies(query, context);
}; 