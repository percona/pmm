import { QANReportResponse, QANRow } from 'api/qan';

// Helper function to get the correct query count from metrics
const getQueryCount = (row: QANRow): number => {
  // The API returns camelCase field names, so check both formats
  const metricsCount = row.metrics?.numQueries?.stats?.sum || row.metrics?.num_queries?.stats?.sum;
  if (metricsCount !== undefined && metricsCount !== null && !isNaN(metricsCount)) {
    return metricsCount;
  }
  
  // Fallback to the deprecated num_queries field if metrics not available
  return row.num_queries || 0;
};

// Helper function to get the correct load value
const getLoadValue = (row: QANRow): number => {
  // Load values are in the sparkline data, sum them up for total load
  if (row.sparkline && row.sparkline.length > 0) {
    const totalLoad = row.sparkline.reduce((sum, point) => {
      return sum + (point.load || 0);
    }, 0);
    return totalLoad;
  }
  
  // Fallback to metrics if sparkline not available
  const loadFromMetrics = row.metrics?.load?.stats?.sumPerSec;
  if (loadFromMetrics !== undefined && loadFromMetrics !== null && !isNaN(loadFromMetrics)) {
    return loadFromMetrics;
  }
  
  // Final fallback to direct load field
  return row.load || 0;
};

// Helper function to get query rate (QPS)
const getQueryRate = (row: QANRow): number => {
  // QPS can come from metrics or direct field
  const rateFromMetrics = row.metrics?.numQueries?.stats?.sumPerSec || row.metrics?.num_queries?.stats?.sumPerSec;
  if (rateFromMetrics !== undefined && rateFromMetrics !== null && !isNaN(rateFromMetrics)) {
    return rateFromMetrics;
  }
  
  return row.qps || 0;
};

export const formatQANDataForAI = (qanData: QANReportResponse): string => {
  if (!qanData.rows || qanData.rows.length === 0) {
    return `No QAN data available for the requested time period. This could mean:
- No queries were executed recently
- QAN collection is not enabled
- The database is not configured for query monitoring

Please check your PMM setup and ensure QAN is properly configured.`;
  }

  const totalRow = qanData.rows[0]; // First row is usually totals
  const queryRows = qanData.rows.slice(1); // Remaining rows are individual queries

  let report = `**Real QAN (Query Analytics) Data Analysis**\n\n`;

  // Summary section
  if (totalRow && (totalRow.fingerprint === 'TOTAL' || totalRow.dimension === '')) {
    const totalQueries = getQueryCount(totalRow);
    report += `**Performance Summary:**\n`;
    report += `- Total Queries Analyzed: ${totalQueries.toLocaleString()}\n`;
    report += `- Query Rate: ${getQueryRate(totalRow).toFixed(2)} queries/second\n`;
    report += `- Average Load: ${getLoadValue(totalRow).toFixed(3)} seconds\n`;
    report += `- Database: ${totalRow.database || 'Multiple'}\n`;
    report += `- Time Period: Recent data from PMM QAN\n\n`;
  }

  // Top slow queries
  if (queryRows.length > 0) {
    report += `**Top ${queryRows.length} Queries by Performance Impact:**\n\n`;
    
    queryRows.forEach((row, index) => {
      const queryCount = getQueryCount(row);
      report += `${index + 1}. **Query ID:** \`${row.dimension}\`\n`;
      report += `   **Database:** ${row.database}\n`;
      
      // Clean up fingerprint for display
      const fingerprint = row.fingerprint.length > 200 
        ? row.fingerprint.substring(0, 200) + '...' 
        : row.fingerprint;
      report += `   **Query:** \`${fingerprint}\`\n`;
      
      report += `   **Metrics:**\n`;
      report += `   - Execution Count: ${queryCount.toLocaleString()} times\n`;
      report += `   - Query Rate: ${getQueryRate(row).toFixed(2)} queries/second\n`;
      report += `   - Load Impact: ${getLoadValue(row).toFixed(3)} seconds\n`;
      
      // Add detailed metrics if available
      if (row.metrics) {
        if (row.metrics.queryTime?.stats || row.metrics.query_time?.stats) {
          const qt = row.metrics.queryTime?.stats || row.metrics.query_time?.stats;
          if (qt.avg) report += `   - Average Query Time: ${qt.avg.toFixed(3)}s\n`;
          if (qt.max) report += `   - Maximum Query Time: ${qt.max.toFixed(3)}s\n`;
          if (qt.sum) report += `   - Total Query Time: ${qt.sum.toFixed(3)}s\n`;
        }
        
        if (row.metrics.lockTime?.stats || row.metrics.lock_time?.stats) {
          const lt = row.metrics.lockTime?.stats || row.metrics.lock_time?.stats;
          if (lt.avg) report += `   - Average Lock Time: ${lt.avg.toFixed(3)}s\n`;
          if (lt.sum && lt.sum > 0) report += `   - Total Lock Time: ${lt.sum.toFixed(3)}s\n`;
        }
        
        if (row.metrics.rowsExamined?.stats || row.metrics.rows_examined?.stats) {
          const re = row.metrics.rowsExamined?.stats || row.metrics.rows_examined?.stats;
          if (re.avg) report += `   - Rows Examined: ${re.avg.toLocaleString()} avg\n`;
          if (re.sum) report += `   - Total Rows Examined: ${re.sum.toLocaleString()}\n`;
        }
        
        if (row.metrics.rowsSent?.stats || row.metrics.rows_sent?.stats) {
          const rs = row.metrics.rowsSent?.stats || row.metrics.rows_sent?.stats;
          if (rs.avg) report += `   - Rows Sent: ${rs.avg.toLocaleString()} avg\n`;
          if (rs.sum) report += `   - Total Rows Sent: ${rs.sum.toLocaleString()}\n`;
        }
      }
      
      report += `\n`;
    });
  }

  // Analysis request
  report += `**Analysis Request:**\n`;
  report += `Please analyze this real QAN data from our PMM instance and provide:\n\n`;
  report += `1. **Performance Assessment:** Overall database performance insights\n`;
  report += `2. **Query Optimization:** Specific recommendations for the slowest queries\n`;
  report += `3. **Index Recommendations:** Suggested indexes based on query patterns\n`;
  report += `4. **Resource Usage:** Analysis of lock times, row examination efficiency\n`;
  report += `5. **Prioritization:** Which issues to address first for maximum impact\n`;
  report += `6. **Monitoring:** Key metrics to watch going forward\n\n`;
  report += `Focus on actionable recommendations that can improve database performance based on this real-world data.`;

  return report;
};

export const formatQANError = (error: any): string => {
  return `**QAN Data Fetch Error**

Unable to retrieve QAN data from PMM. This could be due to:

- QAN collection not enabled or configured
- No database activity in the recent time period  
- Network connectivity issues
- PMM server not running or accessible

**Error Details:** ${error?.message || 'Unknown error'}

**Suggested Actions:**
1. Check if PMM Server is running and accessible
2. Verify QAN is enabled for your database services
3. Ensure there has been recent database activity
4. Check PMM configuration and logs

You can still ask questions about database performance optimization, and I'll provide general best practices and recommendations.`;
}; 