import { useMemo, useRef, useState } from 'react';
import type { FC } from 'react';
import {
  Navigate,
  Link as RouterLink,
  useSearchParams,
} from 'react-router-dom';
import { useDetailsPaneNavigation } from '@percona/percona-ui';
import { RealtimePage } from '../components/rta-page';
import { useRealtimeQueries, useRealtimeSessions } from 'hooks/api/useRealtime';
import OverviewTable from './table/OverviewTable';
import { isTransactionControl } from './table/OverviewTable.utils';
import { DetailsPane } from './details-pane';
import type { QueryData } from 'types/rta.types';
import { Icon } from 'components/icon';
import { Messages } from './RealtimeOverview.messages';
import { createRealtimeSessionsUrl } from 'utils/link.utils';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch';
import Tooltip from '@mui/material/Tooltip';
import { ServicesAutocompleteInput } from '../components/services-autocomplete-input';
import { AutoRefreshSelect } from './auto-refresh-select';
import { exportRtaQueriesToCsv } from './export/exportRtaQueriesToCsv';
import { ServiceType } from 'types/services.types';

const EMPTY_QUERIES: QueryData[] = [];

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
  const [hideCommit, setHideCommit] = useState(false);
  // Synced from the table after filters; details-pane arrows use this list, not the full API result.
  const [navigableQueries, setNavigableQueries] = useState<QueryData[]>([]);
  const [selectedQuery, setSelectedQuery] = useState<QueryData>();
  // We need to store the previous fetching state to restore it when the details pane is closed
  const previousFetchingState = useRef<boolean>(fetching);
  const { data: sessions = [], isLoading } = useRealtimeSessions();
  // Transaction-control statements are a MySQL concern, so the toggle is not
  // shown to users whose sessions are all MongoDB. It follows the sessions
  // rather than the current selection, so it does not appear and disappear
  // while services are being picked.
  const hasMySqlSession = sessions.some(
    (session) => session.serviceType === ServiceType.mysql
  );
  // Gated on the toggle being on screen: if the last MySQL session goes away the
  // control unmounts, and a filter nobody can see must not keep hiding rows (nor
  // silently shrink the CSV export, which exports the filtered rows).
  const hideTransactionControl = hideCommit && hasMySqlSession;
  const tableQueries = useMemo(() => {
    const allQueries = queries ?? EMPTY_QUERIES;
    return hideTransactionControl
      ? allQueries.filter((query) => !isTransactionControl(query))
      : allQueries;
  }, [queries, hideTransactionControl]);

  const handleQuerySelected = (query: QueryData) => {
    setSelectedQuery(query);
    previousFetchingState.current = fetching;
    setFetching(false);
  };

  const handleCloseDetails = () => {
    setSelectedQuery(undefined);
    setFetching(previousFetchingState.current);
  };

  const { isFirst, isLast, next, previous } =
    useDetailsPaneNavigation<QueryData>({
      rows: navigableQueries,
      selected: selectedQuery,
      getRowId: (query) => query.queryId,
      onSelect: handleQuerySelected,
    });

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
            flex={1}
            direction="row"
            flexWrap="wrap"
            alignItems="flex-start"
            alignContent="flex-start"
            rowGap={0}
            columnGap={1}
            sx={{
              width: '100%',
              minWidth: 0,
            }}
          >
            <Box
              sx={{
                flex: '1 1 320px',
                minWidth: 200,
                maxWidth: { xs: '100%', md: 320 },
                pr: { md: 1 },
              }}
            >
              <ServicesAutocompleteInput
                data-testid="overview-table-services-autocomplete-input"
                sessions={sessions}
                serviceIds={serviceIds}
                onServiceIdsChange={handleServiceIdsChange}
                inputProps={{
                  size: 'small',
                }}
              />
            </Box>
            <Stack
              direction="row"
              flexWrap="wrap"
              alignItems="center"
              gap={1}
              sx={{ mt: 1 }}
            >
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
                size="medium"
                startIcon={
                  fetching ? <Icon name="pause" /> : <Icon name="play-arrow" />
                }
                disabled={serviceIds.length === 0}
                color="inherit"
                variant="text"
                onClick={() => setFetching(!fetching)}
                disableElevation
                sx={
                  !fetching && serviceIds.length > 0
                    ? { backgroundColor: 'action.selected' }
                    : undefined
                }
              >
                {fetching ? Messages.pause : Messages.resume}
              </Button>
              {!fetching && serviceIds.length !== 0 && (
                <Button
                  data-testid="overview-table-refresh-button"
                  size="medium"
                  startIcon={<Icon name="refresh" />}
                  onClick={() => refetch()}
                  color="inherit"
                  disableElevation
                >
                  {Messages.refresh}
                </Button>
              )}
              {!fetching && (
                <Button
                  data-testid="overview-table-export-button"
                  size="small"
                  variant="text"
                  startIcon={<Icon name="file-download" />}
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
                  sx={{
                    width: 100,
                    height: 36,
                  }}
                >
                  {Messages.export}
                </Button>
              )}
              {/* This filters the rows, it does not drive live updates: keep it
                  out of the auto-refresh / playback group so that group reads as
                  one control. */}
              {hasMySqlSession && (
                <>
                  <Divider
                    orientation="vertical"
                    flexItem
                    sx={{ my: 1, mx: 0.5 }}
                  />
                  <Tooltip title={Messages.hideCommitTooltip} arrow>
                    <FormControlLabel
                      data-testid="overview-table-hide-commit-toggle"
                      control={
                        <Switch
                          size="small"
                          checked={hideCommit}
                          onChange={(event) =>
                            setHideCommit(event.target.checked)
                          }
                        />
                      }
                      label={Messages.hideCommit}
                      sx={{ whiteSpace: 'nowrap', mr: 0 }}
                    />
                  </Tooltip>
                </>
              )}
            </Stack>
            <Box sx={{ flex: '0 0 auto', ml: { md: 'auto' }, my: 1 }}>
              <Button
                color="inherit"
                data-testid="overview-table-all-sessions-button"
                startIcon={<Icon name="dynamic-feed" />}
                component={RouterLink}
                size="medium"
                to={createRealtimeSessionsUrl(serviceIds)}
              >
                {Messages.allSessions}
              </Button>
            </Box>
          </Stack>
        )}
      />
      <DetailsPane
        query={selectedQuery}
        onClose={handleCloseDetails}
        isFirstQuery={isFirst}
        isLastQuery={isLast}
        onNext={next}
        onPrevious={previous}
      />
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
