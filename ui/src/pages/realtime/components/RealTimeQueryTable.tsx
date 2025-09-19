import React, { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  IconButton,
  Collapse,
  Box,
  Typography,
  TableSortLabel,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Stack,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  Code as CodeIcon,
  Schedule as ScheduleIcon,
  Dataset as DatabaseIcon,
} from '@mui/icons-material';
import { RealTimeQueryData } from 'types/realtime.types';
import {
  formatDuration,
  formatTimestamp,
  getQueryStateColor,
  getQueryStateLabel,
  formatQueryText,
  getQueryComplexity,
} from 'utils/realtimeUtils';

interface RealTimeQueryTableProps {
  queries: RealTimeQueryData[];
}

type SortField = 'duration' | 'timestamp' | 'database';
type SortDirection = 'asc' | 'desc';

export const RealTimeQueryTable: React.FC<RealTimeQueryTableProps> = ({ queries }) => {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [queryDialogOpen, setQueryDialogOpen] = useState(false);
  const [selectedQuery, setSelectedQuery] = useState<RealTimeQueryData | null>(null);
  const [sortField, setSortField] = useState<SortField>('duration');
  const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

  const handleExpandRow = (queryId: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(queryId)) {
      newExpanded.delete(queryId);
    } else {
      newExpanded.add(queryId);
    }
    setExpandedRows(newExpanded);
  };

  const handleShowQuery = (query: RealTimeQueryData) => {
    setSelectedQuery(query);
    setQueryDialogOpen(true);
  };

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('desc');
    }
  };

  const sortedQueries = [...queries].sort((a, b) => {
    let aValue: any, bValue: any;
    
    switch (sortField) {
      case 'duration':
        aValue = a.currentExecutionTime || 0;
        bValue = b.currentExecutionTime || 0;
        break;
      case 'timestamp':
        aValue = new Date(a.timestamp || 0).getTime();
        bValue = new Date(b.timestamp || 0).getTime();
        break;
      case 'database':
        aValue = a.database || '';
        bValue = b.database || '';
        break;
      default:
        return 0;
    }

    if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1;
    if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1;
    return 0;
  });

  if (queries.length === 0) {
    return (
      <Box textAlign="center" py={4}>
        <Typography variant="h6" color="text.secondary">
          No queries found
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Try adjusting your filters or check if real-time analytics is enabled
        </Typography>
      </Box>
    );
  }

  return (
    <>
      <TableContainer component={Paper} sx={{ width: '100%', overflowX: 'auto' }}>
        <Table sx={{ minWidth: 1200, tableLayout: 'fixed' }}>
          <TableHead>
            <TableRow>
              <TableCell sx={{ width: '50px' }} />
              <TableCell sx={{ width: '120px' }}>Operation ID</TableCell>
              <TableCell sx={{ width: '300px' }}>Fingerprint</TableCell>
              <TableCell sx={{ width: '150px' }}>
                <TableSortLabel
                  active={sortField === 'database'}
                  direction={sortField === 'database' ? sortDirection : 'asc'}
                  onClick={() => handleSort('database')}
                >
                  Database
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ width: '120px' }}>
                <TableSortLabel
                  active={sortField === 'duration'}
                  direction={sortField === 'duration' ? sortDirection : 'asc'}
                  onClick={() => handleSort('duration')}
                >
                  Duration
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ width: '100px' }}>State</TableCell>
              <TableCell sx={{ width: '180px' }}>
                <TableSortLabel
                  active={sortField === 'timestamp'}
                  direction={sortField === 'timestamp' ? sortDirection : 'asc'}
                  onClick={() => handleSort('timestamp')}
                >
                  Timestamp
                </TableSortLabel>
              </TableCell>
              <TableCell sx={{ width: '80px' }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {sortedQueries.map((query) => {
              const isExpanded = expandedRows.has(query.queryId);
              const complexity = getQueryComplexity(query);
              
              return (
                <React.Fragment key={query.queryId}>
                  <TableRow hover>
                    <TableCell>
                      <IconButton
                        size="small"
                        onClick={() => handleExpandRow(query.queryId)}
                      >
                        {isExpanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                      </IconButton>
                    </TableCell>
                    <TableCell>
                      <Typography 
                        variant="body2" 
                        sx={{ 
                          fontFamily: 'monospace',
                          fontSize: '0.875rem',
                          fontWeight: 'medium'
                        }}
                      >
                        {query.mongodb?.opid || 'N/A'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Tooltip title={query.fingerprint || ''}>
                        <Typography 
                          variant="body2" 
                          noWrap 
                          sx={{ 
                            maxWidth: 300,
                            fontFamily: 'monospace',
                            fontSize: '0.75rem'
                          }}
                        >
                          {query.fingerprint || 'N/A'}
                        </Typography>
                      </Tooltip>
                    </TableCell>
                    <TableCell>
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <DatabaseIcon fontSize="small" color="action" />
                        <Typography variant="body2">{query.database || 'Unknown'}</Typography>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      <Stack direction="row" alignItems="center" spacing={1}>
                        <ScheduleIcon fontSize="small" color="action" />
                        <Typography variant="body2" fontWeight="medium">
                          {formatDuration((query.currentExecutionTime || 0) * 1000)}
                        </Typography>
                      </Stack>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={getQueryStateLabel(query.state)}
                        size="small"
                        sx={{
                          backgroundColor: getQueryStateColor(query.state),
                          color: 'white',
                        }}
                      />
                    </TableCell>
                    <TableCell>
                      <Typography variant="body2">
                        {formatTimestamp(query.timestamp || '')}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Tooltip title="View Query Details">
                        <IconButton
                          size="small"
                          onClick={() => handleShowQuery(query)}
                        >
                          <CodeIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                  <TableRow>
                    <TableCell colSpan={9} sx={{ py: 0 }}>
                      <Collapse in={isExpanded} timeout="auto" unmountOnExit>
                        <Box sx={{ p: 2, bgcolor: 'grey.50' }}>
                          <Typography variant="subtitle2" gutterBottom>
                            Query Details
                          </Typography>
                          <Stack spacing={1}>
                            {query.serviceName && (
                              <Box>
                                <Typography variant="caption" color="text.secondary">
                                  Service: {query.serviceName}
                                </Typography>
                              </Box>
                            )}
                            {query.mongodb?.opid && (
                              <Box>
                                <Typography variant="caption" color="text.secondary">
                                  MongoDB Operation ID: {query.mongodb.opid}
                                </Typography>
                              </Box>
                            )}
                {query.mongodb?.currentOpRaw && (
                    <Box>
                        <Typography variant="caption" color="text.secondary">
                            Raw currentOp: {query.mongodb.currentOpRaw}
                        </Typography>
                    </Box>
                )}
                            <Box>
                              <Typography variant="caption" color="text.secondary">
                                Complexity: {complexity}
                              </Typography>
                            </Box>
                            {query.labels && Object.keys(query.labels).length > 0 && (
                              <Box>
                                <Typography variant="caption" color="text.secondary">
                                  Labels: {Object.entries(query.labels)
                                    .map(([key, value]) => `${key}=${value}`)
                                    .join(', ')}
                                </Typography>
                              </Box>
                            )}
                          </Stack>
                        </Box>
                      </Collapse>
                    </TableCell>
                  </TableRow>
                </React.Fragment>
              );
            })}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Query Details Dialog */}
      <Dialog
        open={queryDialogOpen}
        onClose={() => setQueryDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Query Details</DialogTitle>
        <DialogContent>
          {selectedQuery && (
            <Stack spacing={2}>
              {selectedQuery.mongodb?.currentOpRaw ? (
                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Raw currentOp Document
                  </Typography>
                  <Box 
                    sx={{ 
                      backgroundColor: 'grey.100', 
                      p: 2, 
                      borderRadius: 1, 
                      maxHeight: 400, 
                      overflow: 'auto',
                      fontFamily: 'monospace',
                      fontSize: '0.75rem',
                      border: '1px solid',
                      borderColor: 'grey.300'
                    }}
                  >
                    <pre style={{ margin: 0, whiteSpace: 'pre-wrap' }}>
                      {(() => {
                        try {
                          return JSON.stringify(JSON.parse(selectedQuery.mongodb.currentOpRaw), null, 2);
                        } catch {
                          return selectedQuery.mongodb.currentOpRaw;
                        }
                      })()}
                    </pre>
                  </Box>
                </Box>
              ) : (
                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Query Text
                  </Typography>
                  <TextField
                    multiline
                    fullWidth
                    value={formatQueryText(selectedQuery.queryText)}
                    variant="outlined"
                    InputProps={{
                      readOnly: true,
                      sx: { fontFamily: 'monospace', fontSize: '0.875rem' }
                    }}
                    rows={10}
                  />
                </Box>
              )}
              <Box>
                <Typography variant="subtitle2" gutterBottom>
                  Metadata
                </Typography>
                <Stack spacing={1}>
                  {selectedQuery.mongodb?.opid && (
                    <Box display="flex" justifyContent="space-between">
                      <Typography variant="body2" color="text.secondary">
                        Operation ID:
                      </Typography>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                        {selectedQuery.mongodb.opid}
                      </Typography>
                    </Box>
                  )}
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Database:
                    </Typography>
                    <Typography variant="body2">{selectedQuery.database || 'Unknown'}</Typography>
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Operation:
                    </Typography>
                    <Typography variant="body2">{selectedQuery.mongodb?.operationType || 'Unknown'}</Typography>
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      Duration:
                    </Typography>
                    <Typography variant="body2">{formatDuration((selectedQuery.currentExecutionTime || 0) * 1000)}</Typography>
                  </Box>
                  <Box display="flex" justifyContent="space-between">
                    <Typography variant="body2" color="text.secondary">
                      State:
                    </Typography>
                    <Chip
                      label={getQueryStateLabel(selectedQuery.state)}
                      size="small"
                      sx={{
                        backgroundColor: getQueryStateColor(selectedQuery.state),
                        color: 'white',
                      }}
                    />
                  </Box>
                </Stack>
              </Box>
            </Stack>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setQueryDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </>
  );
};
