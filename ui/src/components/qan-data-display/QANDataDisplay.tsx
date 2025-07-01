import React, { useState, useMemo, useEffect } from 'react';
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
  CircularProgress,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  OutlinedInput,
  SelectChangeEvent,
  TableSortLabel,
  Pagination,
  Switch,
} from '@mui/material';
import { QANReportResponse, QANRow } from '../../api/qan';
import { useQANFilters } from '../../hooks/api/useQAN';
import AnalyticsIcon from '@mui/icons-material/Analytics';
import RecommendIcon from '@mui/icons-material/Lightbulb';
import FilterListIcon from '@mui/icons-material/FilterList';
import AccessTimeIcon from '@mui/icons-material/AccessTime';
import RefreshIcon from '@mui/icons-material/Refresh';
import WarningIcon from '@mui/icons-material/Warning';
import { QueryAnalysisDialog } from './QueryAnalysisDialog';
import { generateDetailedQueryAnalysisPrompt } from '../../utils/queryAnalysisPrompts';
import { 
  formatNumber, 
  formatDuration, 
  truncateQuery, 
  getQueryCount, 
  getLoadValue, 
  getQueryRate
} from '../../utils/formatters';
import { 
  detectQueryAnomalies, 
  analyzeQANReport, 
  analyzeQANReportWithAI,
  AnomalyDetectionResult,
  AnomalySeverity 
} from '../../utils/queryAnomalyDetection';
import AnomalyWarningIcon from './AnomalyWarningIcon';

interface QANDataDisplayProps {
  data: QANReportResponse;
  selectedServices?: string[];
  onServiceFilterChange?: (services: string[]) => void;
  timeRangeHours?: number;
  onTimeRangeChange?: (hours: number) => void;
  onAnalyzeQuery?: (queryData: string) => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
  // Sorting and pagination props
  orderBy?: string;
  onSortChange?: (orderBy: string) => void;
  page?: number;
  pageSize?: number;
  onPageChange?: (page: number, pageSize: number) => void;
}

const QANDataDisplay: React.FC<QANDataDisplayProps> = ({ 
  data, 
  selectedServices = [],
  onServiceFilterChange,
  timeRangeHours,
  onTimeRangeChange,
  onAnalyzeQuery,
  onRefresh,
  isRefreshing = false,
  // Sorting and pagination props
  orderBy,
  onSortChange,
  page,
  pageSize,
  onPageChange,
}) => {

  
  // Service filter state
  const [selectedServicesState, setSelectedServicesState] = useState<string[]>(selectedServices);
  
  // Sync internal state with props
  useEffect(() => {
    setSelectedServicesState(selectedServices);
  }, [selectedServices]);

  // Analysis dialog state
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedQuery, setSelectedQuery] = useState<QANRow | null>(null);
  const [selectedQueryRank, setSelectedQueryRank] = useState<number>(0);

  // AI detection state
  const [useAIDetection, setUseAIDetection] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [aiAnalysisError, setAiAnalysisError] = useState<string | null>(null);

  // Anomaly detection results
  const [anomalyAnalysis, setAnomalyAnalysis] = useState(() => analyzeQANReport(data));
  
  // Update anomaly analysis when data changes
  useEffect(() => {
    const runAnalysis = async () => {
      if (useAIDetection) {
        setIsAnalyzing(true);
        setAiAnalysisError(null);
        try {
          const result = await analyzeQANReportWithAI(data);
          setAnomalyAnalysis(result);
        } catch (error) {
          setAiAnalysisError(error instanceof Error ? error.message : 'AI analysis failed');
          // Fallback to rule-based analysis
          setAnomalyAnalysis(analyzeQANReport(data));
        } finally {
          setIsAnalyzing(false);
        }
      } else {
        setAnomalyAnalysis(analyzeQANReport(data));
      }
    };
    
    runAnalysis();
  }, [data, useAIDetection]);

  // Store individual query anomaly results for efficient lookup
  const [queryAnomalies, setQueryAnomalies] = useState(new Map<string, AnomalyDetectionResult>());
  
  // Update individual query anomalies when needed (for rule-based analysis)
  useEffect(() => {
    if (!useAIDetection) {
      const results = new Map<string, AnomalyDetectionResult>();
      
      if (data.rows && data.rows.length > 0) {
        const queryRows = data.rows.filter(row => 
          row.fingerprint !== 'TOTAL' && row.dimension !== '' && (row.rank || 0) > 0
        );
        
        // Calculate average metrics for context
        const avgMetrics = queryRows.reduce((acc, query) => {
          const avgTime = query.metrics?.queryTime?.stats?.avg || query.metrics?.query_time?.stats?.avg || 0;
          const load = getLoadValue(query);
          const queryRate = getQueryRate(query);
          
          acc.avgTime += avgTime;
          acc.avgLoad += load;
          acc.avgQueryRate += queryRate;
          
          return acc;
        }, { avgTime: 0, avgLoad: 0, avgQueryRate: 0 });
        
        const count = queryRows.length;
        const normalizedAvgMetrics = {
          avgTime: avgMetrics.avgTime / count,
          avgLoad: avgMetrics.avgLoad / count,
          avgQueryRate: avgMetrics.avgQueryRate / count
        };
        
        queryRows.forEach(query => {
          const result = detectQueryAnomalies(query, {
            totalQueries: queryRows,
            avgMetrics: normalizedAvgMetrics,
            rank: query.rank || 0
          });
          results.set(query.dimension, result);
        });
      }
      
      setQueryAnomalies(results);
    } else if ('batchAnalysis' in anomalyAnalysis && anomalyAnalysis.batchAnalysis) {
      // For AI analysis, extract individual query results from batch analysis
      const results = new Map<string, AnomalyDetectionResult>();
      const batchAnalysis = (anomalyAnalysis as any).batchAnalysis;
      
      if (batchAnalysis && batchAnalysis.analyses) {
        Object.entries(batchAnalysis.analyses).forEach(([queryId, analysis]: [string, any]) => {
          // Convert AI analysis to our format
          const result: AnomalyDetectionResult = {
            queryId: queryId,
            hasAnomalies: analysis.hasAnomalies,
            anomalies: analysis.anomalies.map((a: any) => ({
              type: a.type as any,
              severity: a.severity as any,
              description: a.description,
              recommendation: a.recommendation,
              confidence: a.confidence,
              metrics: {
                riskLevel: a.riskLevel,
                estimatedFixTime: a.estimatedFixTime,
                impact: `${a.severity} risk`
              }
            })),
            overallSeverity: analysis.severity as any,
            aiAnalysisPrompt: `AI detected ${analysis.anomalies.length} issues for this query.`
          };
          results.set(queryId, result);
        });
      }
      
      setQueryAnomalies(results);
    } else {
      // Clear if no batch analysis available
      setQueryAnomalies(new Map());
    }
  }, [data, useAIDetection, anomalyAnalysis]);

  // Create filters request for the same time period as the data
  const filtersRequest = useMemo(() => {
    const now = new Date();
    const hours = timeRangeHours ?? 12;
    const startTime = new Date(now.getTime() - hours * 60 * 60 * 1000); // Use prop instead of hardcoded 24
    
    return {
      period_start_from: startTime.toISOString(),
      period_start_to: now.toISOString(),
      main_metric_name: 'load'
    };
  }, [timeRangeHours]);

  // Get available filters from the API
  const { data: filtersData, isLoading: filtersLoading } = useQANFilters(filtersRequest, {
    enabled: true, // Re-enable now that request format is fixed
    retry: 1
  });

  // Extract available services from filters
  const availableServices = useMemo(() => {
    if (!filtersData?.labels?.serviceName?.name) {
      // Fallback to extracting from database field if filters API not available
      const services = new Set<string>();
      data.rows.forEach(row => {
        if (row.database && row.fingerprint !== 'TOTAL' && row.dimension !== '') {
          services.add(row.database);
        }
      });
      const serviceArray = Array.from(services).sort();
      return serviceArray;
    }
    // Use service names from filters API
    const apiServices = filtersData.labels.serviceName.name
      .map(service => service.value)
      .filter(service => service && service.trim() !== '')
      .sort();
    return apiServices;
  }, [filtersData, data.rows, filtersLoading]);

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

  // Mapping between UI field names and backend API order_by values
  const getBackendOrderBy = (field: string, direction: 'asc' | 'desc'): string => {
    const fieldMap: Record<string, string> = {
      'rank': 'rank',
      'database': 'database', 
      'count': 'num_queries',
      'qps': 'num_queries_per_sec',
      'load': 'load',
      'avgTime': 'query_time',
      'maxTime': 'query_time_max',  // Use query_time for max time as well - backend will use max aggregation
      'rowsExamined': 'rows_examined'
    };
    
    const backendField = fieldMap[field] || 'load';
    return direction === 'desc' ? `-${backendField}` : backendField;
  };

  // Handle sorting - call parent to trigger new API request
  const handleRequestSort = (field: string) => {
    if (!onSortChange) {
      return;
    }
    
    // Determine current direction for this field
    const currentOrderBy = orderBy || '-load';
    
    // Check if this is the currently sorted field
    const backendFieldMap: Record<string, string> = {
      'rank': 'rank',
      'database': 'database',
      'count': 'num_queries',
      'qps': 'num_queries_per_sec', 
      'load': 'load',
      'avgTime': 'query_time',
      'maxTime': 'query_time_max',
      'rowsExamined': 'rows_examined'
    };
    
    const backendField = backendFieldMap[field] || field;
    const isCurrentField = currentOrderBy === backendField || currentOrderBy === `-${backendField}`;
    const isCurrentlyDesc = currentOrderBy.startsWith('-');
    
    // Toggle direction if same field, otherwise default to desc for most fields
    let newDirection: 'asc' | 'desc';
    if (isCurrentField) {
      // Toggle: if currently desc, make it asc; if currently asc, make it desc
      newDirection = isCurrentlyDesc ? 'asc' : 'desc';
    } else {
      // Default to desc for performance metrics, asc for text fields
      newDirection = ['database'].includes(field) ? 'asc' : 'desc';
    }
    
    const newOrderBy = getBackendOrderBy(field, newDirection);
    onSortChange(newOrderBy);
  };

  // Helper to determine if field is currently sorted
  const isFieldSorted = (field: string): boolean => {
    const currentOrderBy = orderBy || '-load';
    const backendFieldMap: Record<string, string> = {
      'rank': 'rank',
      'database': 'database',
      'count': 'num_queries',
      'qps': 'num_queries_per_sec',
      'load': 'load',
      'avgTime': 'query_time',
      'maxTime': 'query_time_max',
      'rowsExamined': 'rows_examined'
    };
    
    const backendField = backendFieldMap[field] || field;
    return currentOrderBy === backendField || currentOrderBy === `-${backendField}`;
  };

  // Helper to get sort direction for field
  const getSortDirection = (field: string): 'asc' | 'desc' => {
    const currentOrderBy = orderBy || '-load';
    if (isFieldSorted(field)) {
      return currentOrderBy.startsWith('-') ? 'desc' : 'asc';
    }
    return 'asc';
  };

  // Split data: separate TOTAL row from query rows
  const { totalRow, queryRows } = useMemo(() => {
    // Backend returns both TOTAL row and query rows
    // TOTAL row: rank=0, fingerprint='TOTAL' 
    // Query rows: rank>=1, actual queries
    const total = data.rows.find(row => row.fingerprint === 'TOTAL' || row.dimension === '' || row.rank === 0);
    const queries = data.rows.filter(row => row.fingerprint !== 'TOTAL' && row.dimension !== '' && row.rank > 0);
    
    return { totalRow: total, queryRows: queries };
  }, [data.rows, data.total_rows]);

  // Handle pagination - call parent to trigger new API request
  const handleChangePage = (_: unknown, newPage: number) => {
    if (!onPageChange || !pageSize) return;
    onPageChange(newPage, pageSize);
  };

  const handleChangeRowsPerPage = (event: SelectChangeEvent<number>) => {
    const newPageSize = parseInt(event.target.value as string, 10);
    if (!onPageChange) return;
    onPageChange(0, newPageSize); // Reset to first page with new page size
  };

  // Calculate max load for progress bars using query rows only
  const validLoads = queryRows.map(row => getLoadValue(row)).filter(load => !isNaN(load) && load > 0);
  const maxLoad = validLoads.length > 0 ? Math.max(...validLoads) : 1;



  const formatQueryForAnalysis = (row: QANRow, rank: number): string => {
    return generateDetailedQueryAnalysisPrompt({
      selectedQuery: row,
      rank,
    });
  };

  const handleAnalyzeInChat = (row: QANRow, rank: number) => {
    const queryData = formatQueryForAnalysis(row, rank);
    onAnalyzeQuery?.(queryData);
  };

  const handleServiceFilterChange = (event: SelectChangeEvent<typeof selectedServicesState>) => {
    const value = event.target.value;
    const newServices = typeof value === 'string' ? value.split(',') : value;
    setSelectedServicesState(newServices);
    
    // Call parent callback to trigger new API request
    if (onServiceFilterChange) {
      onServiceFilterChange(newServices);
    }
  };



  const handleAnalyzeInPopup = (row: QANRow, rank: number) => {
    setSelectedQuery(row);
    setSelectedQueryRank(rank);
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
    setSelectedQuery(null);
    setSelectedQueryRank(0);
  };

  // Handle AI analysis request from anomaly detection
  const handleAnomalyAnalysis = (result: AnomalyDetectionResult) => {
    const query = queryRows.find(q => q.dimension === result.queryId);
    if (query) {
      handleAnalyzeInPopup(query, query.rank);
    }
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
            <Box>
              <Typography variant="body2" color="textSecondary">
                Query Anomalies
              </Typography>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                {useAIDetection && isAnalyzing ? (
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <CircularProgress size={20} />
                    <Typography variant="body2" color="textSecondary">
                      AI analyzing...
                    </Typography>
                  </Box>
                ) : (
                  <>
                    <Typography variant="h5">
                      {anomalyAnalysis.anomalousQueries}/{anomalyAnalysis.totalQueries}
                    </Typography>
                    {anomalyAnalysis.criticalAnomalies > 0 && (
                      <Chip 
                        label={`${anomalyAnalysis.criticalAnomalies} Critical`}
                        color="error"
                        size="small"
                        variant="filled"
                      />
                    )}
                  </>
                )}
              </Box>
            </Box>
            
            {/* AI Health Score (if available) */}
            {useAIDetection && (
              <Box>
                <Typography variant="body2" color="textSecondary">
                  Health Score
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  {isAnalyzing ? (
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <CircularProgress size={20} />
                      <Typography variant="body2" color="textSecondary">
                        Computing...
                      </Typography>
                    </Box>
                  ) : (
                    'overallHealthScore' in anomalyAnalysis && (
                      <Typography variant="h5" color={
                        (anomalyAnalysis as any).overallHealthScore >= 80 ? 'success.main' :
                        (anomalyAnalysis as any).overallHealthScore >= 60 ? 'warning.main' : 'error.main'
                      }>
                        {(anomalyAnalysis as any).overallHealthScore}/100
                      </Typography>
                    )
                  )}
                </Box>
              </Box>
            )}
          </Box>
        </Paper>
      )}

      {/* Critical Anomalies Alert */}
      {anomalyAnalysis.criticalAnomalies > 0 && anomalyAnalysis.topAnomalies.length > 0 && (
        <Paper sx={{ mb: 3 }}>
          <Box sx={{ p: 2 }}>
            <Typography variant="h6" sx={{ mb: 2, color: 'error.main' }}>
              ⚠️ Critical Performance Anomalies Detected
            </Typography>
            {anomalyAnalysis.topAnomalies
              .filter(({ result }) => result.overallSeverity === AnomalySeverity.CRITICAL)
              .slice(0, 3)
              .map(({ query, result }) => (
                <AnomalyWarningIcon
                  key={query.dimension}
                  result={result}
                  onAnalyzeClick={handleAnomalyAnalysis}
                  variant="detailed"
                />
              ))}
          </Box>
        </Paper>
      )}

      {/* Top Queries Table */}
      <Paper>
        <Box sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">
              Top Queries ({data.total_rows || 0} total)
            </Typography>
            
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              {/* AI Detection Toggle */}
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Tooltip title={
                  isAnalyzing ? "AI analysis in progress..." :
                  useAIDetection ? "Using AI-powered anomaly detection" : "Using rule-based anomaly detection"
                }>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <Typography variant="body2" color="textSecondary">
                      AI Anomaly Detection
                    </Typography>
                    <Switch
                      checked={useAIDetection}
                      onChange={(e) => setUseAIDetection(e.target.checked)}
                      size="small"
                      color="primary"
                      disabled={isAnalyzing}
                    />
                    {isAnalyzing && (
                      <CircularProgress size={16} />
                    )}
                  </Box>
                </Tooltip>
                {aiAnalysisError && (
                  <Tooltip title={`AI Analysis Error: ${aiAnalysisError}`}>
                    <WarningIcon color="warning" fontSize="small" />
                  </Tooltip>
                )}
              </Box>

              {/* Refresh Button */}
              {onRefresh && (
                <Tooltip title="Refresh Data">
                  <IconButton
                    onClick={onRefresh}
                    disabled={isRefreshing}
                    color="primary"
                  >
                    {isRefreshing ? (
                      <CircularProgress size={24} />
                    ) : (
                      <RefreshIcon />
                    )}
                  </IconButton>
                </Tooltip>
              )}

              {/* Time Range Filter */}
              {onTimeRangeChange && (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <AccessTimeIcon color="action" />
                  <FormControl sx={{ minWidth: 120 }} size="small">
                    <InputLabel id="time-range-label">Time Range</InputLabel>
                    <Select
                      labelId="time-range-label"
                      value={timeRangeHours || 12}
                      onChange={(e) => onTimeRangeChange(Number(e.target.value))}
                      input={<OutlinedInput label="Time Range" />}
                    >
                      <MenuItem value={5 / 60}>5 minutes</MenuItem>
                      <MenuItem value={10 / 60}>10 minutes</MenuItem>
                      <MenuItem value={0.25}>15 minutes</MenuItem>
                      <MenuItem value={0.5}>30 minutes</MenuItem>
                      <MenuItem value={1}>1 hour</MenuItem>
                      <MenuItem value={3}>3 hours</MenuItem>
                      <MenuItem value={6}>6 hours</MenuItem>
                      <MenuItem value={12}>12 hours</MenuItem>
                      <MenuItem value={24}>24 hours</MenuItem>
                    </Select>
                  </FormControl>
                </Box>
              )}
              
              {/* Service Filter */}
              {availableServices.length > 1 && (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <FilterListIcon color="action" />
                  <FormControl sx={{ minWidth: 200 }} size="small">
                    <InputLabel id="service-filter-label">Filter by Service</InputLabel>
                    <Select
                      labelId="service-filter-label"
                      multiple
                      value={selectedServicesState}
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
          </Box>
          
          {/* Filter Summary */}
          {(selectedServicesState.length > 0 || timeRangeHours) && (
            <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
              {timeRangeHours && (
                <Typography variant="body2" color="textSecondary">
                  Time Range: {timeRangeHours < 1 ? `${timeRangeHours * 60} minutes` : `${timeRangeHours} hour${timeRangeHours !== 1 ? 's' : ''}`}
                </Typography>
              )}
              {selectedServicesState.length > 0 && (
                <>
                  <Typography variant="body2" color="textSecondary">
                    Services: {selectedServicesState.length} selected ({selectedServicesState.join(', ')})
                  </Typography>
                  <Button 
                    size="small" 
                    variant="outlined" 
                    onClick={() => {
                      setSelectedServicesState([]);
                      if (onServiceFilterChange) {
                        onServiceFilterChange([]);
                      }
                    }}
                    sx={{ textTransform: 'none' }}
                  >
                    Clear Service Filters
                  </Button>
                </>
              )}
            </Box>
          )}
        </Box>
        
        <TableContainer>
          {queryRows.length === 0 && selectedServicesState.length > 0 ? (
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
            <>
          <Table>
            <TableHead>
              <TableRow>
                    <TableCell>
                      <TableSortLabel
                        active={isFieldSorted('rank')}
                        direction={getSortDirection('rank')}
                        onClick={() => handleRequestSort('rank')}
                      >
                        Rank
                      </TableSortLabel>
                    </TableCell>
                <TableCell>Query</TableCell>
                    <TableCell>
                      <TableSortLabel
                        active={isFieldSorted('database')}
                        direction={getSortDirection('database')}
                        onClick={() => handleRequestSort('database')}
                      >
                        Database
                      </TableSortLabel>
                    </TableCell>
                    <TableCell align="right">
                      <TableSortLabel
                        active={isFieldSorted('count')}
                        direction={getSortDirection('count')}
                        onClick={() => handleRequestSort('count')}
                      >
                        Count
                      </TableSortLabel>
                    </TableCell>
                    <TableCell align="right">
                      <TableSortLabel
                        active={isFieldSorted('qps')}
                        direction={getSortDirection('qps')}
                        onClick={() => handleRequestSort('qps')}
                      >
                        QPS
                      </TableSortLabel>
                    </TableCell>
                    <TableCell align="right">
                      <TableSortLabel
                        active={isFieldSorted('load')}
                        direction={getSortDirection('load')}
                        onClick={() => handleRequestSort('load')}
                      >
                        Load
                      </TableSortLabel>
                    </TableCell>
                    <TableCell align="right">
                      <TableSortLabel
                        active={isFieldSorted('avgTime')}
                        direction={getSortDirection('avgTime')}
                        onClick={() => handleRequestSort('avgTime')}
                      >
                        Avg Time
                      </TableSortLabel>
                    </TableCell>
                    <TableCell align="right">
                      <TableSortLabel
                        active={isFieldSorted('maxTime')}
                        direction={getSortDirection('maxTime')}
                        onClick={() => handleRequestSort('maxTime')}
                      >
                        Max Time
                      </TableSortLabel>
                    </TableCell>
                    <TableCell align="right">
                      <TableSortLabel
                        active={isFieldSorted('rowsExamined')}
                        direction={getSortDirection('rowsExamined')}
                        onClick={() => handleRequestSort('rowsExamined')}
                      >
                        Rows/Docs Examined
                      </TableSortLabel>
                    </TableCell>
                <TableCell align="center">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
                  {queryRows.map((row) => {
                const avgTime = row.metrics?.queryTime?.stats?.avg || row.metrics?.query_time?.stats?.avg || 0;
                const maxTime = row.metrics?.queryTime?.stats?.max || row.metrics?.query_time?.stats?.max || 0;
                const rowsExamined = row.metrics?.rowsExamined?.stats?.sum || row.metrics?.rows_examined?.stats?.sum || 
                  row.metrics?.docsExamined?.stats?.sum || row.metrics?.docs_examined?.stats?.sum || 0;
                const loadPercentage = maxLoad > 0 ? ((getLoadValue(row) / maxLoad) * 100) : 0;
                const anomalyResult = queryAnomalies.get(row.dimension);

                return (
                  <TableRow key={row.dimension} hover>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip 
                            label={row.rank}
                          size="small" 
                            color={row.rank <= 3 ? 'error' : 'default'}
                        />
                        {/* Show loading indicator while AI analysis is running, otherwise show anomaly icons */}
                        {useAIDetection && isAnalyzing ? (
                          <Tooltip title="AI Anomaly Detection in progress...">
                            <CircularProgress size={16} />
                          </Tooltip>
                        ) : (
                          anomalyResult && anomalyResult.hasAnomalies && (
                            <AnomalyWarningIcon 
                              result={anomalyResult}
                              onAnalyzeClick={handleAnomalyAnalysis}
                              variant="icon"
                            />
                          )
                        )}
                      </Box>
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
                          <Tooltip title="Analyze with AI">
                            <IconButton
                              size="small"
                              color="primary"
                              onClick={() => handleAnalyzeInChat(row, row.rank)}
                            >
                              <AnalyticsIcon fontSize="small" />
                            </IconButton>
                          </Tooltip>
                        <Tooltip title="Get AI Recommendations">
                          <IconButton
                            size="small"
                            color="secondary"
                              onClick={() => handleAnalyzeInPopup(row, row.rank)}
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
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', p: 2 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Typography variant="body2" color="textSecondary">
                    Rows per page:
            </Typography>
                  <Select
                    value={pageSize || 10}
                    onChange={handleChangeRowsPerPage}
                    size="small"
                    sx={{ minWidth: 80 }}
                  >
                    <MenuItem value={5}>5</MenuItem>
                    <MenuItem value={10}>10</MenuItem>
                    <MenuItem value={25}>25</MenuItem>
                    <MenuItem value={50}>50</MenuItem>
                  </Select>
                  <Typography variant="body2" color="textSecondary">
                    {((page || 0) * (pageSize || 10) + 1)}-{Math.min((page || 0) * (pageSize || 10) + queryRows.length, data.total_rows || 0)} of {data.total_rows || 0}
                </Typography>
              </Box>
                <Pagination
                  count={Math.ceil((data.total_rows || 0) / (pageSize || 10))}
                  page={(page || 0) + 1}
                  onChange={(_, newPage) => handleChangePage(_, newPage - 1)}
                  color="primary"
                  shape="rounded"
                  showFirstButton
                  showLastButton
                  siblingCount={1}
                  boundaryCount={1}
                />
              </Box>

            </>
          )}
        </TableContainer>
            </Paper>

      {/* Query Analysis Dialog */}
      <QueryAnalysisDialog
        open={dialogOpen}
        onClose={handleCloseDialog}
        selectedQuery={selectedQuery}
        rank={selectedQueryRank}
      />
    </Box>
  );
};

export default QANDataDisplay; 