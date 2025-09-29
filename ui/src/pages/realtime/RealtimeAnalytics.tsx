import React, { useState, useMemo } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Chip,
  Stack,
  Alert,
  CircularProgress,
  Tabs,
  Tab,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  TextField,
  InputAdornment,
  IconButton,
  Tooltip,
  SelectChangeEvent,
} from '@mui/material';
import {
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';
import { Page } from 'components/page/Page';
import { useRealTimeData, useRealTimeServices } from 'hooks/api/useRealtime';
import { RealTimeQueryTable } from './components/RealTimeQueryTable';
import { RealTimeQueryChart } from './components/RealTimeQueryChart';
import { RealTimeServiceCard } from './components/RealTimeServiceCard';
import { RealTimeConfigDialog } from './components/RealTimeConfigDialog';
import { QueryState } from 'types/realtime.types';
import { searchQueries, filterQueriesByState, sortQueriesByDuration, deduplicateQueriesByOpId } from 'utils/realtimeUtils';
import { Messages } from './RealtimeAnalytics.messages';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index, ...other }) => (
  <div
    role="tabpanel"
    hidden={value !== index}
    id={`realtime-tabpanel-${index}`}
    aria-labelledby={`realtime-tab-${index}`}
    {...other}
  >
    {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
  </div>
);

export const RealtimeAnalytics: React.FC = () => {
  const [selectedServiceId, setSelectedServiceId] = useState<string>('');
  const [tabValue, setTabValue] = useState(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [stateFilter, setStateFilter] = useState<QueryState | 'all'>('all');
  const [configDialogOpen, setConfigDialogOpen] = useState(false);

  const { data: servicesData, isLoading: servicesLoading, error: servicesError } = useRealTimeServices();
  const { data: queriesData, isLoading: queriesLoading, error: queriesError, refetch } = useRealTimeData(
    selectedServiceId || undefined
  );

  const filteredQueries = useMemo(() => {
    if (!queriesData?.queries) return [];
    
    let filtered = queriesData.queries;
    
    // First, deduplicate queries by MongoDB opid to prevent showing the same operation multiple times
    filtered = deduplicateQueriesByOpId(filtered);
    
    // Apply search filter
    filtered = searchQueries(filtered, searchTerm);
    
    // Apply state filter
    if (stateFilter !== 'all') {
      filtered = filterQueriesByState(filtered, stateFilter as QueryState);
    }
    
    // Sort by duration (longest first)
    filtered = sortQueriesByDuration(filtered);
    
    return filtered;
  }, [queriesData?.queries, searchTerm, stateFilter]);

  const runningQueries = useMemo(() => {
    if (!queriesData?.queries) return [];
    
    // Deduplicate and then filter for running queries only
    const deduplicated = deduplicateQueriesByOpId(queriesData.queries);
    return deduplicated.filter(query => query.state === QueryState.RUNNING);
  }, [queriesData?.queries]);

  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const handleServiceChange = (event: SelectChangeEvent<string>) => {
    setSelectedServiceId(event.target.value);
  };

  const handleRefresh = () => {
    refetch();
  };

  if (servicesLoading) {
    return (
      <Page title={Messages.title}>
        <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
          <CircularProgress />
        </Box>
      </Page>
    );
  }

  if (servicesError) {
    return (
      <Page title={Messages.title}>
        <Alert severity="error">
          {Messages.errorLoadingServices}
        </Alert>
      </Page>
    );
  }

  const enabledServices = servicesData?.filter((s: any) => s.isEnabled) || [];

  return (
    <Page title={Messages.title}>
      <Box sx={{ width: '100%', maxWidth: 'none', margin: 0, padding: 0 }}>
        <Stack spacing={3}>
        {/* Service Selection and Controls */}
        <Card>
          <CardContent>
            <Grid container spacing={2} alignItems="center">
              <Grid item xs={12} md={4}>
                <FormControl fullWidth>
                  <InputLabel>{Messages.selectService}</InputLabel>
                  <Select
                    value={selectedServiceId}
                    onChange={handleServiceChange}
                    label={Messages.selectService}
                  >
                    <MenuItem value="">
                      <em>{Messages.allServices}</em>
                    </MenuItem>
                    {enabledServices.map((service: any) => (
                      <MenuItem key={service.serviceId} value={service.serviceId}>
                        {service.serviceName} ({service.serviceType})
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} md={4}>
                <TextField
                  fullWidth
                  placeholder={Messages.searchQueries}
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  InputProps={{
                    startAdornment: (
                      <InputAdornment position="start">
                        <SearchIcon />
                      </InputAdornment>
                    ),
                  }}
                />
              </Grid>
              <Grid item xs={12} md={4}>
                <Stack direction="row" spacing={1}>
                  <Tooltip title={Messages.refresh}>
                    <IconButton onClick={handleRefresh} disabled={queriesLoading}>
                      <RefreshIcon />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title={Messages.settings}>
                    <IconButton onClick={() => setConfigDialogOpen(true)}>
                      <SettingsIcon />
                    </IconButton>
                  </Tooltip>
                </Stack>
              </Grid>
            </Grid>
          </CardContent>
        </Card>

        {/* Service Cards */}
        {enabledServices.length > 0 && (
          <Grid container spacing={2}>
            {enabledServices.map((service: any) => (
              <Grid item xs={12} md={6} lg={4} key={service.serviceId}>
                <RealTimeServiceCard
                  service={service}
                  isSelected={selectedServiceId === service.serviceId}
                  onSelect={() => setSelectedServiceId(service.serviceId)}
                />
              </Grid>
            ))}
          </Grid>
        )}

        {/* Main Content Tabs */}
        <Card>
          <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
            <Tabs value={tabValue} onChange={handleTabChange}>
              <Tab label={Messages.overview} />
              <Tab label={Messages.queries} />
              <Tab label={Messages.charts} />
            </Tabs>
          </Box>

          <TabPanel value={tabValue} index={0}>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      {Messages.runningQueries}
                    </Typography>
                    <Typography variant="h3" color="primary">
                      {runningQueries.length}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {Messages.currentlyExecuting}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} md={6}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      {Messages.totalQueries}
                    </Typography>
                    <Typography variant="h3" color="primary">
                      {queriesData?.totalCount || 0}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {Messages.inLast2Minutes}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          </TabPanel>

          <TabPanel value={tabValue} index={1}>
            <Stack spacing={2}>
              <Box display="flex" gap={1} flexWrap="wrap">
                <Chip
                  label={Messages.all}
                  onClick={() => setStateFilter('all')}
                  color={stateFilter === 'all' ? 'primary' : 'default'}
                />
                <Chip
                  label={Messages.running}
                  onClick={() => setStateFilter(QueryState.RUNNING)}
                  color={stateFilter === QueryState.RUNNING ? 'primary' : 'default'}
                />
                <Chip
                  label={Messages.finished}
                  onClick={() => setStateFilter(QueryState.FINISHED)}
                  color={stateFilter === QueryState.FINISHED ? 'primary' : 'default'}
                />
                <Chip
                  label={Messages.waiting}
                  onClick={() => setStateFilter(QueryState.WAITING)}
                  color={stateFilter === QueryState.WAITING ? 'primary' : 'default'}
                />
              </Box>
              
              {queriesLoading ? (
                <Box display="flex" justifyContent="center" p={3}>
                  <CircularProgress />
                </Box>
              ) : queriesError ? (
                <Alert severity="error">
                  {Messages.errorLoadingQueries}
                </Alert>
              ) : (
                <RealTimeQueryTable queries={filteredQueries} />
              )}
            </Stack>
          </TabPanel>

          <TabPanel value={tabValue} index={2}>
            {queriesLoading ? (
              <Box display="flex" justifyContent="center" p={3}>
                <CircularProgress />
              </Box>
            ) : queriesError ? (
              <Alert severity="error">
                {Messages.errorLoadingQueries}
              </Alert>
            ) : (
              <RealTimeQueryChart queries={filteredQueries} />
            )}
          </TabPanel>
        </Card>
      </Stack>

      <RealTimeConfigDialog
        open={configDialogOpen}
        onClose={() => setConfigDialogOpen(false)}
        services={enabledServices}
      />
      </Box>
    </Page>
  );
};
