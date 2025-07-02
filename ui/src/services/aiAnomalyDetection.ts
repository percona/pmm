import { aiChatAPI } from '../api/aichat';
import { QANRow, QANReportResponse } from '../api/qan';
import { getQueryCount, getLoadValue, getQueryRate } from '../utils/formatters';
import { AnomalyDetectionResult } from '../utils/queryAnomalyDetection';

export interface AIAnomalyAnalysis {
  queryId: string;
  hasAnomalies: boolean;
  severity: 'low' | 'medium' | 'high' | 'critical';
  anomalies: AIDetectedAnomaly[];
  confidence: number; // 0-1
  recommendationSummary: string;
  estimatedImpact: 'minimal' | 'moderate' | 'significant' | 'severe';
}

export interface AIDetectedAnomaly {
  type: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  confidence: number;
  recommendation: string;
  estimatedFixTime: string; // e.g., "5 minutes", "1 hour"
  riskLevel: 'low' | 'medium' | 'high';
}

export interface AIAnomalyBatchAnalysis {
  totalQueries: number;
  anomalousQueries: number;
  criticalCount: number;
  highCount: number;
  mediumCount: number;
  lowCount: number;
  overallHealthScore: number; // 0-100
  topCriticalQueries: string[]; // Query IDs
  analysisTimestamp: string;
  analyses: Record<string, AIAnomalyAnalysis>;
}

/**
 * Generates an AI prompt for analyzing a single query for anomalies
 */
const generateQueryAnalysisPrompt = (query: QANRow, context: {
  avgMetrics: Record<string, number>;
  totalQueries: number;
}): string => {
  const avgTime = query.metrics?.queryTime?.stats?.avg || query.metrics?.query_time?.stats?.avg || 0;
  const maxTime = query.metrics?.queryTime?.stats?.max || query.metrics?.query_time?.stats?.max || 0;
  const lockTime = query.metrics?.lockTime?.stats?.avg || query.metrics?.lock_time?.stats?.avg || 0;
  const rowsExamined = query.metrics?.rowsExamined?.stats?.avg || query.metrics?.rows_examined?.stats?.avg || 
        query.metrics?.docsExamined?.stats?.avg || query.metrics?.docs_examined?.stats?.avg || 0;
  const rowsSent = query.metrics?.rowsSent?.stats?.avg || query.metrics?.rows_sent?.stats?.avg || 
        query.metrics?.docsReturned?.stats?.avg || query.metrics?.docs_returned?.stats?.avg || 0;

  return `**AI Query Anomaly Detection Request**

Please analyze this database query for performance anomalies and return a JSON response.

**Query Information:**
- Database: ${query.database || 'Unknown'}
- Rank: #${query.rank} out of ${context.totalQueries}
- Query: \`${query.fingerprint}\`

**Performance Metrics:**
- Execution Count: ${formatNumber(getQueryCount(query))} times
- Query Rate: ${getQueryRate(query).toFixed(2)} QPS
- Load Impact: ${getLoadValue(query).toFixed(3)} seconds
- Average Execution Time: ${avgTime.toFixed(3)}s
- Maximum Execution Time: ${maxTime.toFixed(3)}s
- Lock Time: ${lockTime.toFixed(3)}s
- Rows Examined: ${formatNumber(rowsExamined)}
- Rows Sent: ${formatNumber(rowsSent)}

**Context (Dataset Averages):**
- Average Query Time: ${(context.avgMetrics.avgTime || 0).toFixed(3)}s
- Average Load: ${(context.avgMetrics.avgLoad || 0).toFixed(3)}s
- Average QPS: ${(context.avgMetrics.avgQueryRate || 0).toFixed(2)}

**Required JSON Response Format:**
\`\`\`json
{
  "queryId": "${query.dimension}",
  "hasAnomalies": boolean,
  "severity": "low|medium|high|critical",
  "anomalies": [
    {
      "type": "descriptive_type_name",
      "description": "Brief description of the issue",
      "severity": "low|medium|high|critical",
      "confidence": 0.85,
      "recommendation": "Specific actionable recommendation",
      "estimatedFixTime": "15 minutes",
      "riskLevel": "low|medium|high"
    }
  ],
  "confidence": 0.90,
  "recommendationSummary": "Overall optimization strategy",
  "estimatedImpact": "minimal|moderate|significant|severe"
}
\`\`\`

**Analysis Guidelines:**
1. Focus on actionable performance issues
2. Consider query patterns (SQL/MongoDB syntax)
3. Compare metrics against dataset averages
4. Provide specific, implementable recommendations
5. Assign confidence scores based on evidence strength
6. Consider both immediate and systemic impacts

 Please analyze this query and respond ONLY with the JSON object, don't add any other text and don't include markdown formatting.`;
};

/**
 * Generates batch analysis prompt for multiple queries
 */
const generateBatchAnalysisPrompt = (qanData: QANReportResponse): string => {
  const queryRows = qanData.rows?.filter(row => 
    row.fingerprint !== 'TOTAL' && row.dimension !== '' && (row.rank || 0) > 0
  ) || [];

  // Get total row for context
  const totalRow = qanData.rows?.find(row => row.fingerprint === 'TOTAL');
  const totalQueries = totalRow ? getQueryCount(totalRow) : 0;
  const totalLoad = totalRow ? getLoadValue(totalRow) : 0;
  const totalQPS = totalRow ? getQueryRate(totalRow) : 0;

  // Get database distribution
  const databases = [...new Set(queryRows.map(q => q.database).filter(Boolean))];

  const topQueries = queryRows.slice(0, 15).map(query => {
    const avgTime = query.metrics?.queryTime?.stats?.avg || query.metrics?.query_time?.stats?.avg || 0;
    const maxTime = query.metrics?.queryTime?.stats?.max || query.metrics?.query_time?.stats?.max || 0;
    const lockTime = query.metrics?.lockTime?.stats?.avg || query.metrics?.lock_time?.stats?.avg || 0;
    const rowsExamined = query.metrics?.rowsExamined?.stats?.avg || query.metrics?.rows_examined?.stats?.avg || 
          query.metrics?.docsExamined?.stats?.avg || query.metrics?.docs_examined?.stats?.avg || 0;
    const rowsSent = query.metrics?.rowsSent?.stats?.avg || query.metrics?.rows_sent?.stats?.avg || 
          query.metrics?.docsReturned?.stats?.avg || query.metrics?.docs_returned?.stats?.avg || 0;
    const efficiency = rowsExamined > 0 ? (rowsSent / rowsExamined * 100) : 0;
    
    return `- Query ID: ${query.dimension}
  Database: ${query.database || 'Unknown'}
  Rank: ${query.rank}
  QPS: ${getQueryRate(query).toFixed(2)}
  Load: ${getLoadValue(query).toFixed(3)}s
  Avg Time: ${avgTime.toFixed(3)}s
  Max Time: ${maxTime.toFixed(3)}s
  Lock Time: ${lockTime.toFixed(3)}s
  Efficiency: ${efficiency.toFixed(2)}%
  Query: ${query.fingerprint?.substring(0, 150)}${query.fingerprint && query.fingerprint.length > 150 ? '...' : ''}`;
  }).join('\n\n');

  return `**AI Batch Query Anomaly Analysis Request**

Please analyze this QAN dataset for performance anomalies and provide an overall health assessment.

**Dataset Overview:**
- Total Unique Queries: ${queryRows.length}
- Total Query Executions: ${formatNumber(totalQueries)}
- Total Load Impact: ${totalLoad.toFixed(3)}s
- Overall QPS: ${totalQPS.toFixed(2)}
- Databases: ${databases.length > 0 ? databases.join(', ') : 'Multiple'}
- Time Period: Recent PMM QAN data
- Analysis Scope: Top ${Math.min(15, queryRows.length)} queries by performance impact

**Detailed Query Information:**
${topQueries}

**Required JSON Response Format:**
\`\`\`json
{
  "totalQueries": ${queryRows.length},
  "anomalousQueries": 0,
  "criticalCount": 0,
  "highCount": 0,
  "mediumCount": 0,
  "lowCount": 0,
  "overallHealthScore": 85,
  "topCriticalQueries": ["query_id_1", "query_id_2"],
  "analysisTimestamp": "${new Date().toISOString()}",
  "analyses": {
    "query_dimension_id": {
      "queryId": "query_dimension_id",
      "hasAnomalies": true,
      "severity": "high",
      "anomalies": [{
        "type": "high_execution_time",
        "description": "Query execution time exceeds normal patterns",
        "severity": "high",
        "confidence": 0.9,
        "recommendation": "Add appropriate indexes",
        "estimatedFixTime": "30 minutes",
        "riskLevel": "medium"
      }],
      "confidence": 0.85,
      "recommendationSummary": "Optimize slow queries with indexing",
      "estimatedImpact": "significant"
    }
  }
}
\`\`\`

**Analysis Guidelines:**
1. Analyze each query using the provided Query ID, metrics, and fingerprint
2. Identify the most problematic queries based on:
   - Execution time patterns (avg, max)
   - Query frequency and load impact
   - Resource efficiency (rows examined vs sent)
   - Lock time issues
   - Query complexity patterns
3. Provide an overall health score (0-100) where:
   - 90-100: Excellent performance
   - 80-89: Good performance with minor issues
   - 60-79: Fair performance with optimization opportunities
   - 40-59: Poor performance requiring attention
   - 0-39: Critical performance issues
4. Focus on actionable, specific recommendations
5. Consider both SQL and MongoDB query patterns
6. Prioritize by business impact (frequency × execution time × efficiency)
7. Include confidence scores based on evidence strength
8. Provide realistic estimated fix times

Please analyze this dataset and respond ONLY with the JSON object, don't add any other text and don't include markdown formatting.`;
};

/**
 * Analyzes a single query using AI and returns structured anomaly data
 */
export const analyzeQueryWithAI = async (
  query: QANRow,
  context: {
    avgMetrics: Record<string, number>;
    totalQueries: number;
  }
): Promise<AIAnomalyAnalysis> => {
  try {
    const prompt = generateQueryAnalysisPrompt(query, context);
    
    // Create a temporary session for this analysis
    const sessionId = `anomaly_${query.dimension}_${Date.now()}`;
    
    // Send analysis request to AI
    const response = await aiChatAPI.sendMessage({
      message: prompt,
      session_id: sessionId
    });
    
    // Extract JSON from the AI response
    const content = response.message?.content || '';
    
    // Try to extract JSON with markdown formatting first
    let jsonMatch = content.match(/```json\s*([\s\S]*?)\s*```/);
    let jsonString = '';
    
    if (jsonMatch) {
      jsonString = jsonMatch[1];
    } else {
      // If no markdown formatting, try to find JSON in the content
      const jsonStart = content.indexOf('{');
      const jsonEnd = content.lastIndexOf('}');
      
      if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
        jsonString = content.substring(jsonStart, jsonEnd + 1);
      } else {
        throw new Error('AI response does not contain valid JSON');
      }
    }
    
    const analysisResult: AIAnomalyAnalysis = JSON.parse(jsonString);
    
    // Validate and sanitize the response
    return {
      queryId: query.dimension,
      hasAnomalies: analysisResult.hasAnomalies || false,
      severity: analysisResult.severity || 'low',
      anomalies: analysisResult.anomalies || [],
      confidence: Math.max(0, Math.min(1, analysisResult.confidence || 0)),
      recommendationSummary: analysisResult.recommendationSummary || 'No specific recommendations',
      estimatedImpact: analysisResult.estimatedImpact || 'minimal'
    };
    
  } catch (error) {
    console.error('AI anomaly analysis failed for query:', query.dimension, error);
    
    // Fallback to basic analysis
    return {
      queryId: query.dimension,
      hasAnomalies: false,
      severity: 'low',
      anomalies: [],
      confidence: 0,
      recommendationSummary: 'Analysis unavailable - AI service error',
      estimatedImpact: 'minimal'
    };
  }
};

/**
 * Analyzes multiple queries in batch using AI
 */
export const analyzeBatchWithAI = async (
  qanData: QANReportResponse
): Promise<AIAnomalyBatchAnalysis> => {
  try {
    const prompt = generateBatchAnalysisPrompt(qanData);
    
    // Create a temporary session for batch analysis
    const sessionId = `batch_anomaly_${Date.now()}`;
    
    // Send batch analysis request to AI
    const response = await aiChatAPI.sendMessage({
      message: prompt,
      session_id: sessionId
    });
    
    // Extract JSON from the AI response
    const content = response.message?.content || '';
    
    // Try to extract JSON with markdown formatting first
    let jsonMatch = content.match(/```json\s*([\s\S]*?)\s*```/);
    let jsonString = '';
    
    if (jsonMatch) {
      jsonString = jsonMatch[1];
    } else {
      // If no markdown formatting, try to find JSON in the content
      const jsonStart = content.indexOf('{');
      const jsonEnd = content.lastIndexOf('}');
      
      if (jsonStart !== -1 && jsonEnd !== -1 && jsonEnd > jsonStart) {
        jsonString = content.substring(jsonStart, jsonEnd + 1);
      } else {
        throw new Error('AI response does not contain valid JSON');
      }
    }
    
    const batchResult: AIAnomalyBatchAnalysis = JSON.parse(jsonString);
    
    // Validate and sanitize the response
    return {
      totalQueries: batchResult.totalQueries || 0,
      anomalousQueries: batchResult.anomalousQueries || 0,
      criticalCount: batchResult.criticalCount || 0,
      highCount: batchResult.highCount || 0,
      mediumCount: batchResult.mediumCount || 0,
      lowCount: batchResult.lowCount || 0,
      overallHealthScore: Math.max(0, Math.min(100, batchResult.overallHealthScore || 50)),
      topCriticalQueries: batchResult.topCriticalQueries || [],
      analysisTimestamp: batchResult.analysisTimestamp || new Date().toISOString(),
      analyses: batchResult.analyses || {}
    };
    
  } catch (error) {
    console.error('AI batch anomaly analysis failed:', error);
    
    // Fallback response
    const queryRows = qanData.rows?.filter(row => 
      row.fingerprint !== 'TOTAL' && row.dimension !== '' && (row.rank || 0) > 0
    ) || [];
    
    return {
      totalQueries: queryRows.length,
      anomalousQueries: 0,
      criticalCount: 0,
      highCount: 0,
      mediumCount: 0,
      lowCount: 0,
      overallHealthScore: 50,
      topCriticalQueries: [],
      analysisTimestamp: new Date().toISOString(),
      analyses: {}
    };
  }
};

/**
 * Utility function to format numbers for AI prompts
 */
const formatNumber = (num: number): string => {
  if (num >= 1000000) {
    return `${(num / 1000000).toFixed(1)}M`;
  } else if (num >= 1000) {
    return `${(num / 1000).toFixed(1)}K`;
  }
  return num.toString();
};

/**
 * Converts AI anomaly analysis to the format expected by our UI components
 */
export const convertAIAnalysisToUIFormat = (aiAnalysis: AIAnomalyAnalysis): AnomalyDetectionResult => {
  // Map AI anomaly types to our enum values with fallback
  const mapAnomalyType = (type: string): string => {
    const typeMap: Record<string, string> = {
      'high_execution_time': 'high_execution_time',
      'excessive_rows_examined': 'excessive_rows_examined', 
      'high_lock_time': 'high_lock_time',
      'missing_index': 'missing_index',
      'full_table_scan': 'full_table_scan',
      'cartesian_join': 'cartesian_join',
      'temp_table_disk': 'temp_table_disk',
      'high_frequency_slow': 'high_frequency_slow',
      'outlier_pattern': 'outlier_pattern',
      'resource_intensive': 'resource_intensive'
    };
    return typeMap[type] || 'resource_intensive'; // fallback to resource_intensive
  };

  return {
    queryId: aiAnalysis.queryId,
    hasAnomalies: aiAnalysis.hasAnomalies,
    anomalies: aiAnalysis.anomalies.map(anomaly => ({
      type: mapAnomalyType(anomaly.type) as any, // Cast to match enum
      severity: anomaly.severity as any, // Cast to match enum
      description: anomaly.description,
      recommendation: anomaly.recommendation,
      confidence: anomaly.confidence,
      metrics: {
        riskLevel: anomaly.riskLevel,
        estimatedFixTime: anomaly.estimatedFixTime,
        impact: `${anomaly.severity} risk level`
      }
    })),
    overallSeverity: aiAnalysis.severity as any, // Cast to match enum
    aiAnalysisPrompt: `**AI-Generated Analysis Summary**

**Query:** ${aiAnalysis.queryId}
**Overall Severity:** ${aiAnalysis.severity.toUpperCase()}
**Confidence:** ${(aiAnalysis.confidence * 100).toFixed(0)}%

**Detected Issues:**
${aiAnalysis.anomalies.map(a => `- ${a.description} (${a.severity})`).join('\n')}

**Recommendations:**
${aiAnalysis.recommendationSummary}

**Estimated Impact:** ${aiAnalysis.estimatedImpact}

Would you like me to provide more detailed optimization guidance for this query?`
  };
}; 