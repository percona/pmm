import { QANRow } from '../api/qan';
import { formatNumber, formatDuration, getQueryCount, getQueryRate, getLoadValue } from './formatters';

export interface QueryAnalysisPromptOptions {
  selectedQuery: QANRow;
  rank: number;
}

/**
 * Generates a more detailed analysis prompt for the main chat interface
 * Includes additional formatting and context for better presentation
 */
export const generateDetailedQueryAnalysisPrompt = ({
  selectedQuery,
  rank,
}: QueryAnalysisPromptOptions): string => {
  const avgTime = selectedQuery.metrics?.queryTime?.stats?.avg || selectedQuery.metrics?.query_time?.stats?.avg || 0;
  const maxTime = selectedQuery.metrics?.queryTime?.stats?.max || selectedQuery.metrics?.query_time?.stats?.max || 0;
  const rowsExamined = selectedQuery.metrics?.rowsExamined?.stats?.avg || selectedQuery.metrics?.rows_examined?.stats?.avg || 0;
  const rowsSent = selectedQuery.metrics?.rowsSent?.stats?.avg || 0;
  const lockTime = selectedQuery.metrics?.lockTime?.stats?.avg || 0;

  return `**Query Performance Analysis Request**

**Query Rank:** #${rank} (by performance impact)

**Query Details:**
- **Database:** ${selectedQuery.database || 'N/A'}
- **Query ID:** ${selectedQuery.dimension}
- **SQL Query:** 
\`\`\`sql
${selectedQuery.fingerprint || 'N/A'}
\`\`\`

**Performance Metrics:**
- **Execution Count:** ${formatNumber(getQueryCount(selectedQuery))} times
- **Query Rate:** ${(getQueryRate(selectedQuery)).toFixed(2)} queries/second
- **Load Impact:** ${formatDuration(getLoadValue(selectedQuery))} seconds
- **Average Execution Time:** ${formatDuration(avgTime)}
- **Maximum Execution Time:** ${formatDuration(maxTime)}
- **Average Lock Time:** ${formatDuration(lockTime)}
- **Rows Examined (avg):** ${formatNumber(rowsExamined)}
- **Rows Sent (avg):** ${formatNumber(rowsSent)}

**Analysis Request:**
Please analyze this specific query and provide:

**Performance Assessment:** Is this query performing well or poorly?
**Index Recommendations:** What indexes might help this query?
**Query Rewrite Suggestions:** Any alternative ways to write this query?
**Optimization Opportunities:** What specific improvements can be made?
**Priority Level:** How urgent is it to optimize this query?

**Use any available tools to gather additional context if needed**
**Important:** If any MCP tools fail or return errors, please continue with your analysis using the query data provided above. Don't let tool failures prevent you from providing valuable insights and recommendations based on the available performance metrics.

Try to be short and concise.
Focus on actionable recommendations specific to this query's performance characteristics.`;
}; 