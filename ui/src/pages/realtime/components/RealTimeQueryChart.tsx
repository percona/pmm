import React, { useMemo } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Stack,
  LinearProgress,
} from '@mui/material';
import { RealTimeQueryData, QueryState } from 'types/realtime.types';
import { formatDuration, getQueryStateColor } from 'utils/realtimeUtils';

interface RealTimeQueryChartProps {
  queries: RealTimeQueryData[];
}

export const RealTimeQueryChart: React.FC<RealTimeQueryChartProps> = ({ queries }) => {
  const stateDistribution = useMemo(() => {
    const stateCounts = queries.reduce((acc, query) => {
      acc[query.state] = (acc[query.state] || 0) + 1;
      return acc;
    }, {} as Record<QueryState, number>);

    return Object.entries(stateCounts).map(([state, count]) => ({
      name: state,
      value: count,
      color: getQueryStateColor(state as QueryState),
    }));
  }, [queries]);

  const databaseDistribution = useMemo(() => {
    const dbCounts = queries.reduce((acc, query) => {
      acc[query.database] = (acc[query.database] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    return Object.entries(dbCounts)
      .map(([database, count]) => ({ database, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 10); // Top 10 databases
  }, [queries]);

  const operationDistribution = useMemo(() => {
    const opCounts = queries.reduce((acc, query) => {
      acc[query.mongodb?.operationType || 'unknown'] = (acc[query.mongodb?.operationType || 'unknown'] || 0) + 1;
      return acc;
    }, {} as Record<string, number>);

    return Object.entries(opCounts)
      .map(([operation, count]) => ({ operation, count }))
      .sort((a, b) => b.count - a.count)
      .slice(0, 10); // Top 10 operations
  }, [queries]);

  const durationStats = useMemo(() => {
    if (queries.length === 0) return { min: 0, max: 0, avg: 0, p95: 0 };

    const durations = queries.map(q => q.currentExecutionTime || 0).sort((a, b) => a - b);
    const min = durations[0] || 0;
    const max = durations[durations.length - 1] || 0;
    const avg = durations.reduce((sum, d) => sum + d, 0) / durations.length;
    const p95Index = Math.floor(durations.length * 0.95);
    const p95 = durations[p95Index] || 0;

    return { 
      min: min * 1000, 
      max: max * 1000, 
      avg: avg * 1000, 
      p95: p95 * 1000 
    };
  }, [queries]);

  if (queries.length === 0) {
    return (
      <Box textAlign="center" py={4}>
        <Typography variant="h6" color="text.secondary">
          No data available for charts
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Enable real-time analytics and run some queries to see visualizations
        </Typography>
      </Box>
    );
  }

  return (
    <Grid container spacing={3}>
      {/* State Distribution */}
      <Grid item xs={12} md={6}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Query State Distribution
            </Typography>
            <Stack spacing={2}>
              {stateDistribution.map(({ name, value, color }) => (
                <Box key={name}>
                  <Box display="flex" justifyContent="space-between" alignItems="center" mb={0.5}>
                    <Typography variant="body2" sx={{ color }}>
                      {name}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {value}
                    </Typography>
                  </Box>
                  <LinearProgress
                    variant="determinate"
                    value={(value / queries.length) * 100}
                    sx={{ 
                      height: 6, 
                      borderRadius: 3,
                      '& .MuiLinearProgress-bar': {
                        backgroundColor: color,
                      }
                    }}
                  />
                </Box>
              ))}
            </Stack>
          </CardContent>
        </Card>
      </Grid>

      {/* Database Distribution */}
      <Grid item xs={12} md={6}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Database Distribution
            </Typography>
            <Stack spacing={1}>
              {databaseDistribution.map(({ database, count }) => (
                <Box key={database}>
                  <Box display="flex" justifyContent="space-between" alignItems="center" mb={0.5}>
                    <Typography variant="body2" noWrap sx={{ maxWidth: '70%' }}>
                      {database}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {count}
                    </Typography>
                  </Box>
                  <LinearProgress
                    variant="determinate"
                    value={(count / databaseDistribution[0]?.count) * 100}
                    sx={{ height: 6, borderRadius: 3 }}
                  />
                </Box>
              ))}
            </Stack>
          </CardContent>
        </Card>
      </Grid>

      {/* Duration Statistics */}
      <Grid item xs={12} md={6}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Duration Statistics
            </Typography>
            <Stack spacing={2}>
              <Box>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  Average Duration
                </Typography>
                <Typography variant="h4" color="primary">
                  {formatDuration(durationStats.avg)}
                </Typography>
              </Box>
              <Box>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  P95 Duration
                </Typography>
                <Typography variant="h4" color="primary">
                  {formatDuration(durationStats.p95)}
                </Typography>
              </Box>
              <Box>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  Min / Max Duration
                </Typography>
                <Typography variant="h6">
                  {formatDuration(durationStats.min)} / {formatDuration(durationStats.max)}
                </Typography>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      </Grid>

      {/* Operation Distribution */}
      <Grid item xs={12} md={6}>
        <Card>
          <CardContent>
            <Typography variant="h6" gutterBottom>
              Top Operations
            </Typography>
            <Stack spacing={1}>
              {operationDistribution.map(({ operation, count }) => (
                <Box key={operation}>
                  <Box display="flex" justifyContent="space-between" alignItems="center" mb={0.5}>
                    <Typography variant="body2" noWrap sx={{ maxWidth: '70%' }}>
                      {operation}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {count}
                    </Typography>
                  </Box>
                  <LinearProgress
                    variant="determinate"
                    value={(count / operationDistribution[0]?.count) * 100}
                    sx={{ height: 6, borderRadius: 3 }}
                  />
                </Box>
              ))}
            </Stack>
          </CardContent>
        </Card>
      </Grid>
    </Grid>
  );
};