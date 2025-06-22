import React, { useState, useMemo } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Typography,
  Box,
  Chip,
  Tooltip,
  LinearProgress,
  Button,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  CircularProgress,
  Alert,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  OutlinedInput,
  SelectChangeEvent,
} from '@mui/material';
import { QANReportResponse, QANRow } from '../../api/qan';
import { useQANFilters } from '../../hooks/api/useQAN';
import { aiChatAPI, StreamMessage, ToolCall } from '../../api/aichat';
import AnalyticsIcon from '@mui/icons-material/Analytics';
import RecommendIcon from '@mui/icons-material/Lightbulb';
import FilterListIcon from '@mui/icons-material/FilterList';

interface QANDataDisplayProps {
  data: QANReportResponse;
  maxQueries?: number;
  onAnalyzeQuery?: (queryData: string) => void;
}

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

const QANDataDisplay: React.FC<QANDataDisplayProps> = ({ 
  data, 
  maxQueries = 10,
  onAnalyzeQuery
}) => {
  // Service filter state
  const [selectedServices, setSelectedServices] = useState<string[]>([]);

  // Create filters request for the same time period as the data
  const filtersRequest = useMemo(() => {
    const now = new Date();
    const startTime = new Date(now.getTime() - 24 * 60 * 60 * 1000); // 24 hours ago
    
    return {
      period_start_from: startTime.toISOString(),
      period_start_to: now.toISOString(),
      main_metric_name: 'load'
    };
  }, []);

  // Get available filters from the API
  const { data: filtersData, isLoading: filtersLoading } = useQANFilters(filtersRequest, {
    enabled: true,
    retry: 1
  });

  // Extract available services from filters
  const availableServices = useMemo(() => {
    if (!filtersData?.labels?.service_name?.name) {
      // Fallback to extracting from database field if filters API not available
      const services = new Set<string>();
      data.rows.forEach(row => {
        if (row.database && row.fingerprint !== 'TOTAL' && row.dimension !== '') {
          services.add(row.database);
        }
      });
      return Array.from(services).sort();
    }
    
    // Use service names from filters API
    return filtersData.labels.service_name.name
      .map(service => service.value)
      .filter(service => service && service.trim() !== '')
      .sort();
  }, [filtersData, data.rows]);

  if (!data.rows || data.rows.length === 0) {
    return (
      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          No QAN Data Available
        </Typography>
        <Typography variant="body2" color="textSecondary">
          No query data found for the selected time period.
        </Typography>
      </Paper>
    );
  }

  // Filter out the TOTAL row and apply service filter
  const queryRows = useMemo(() => {
    return data.rows
      .filter(row => {
        // Filter out total rows
        if (row.fingerprint === 'TOTAL' || row.dimension === '') return false;
        
        // Apply service filter
        if (selectedServices.length > 0) {
          // Try to match against service_name field first, then fallback to database
          const serviceToMatch = row.database || ''; // QAN data might not have service_name field yet
          if (!selectedServices.includes(serviceToMatch)) {
            return false;
          }
        }
        
        return true;
      })
      .slice(0, maxQueries);
  }, [data.rows, selectedServices, maxQueries]);

  // Total row can be identified by empty dimension or "TOTAL" fingerprint
  const totalRow = data.rows.find(row => row.fingerprint === 'TOTAL' || row.dimension === '');

  // Calculate max load for progress bars
  const validLoads = queryRows.map(row => getLoadValue(row)).filter(load => !isNaN(load) && load > 0);
  const maxLoad = validLoads.length > 0 ? Math.max(...validLoads) : 1;

  const formatDuration = (seconds: number | undefined | null): string => {
    if (seconds === undefined || seconds === null || isNaN(seconds)) {
      return '0ms';
    }
    if (seconds < 1) {
      return `${(seconds * 1000).toFixed(0)}ms`;
    }
    return `${seconds.toFixed(3)}s`;
  };

  const formatNumber = (num: number | undefined | null): string => {
    if (num === undefined || num === null || isNaN(num)) {
      return '0';
    }
    return num.toLocaleString();
  };

  const truncateQuery = (query: string | undefined | null, maxLength: number = 80): string => {
    if (!query) return 'N/A';
    if (query.length <= maxLength) return query;
    return query.substring(0, maxLength) + '...';
  };

  const formatQueryForAnalysis = (row: QANRow, rank: number): string => {
    const avgTime = row.metrics?.queryTime?.stats?.avg || row.metrics?.query_time?.stats?.avg || 0;
    const maxTime = row.metrics?.queryTime?.stats?.max || row.metrics?.query_time?.stats?.max || 0;
    const rowsExamined = row.metrics?.rowsExamined?.stats?.avg || row.metrics?.rows_examined?.stats?.avg || 0;
    const rowsSent = row.metrics?.rowsSent?.stats?.avg || 0;
    const lockTime = row.metrics?.lockTime?.stats?.avg || 0;

    return `**Query Performance Analysis Request**

**Query Rank:** #${rank} (by performance impact)

**Query Details:**
- **Database:** ${row.database || 'N/A'}
- **Query ID:** ${row.dimension}
- **SQL Query:** 
\`\`\`sql
${row.fingerprint || 'N/A'}
\`\`\`

**Performance Metrics:**
- **Execution Count:** ${formatNumber(getQueryCount(row))} times
- **Query Rate:** ${(getQueryRate(row)).toFixed(2)} queries/second
- **Load Impact:** ${formatDuration(getLoadValue(row))} seconds
- **Average Execution Time:** ${formatDuration(avgTime)}
- **Maximum Execution Time:** ${formatDuration(maxTime)}
- **Average Lock Time:** ${formatDuration(lockTime)}
- **Rows Examined (avg):** ${formatNumber(rowsExamined)}
- **Rows Sent (avg):** ${formatNumber(rowsSent)}

**Analysis Request:**
Please analyze this specific query and provide:

1. **Performance Assessment:** Is this query performing well or poorly?
2. **Optimization Opportunities:** What specific improvements can be made?
3. **Index Recommendations:** What indexes might help this query?
4. **Query Rewrite Suggestions:** Any alternative ways to write this query?
5. **Resource Usage Analysis:** Is the rows examined to rows sent ratio efficient?
6. **Priority Level:** How urgent is it to optimize this query?

Focus on actionable recommendations specific to this query's performance characteristics.`;
  };

  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [analysisResult, setAnalysisResult] = useState<string>('');
  const [selectedQuery, setSelectedQuery] = useState<QANRow | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pendingToolApproval, setPendingToolApproval] = useState<{
    requestId: string;
    toolCalls: ToolCall[];
  } | null>(null);
  const [analysisSessionId] = useState(() => `analysis_${Date.now()}`);

  const handleServiceFilterChange = (event: SelectChangeEvent<typeof selectedServices>) => {
    const value = event.target.value;
    setSelectedServices(typeof value === 'string' ? value.split(',') : value);
  };

  const handleAnalyzeInChat = (queryData: string) => {
    onAnalyzeQuery?.(queryData);
  };

  const handleAnalyzeInPopup = async (row: QANRow, rank: number) => {
    setSelectedQuery(row);
    setOpen(true);
    setLoading(true);
    setError(null);
    setAnalysisResult('');
    setPendingToolApproval(null);
    
    const analysisPrompt = `Please analyze this database query performance and provide optimization recommendations:

Query Details:
- Rank: #${rank} (by performance impact)
- Database: ${row.database}
- Query: ${row.fingerprint || 'N/A'}
- Execution Count: ${formatNumber(getQueryCount(row))} times
- Query Rate: ${getQueryRate(row).toFixed(2)} queries/second
- Load Impact: ${formatDuration(getLoadValue(row))}
- Average Execution Time: ${formatDuration(row.metrics?.queryTime?.stats?.avg || 0)}
- Maximum Execution Time: ${formatDuration(row.metrics?.queryTime?.stats?.max || 0)}
- Rows Examined: ${formatNumber(row.metrics?.rowsExamined?.stats?.avg || 0)} avg
- Rows Sent: ${formatNumber(row.metrics?.rowsSent?.stats?.avg || 0)} avg

Please provide:
1. Performance assessment and potential issues
2. Specific optimization recommendations
3. Index suggestions if applicable
4. Query rewrite suggestions if needed
5. Priority level for addressing this query
6. Use any available tools to gather additional context if needed

Focus on actionable recommendations that can improve query performance.`;

    try {
      const cleanup = aiChatAPI.streamChat(
        analysisSessionId,
        analysisPrompt,
        (message: StreamMessage) => {
          switch (message.type) {
            case 'message':
              if (message.content) {
                setAnalysisResult(prev => prev + message.content);
              }
              break;
            case 'tool_approval_request':
              if (message.tool_calls && message.request_id) {
                setPendingToolApproval({
                  requestId: message.request_id,
                  toolCalls: message.tool_calls
                });
              }
              break;
            case 'tool_execution':
              // Tool execution results will be included in the message stream
              break;
            case 'error':
              setError(message.error || 'An error occurred during analysis');
              setLoading(false);
              break;
            case 'done':
              setLoading(false);
              break;
          }
        },
        (error: string) => {
          setError(error);
          setLoading(false);
        },
        () => {
          setLoading(false);
        }
      );

      // Store cleanup function for potential cancellation
      return cleanup;
    } catch (error) {
      console.error('Error analyzing query:', error);
      setError('Failed to start analysis. Please try again.');
      setLoading(false);
    }
  };

  const handleToolApproval = async (approved: boolean) => {
    if (!pendingToolApproval) return;

    try {
      // Send approval/denial as a special message format
      const approvalMessage = approved 
        ? `[APPROVE_TOOLS:${pendingToolApproval.requestId}]`
        : `[DENY_TOOLS:${pendingToolApproval.requestId}]`;

      aiChatAPI.streamChat(
        analysisSessionId,
        approvalMessage,
        (message: StreamMessage) => {
          switch (message.type) {
            case 'message':
              if (message.content) {
                setAnalysisResult(prev => prev + message.content);
              }
              break;
            case 'tool_execution':
              // Tool execution results will be included in the message stream
              break;
            case 'error':
              setError(message.error || 'An error occurred during tool execution');
              break;
            case 'done':
              setLoading(false);
              break;
          }
        },
        (error: string) => {
          setError(error);
          setLoading(false);
        },
        () => {
          setLoading(false);
        }
      );

      setPendingToolApproval(null);
    } catch (error) {
      console.error('Error handling tool approval:', error);
      setError('Failed to process tool approval');
    }
  };

  const handleCloseDialog = () => {
    setOpen(false);
    setAnalysisResult('');
    setError(null);
    setPendingToolApproval(null);
    setSelectedQuery(null);
  };

  return (
    <Box>
      {/* Summary Section */}
      {totalRow && (
        <Paper sx={{ p: 3, mb: 3 }}>
          <Typography variant="h6" gutterBottom>
            Performance Summary
          </Typography>
          <Box sx={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            <Box>
              <Typography variant="body2" color="textSecondary">
                Total Queries
              </Typography>
              <Typography variant="h5">
                {formatNumber(getQueryCount(totalRow))}
              </Typography>
            </Box>
            <Box>
              <Typography variant="body2" color="textSecondary">
                Query Rate
              </Typography>
              <Typography variant="h5">
                {(getQueryRate(totalRow)).toFixed(2)} /sec
              </Typography>
            </Box>
            <Box>
              <Typography variant="body2" color="textSecondary">
                Total Load
              </Typography>
              <Typography variant="h5">
                {formatDuration(getLoadValue(totalRow))}
              </Typography>
            </Box>
            <Box>
              <Typography variant="body2" color="textSecondary">
                Database
              </Typography>
              <Typography variant="h5">
                {totalRow.database || 'Multiple'}
              </Typography>
            </Box>
          </Box>
        </Paper>
      )}

      {/* Top Queries Table */}
      <Paper>
        <Box sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">
              Top {queryRows.length} Queries by Load
            </Typography>
            
            {/* Service Filter */}
            {availableServices.length > 1 && (
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <FilterListIcon color="action" />
                <FormControl sx={{ minWidth: 200 }} size="small">
                  <InputLabel id="service-filter-label">Filter by Service</InputLabel>
                  <Select
                    labelId="service-filter-label"
                    multiple
                    value={selectedServices}
                    onChange={handleServiceFilterChange}
                    input={<OutlinedInput label="Filter by Service" />}
                    renderValue={(selected) => (
                      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                        {selected.map((value) => (
                          <Chip key={value} label={value} size="small" />
                        ))}
                      </Box>
                    )}
                  >
                    {availableServices.map((service) => (
                      <MenuItem key={service} value={service}>
                        {service}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                {filtersLoading && (
                  <CircularProgress size={16} />
                )}
              </Box>
            )}
          </Box>
          
          {/* Filter Summary */}
          {selectedServices.length > 0 && (
            <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 2 }}>
              <Typography variant="body2" color="textSecondary">
                Showing queries from {selectedServices.length} service{selectedServices.length !== 1 ? 's' : ''}: {selectedServices.join(', ')}
              </Typography>
              <Button 
                size="small" 
                variant="outlined" 
                onClick={() => setSelectedServices([])}
                sx={{ textTransform: 'none' }}
              >
                Clear Filters
              </Button>
            </Box>
          )}
        </Box>
        
        <TableContainer>
          {queryRows.length === 0 && selectedServices.length > 0 ? (
            <Box sx={{ p: 4, textAlign: 'center' }}>
              <FilterListIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
              <Typography variant="h6" color="textSecondary" gutterBottom>
                No Queries Found
              </Typography>
              <Typography variant="body2" color="textSecondary">
                No queries match the selected service filter. Try adjusting your filter criteria.
              </Typography>
            </Box>
          ) : (
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Rank</TableCell>
                <TableCell>Query</TableCell>
                <TableCell>Database</TableCell>
                <TableCell align="right">Count</TableCell>
                <TableCell align="right">QPS</TableCell>
                <TableCell align="right">Load</TableCell>
                <TableCell align="right">Avg Time</TableCell>
                <TableCell align="right">Max Time</TableCell>
                <TableCell align="right">Rows Examined</TableCell>
                <TableCell align="center">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {queryRows.map((row, index) => {
                const avgTime = row.metrics?.queryTime?.stats?.avg || row.metrics?.query_time?.stats?.avg || 0;
                const maxTime = row.metrics?.queryTime?.stats?.max || row.metrics?.query_time?.stats?.max || 0;
                const rowsExamined = row.metrics?.rowsExamined?.stats?.avg || row.metrics?.rows_examined?.stats?.avg || 0;
                const loadPercentage = maxLoad > 0 ? ((getLoadValue(row) / maxLoad) * 100) : 0;

                return (
                  <TableRow key={row.dimension} hover>
                    <TableCell>
                      <Chip 
                        label={index + 1} 
                        size="small" 
                        color={index < 3 ? 'error' : 'default'}
                      />
                    </TableCell>
                    <TableCell>
                      <Tooltip title={row.fingerprint || 'N/A'} placement="top">
                        <Box sx={{ maxWidth: 300 }}>
                          <Typography 
                            variant="body2" 
                            sx={{ 
                              fontFamily: 'monospace',
                              fontSize: '0.75rem',
                              wordBreak: 'break-word'
                            }}
                          >
                            {truncateQuery(row.fingerprint)}
                          </Typography>
                        </Box>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {row.database || 'N/A'}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {formatNumber(getQueryCount(row))}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {(getQueryRate(row)).toFixed(2)}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Box sx={{ minWidth: 100 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', mb: 0.5 }}>
                          <Typography variant="body2" sx={{ mr: 1 }}>
                            {formatDuration(getLoadValue(row))}
                          </Typography>
                        </Box>
                        <LinearProgress 
                          variant="determinate" 
                          value={loadPercentage} 
                          sx={{ height: 4 }}
                          color={loadPercentage > 80 ? 'error' : loadPercentage > 50 ? 'warning' : 'primary'}
                        />
                      </Box>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {avgTime > 0 ? formatDuration(avgTime) : '-'}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {maxTime > 0 ? formatDuration(maxTime) : '-'}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {rowsExamined > 0 ? formatNumber(rowsExamined) : '-'}
                      </Typography>
                    </TableCell>
                    <TableCell align="center">
                      <Box sx={{ display: 'flex', gap: 0.5 }}>
                        {onAnalyzeQuery && (
                          <Tooltip title="Analyze with AI Chat">
                            <IconButton
                              size="small"
                              color="primary"
                              onClick={() => {
                                const queryAnalysis = formatQueryForAnalysis(row, index + 1);
                                handleAnalyzeInChat(queryAnalysis);
                              }}
                            >
                              <AnalyticsIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        )}
                        <Tooltip title="Get AI Recommendations">
                          <IconButton
                            size="small"
                            color="secondary"
                            onClick={() => handleAnalyzeInPopup(row, index + 1)}
                          >
                            <RecommendIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      </Box>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          )}
        </TableContainer>
      </Paper>

      {/* Analysis Result Dialog */}
      <Dialog 
        open={open} 
        onClose={handleCloseDialog}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <RecommendIcon color="secondary" />
            <Typography variant="h6">
              AI Query Analysis & Recommendations
            </Typography>
          </Box>
        </DialogTitle>
        <DialogContent>
          {selectedQuery && (
            <Paper sx={{ p: 2, mb: 2, bgcolor: 'grey.50' }}>
              <Typography variant="subtitle2" gutterBottom>
                Query:
              </Typography>
              <Typography 
                variant="body2" 
                sx={{ 
                  fontFamily: 'monospace',
                  fontSize: '0.875rem',
                  wordBreak: 'break-word',
                  bgcolor: 'background.paper',
                  p: 1,
                  borderRadius: 1,
                  border: '1px solid',
                  borderColor: 'divider'
                }}
              >
                {selectedQuery.fingerprint || 'N/A'}
              </Typography>
            </Paper>
          )}

          {error && (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          )}

          {pendingToolApproval && (
            <Paper sx={{ p: 2, mb: 2, bgcolor: 'warning.light', color: 'warning.contrastText' }}>
              <Typography variant="subtitle2" gutterBottom>
                🔧 Tool Usage Request
              </Typography>
              <Typography variant="body2" sx={{ mb: 2 }}>
                The AI wants to use the following tools to provide better analysis:
              </Typography>
              <Box sx={{ mb: 2 }}>
                {pendingToolApproval.toolCalls.map((tool, index) => (
                  <Box key={index} sx={{ mb: 1 }}>
                    <Typography variant="body2" sx={{ fontWeight: 'bold' }}>
                      • {tool.function.name}
                    </Typography>
                    <Typography variant="caption" sx={{ ml: 2, fontFamily: 'monospace' }}>
                      {tool.function.arguments}
                    </Typography>
                  </Box>
                ))}
              </Box>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button 
                  size="small" 
                  variant="contained" 
                  color="success"
                  onClick={() => handleToolApproval(true)}
                >
                  Approve Tools
                </Button>
                <Button 
                  size="small" 
                  variant="outlined" 
                  onClick={() => handleToolApproval(false)}
                >
                  Deny Tools
                </Button>
              </Box>
            </Paper>
          )}

          {loading && !analysisResult && (
            <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 4 }}>
              <CircularProgress />
              <Typography variant="body2" sx={{ mt: 2 }}>
                {pendingToolApproval ? 'Waiting for tool approval...' : 'Analyzing query performance with AI...'}
              </Typography>
            </Box>
          )}

          {analysisResult && (
            <Box sx={{ mt: 1 }}>
              <Typography 
                variant="body1" 
                component="div"
                sx={{ 
                  whiteSpace: 'pre-wrap',
                  fontFamily: 'inherit',
                  lineHeight: 1.6
                }}
              >
                {analysisResult}
              </Typography>
              {loading && (
                <Box sx={{ display: 'flex', alignItems: 'center', mt: 2, color: 'text.secondary' }}>
                  <CircularProgress size={16} sx={{ mr: 1 }} />
                  <Typography variant="caption">
                    Analysis in progress...
                  </Typography>
                </Box>
              )}
            </Box>
          )}

          {!loading && !analysisResult && !error && !pendingToolApproval && (
            <Typography variant="body2" color="textSecondary">
              Click "Analyze with AI" to start the analysis.
            </Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog} color="primary" variant="contained">
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default QANDataDisplay; 