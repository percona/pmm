import { FC, useRef, useState } from 'react';
import {
  Navigate,
  Link as RouterLink,
  useSearchParams,
} from 'react-router-dom';
import { RealtimePage } from '../components/rta-page';
import { useRealtimeQueries, useRealtimeSessions } from 'hooks/api/useRealtime';
import OverviewTable from './table/OverviewTable';
import { DetailsPane } from './details-pane';
import { QueryData } from 'types/rta.types';
import { Icon } from 'components/icon';
import { Messages } from './RealtimeOverview.messages';
import { createRealtimeSessionsUrl } from 'utils/link.utils';
import FileDownloadOutlined from '@mui/icons-material/FileDownloadOutlined';
import { Tooltip } from '@percona/percona-ui';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import { ServicesAutocompleteInput } from '../components/services-autocomplete-input';
import { AutoRefreshSelect } from './auto-refresh-select';
import { exportRtaQueriesToCsv } from './export/exportRtaQueriesToCsv';

const EMPTY_QUERIES: QueryData[] = [];

const EXPORT_BUTTON_SX = {
  width: 100,
  height: 36,
  '&.Mui-disabled': {
    cursor: 'not-allowed',
    pointerEvents: 'auto',
  },
};

const RealtimeOverviewPage: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const serviceIds = searchParams.getAll('serviceIds');
  const [fetching, setFetching] = useState(serviceIds.length > 0);
  const [refreshInterval, setRefreshInterval] = useState(2000);
  const { data: queries, refetch } = useRealtimeQueries(
    { serviceIds },
    {
      enabled: fetching,
      refetchInterval: refreshInterval,
    }
  );
  const tableQueries = queries ?? EMPTY_QUERIES;
  // Synced from the table after filters; details-pane arrows use this list, not the full API result.
  const [navigableQueries, setNavigableQueries] = useState<QueryData[]>([]);
  const [selectedQuery, setSelectedQuery] = useState<QueryData>();
  // We need to store the previous fetching state to restore it when the details pane is closed
  const previousFetchingState = useRef<boolean>(fetching);
  const { data: sessions = [], isLoading } = useRealtimeSessions();

  const selectedQueryIndex = selectedQuery
    ? navigableQueries.findIndex(
        (query) => query.queryId === selectedQuery.queryId
      )
    : -1;

  const handleQuerySelected = (query: QueryData) => {
    setSelectedQuery(query);
    previousFetchingState.current = fetching;
    setFetching(false);
  };

  const handleCloseDetails = () => {
    setSelectedQuery(undefined);
    setFetching(previousFetchingState.current);
  };

  const handleAdjacentQuery = (offset: -1 | 1) => {
    if (selectedQueryIndex < 0) {
      return;
    }
    const nextIndex = selectedQueryIndex + offset;
    if (nextIndex < 0 || nextIndex >= navigableQueries.length) {
      return;
    }
    handleQuerySelected(navigableQueries[nextIndex]);
  };

  const handleServiceIdsChange = (newServiceIds: string[]) => {
    // start fetching if previous state was empty
    if (serviceIds.length === 0 && newServiceIds.length > 0) {
      setFetching(true);
    } else {
      setFetching((fetching) => {
        // if not fetching, don't start fetching
        if (!fetching) {
          return false;
        }

        return newServiceIds.length !== 0;
      });
    }

    setSearchParams({ serviceIds: newServiceIds });
  };

  if (isLoading) {
    return <RealtimePage />;
  }

  if (sessions.length === 0) {
    return <Navigate to="/rta/selection" />;
  }

  return (
    <RealtimePage>
      <OverviewTable
        queries={tableQueries}
        onQuerySelected={handleQuerySelected}
        onNavigableQueriesChange={setNavigableQueries}
        actions={({ table }) => (
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            sx={{
              pl: 2,
              flex: 1,
            }}
          >
            <Stack gap={2} direction="row" alignItems="center">
              <Stack sx={{ minWidth: 360 }}>
                <ServicesAutocompleteInput
                  data-testid="overview-table-services-autocomplete-input"
                  sessions={sessions}
                  serviceIds={serviceIds}
                  onServiceIdsChange={handleServiceIdsChange}
                  inputProps={{
                    size: 'small',
                  }}
                />
              </Stack>
              <AutoRefreshSelect
                isFetching={fetching}
                refreshInterval={refreshInterval}
                onRefreshIntervalChange={setRefreshInterval}
              />
              <Button
                data-testid={
                  fetching
                    ? 'overview-table-pause-button'
                    : 'overview-table-resume-button'
                }
                size="small"
                startIcon={
                  fetching ? <Icon name="pause" /> : <Icon name="play-arrow" />
                }
                disabled={serviceIds.length === 0}
                color={fetching ? 'inherit' : undefined}
                variant={fetching ? 'text' : 'contained'}
                onClick={() => setFetching(!fetching)}
                disableElevation
                sx={{
                  width: 100,
                  height: 32,
                }}
              >
                {fetching ? Messages.pause : Messages.resume}
              </Button>
              {!fetching && serviceIds.length !== 0 && (
                <Button
                  data-testid="overview-table-refresh-button"
                  size="small"
                  startIcon={<Icon name="refresh" />}
                  onClick={() => refetch()}
                  color="inherit"
                  disableElevation
                  sx={{
                    width: 100,
                    height: 36,
                  }}
                >
                  {Messages.refresh}
                </Button>
              )}
              {fetching ? (
                <Tooltip title={Messages.exportDisabledTooltip} arrow>
                  <Box
                    component="span"
                    sx={{ cursor: 'not-allowed', display: 'inline-flex' }}
                  >
                    <Button
                      data-testid="overview-table-export-button"
                      size="small"
                      variant="text"
                      startIcon={<FileDownloadOutlined />}
                      disabled
                      color="inherit"
                      disableElevation
                      sx={EXPORT_BUTTON_SX}
                    >
                      {Messages.export}
                    </Button>
                  </Box>
                </Tooltip>
              ) : (
                <Button
                  data-testid="overview-table-export-button"
                  size="small"
                  variant="text"
                  startIcon={<FileDownloadOutlined />}
                  disabled={
                    serviceIds.length === 0 ||
                    table.getPrePaginationRowModel().rows.length === 0
                  }
                  onClick={() =>
                    exportRtaQueriesToCsv(
                      table
                        .getPrePaginationRowModel()
                        .rows.map((row) => row.original)
                    )
                  }
                  color="inherit"
                  disableElevation
                  sx={EXPORT_BUTTON_SX}
                >
                  {Messages.export}
                </Button>
              )}
            </Stack>
            <Button
              color="inherit"
              data-testid="overview-table-all-sessions-button"
              startIcon={<Icon name="dynamic-feed" />}
              component={RouterLink}
              to={createRealtimeSessionsUrl(serviceIds)}
            >
              {Messages.allSessions}
            </Button>
          </Stack>
        )}
      />
      <DetailsPane
        query={selectedQuery}
        onClose={handleCloseDetails}
        isFirstQuery={selectedQueryIndex <= 0}
        isLastQuery={
          selectedQueryIndex < 0 ||
          selectedQueryIndex >= navigableQueries.length - 1
        }
        onNext={() => handleAdjacentQuery(1)}
        onPrevious={() => handleAdjacentQuery(-1)}
      />
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
