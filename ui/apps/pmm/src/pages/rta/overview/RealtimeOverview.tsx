import { useMemo, useRef, useState } from 'react';
import type { FC } from 'react';
import {
  Navigate,
  Link as RouterLink,
  useSearchParams,
} from 'react-router-dom';
import { RealtimePage } from '../components/rta-page';
import { useRealtimeQueries, useRealtimeSessions } from 'hooks/api/useRealtime';
import OverviewTable from './table/OverviewTable';
import {
  isBlocked,
  isBlockingUnattributed,
  isBlockingUnknown,
  isSameStatement,
  isTransactionControl,
} from './table/OverviewTable.utils';
import { useStatementNavigation } from './useStatementNavigation';
import { DetailsPane } from './details-pane';
import type { QueryData } from 'types/rta.types';
import DynamicFeed from '@mui/icons-material/DynamicFeed';
import FileDownloadOutlined from '@mui/icons-material/FileDownloadOutlined';
import Pause from '@mui/icons-material/Pause';
import PlayArrow from '@mui/icons-material/PlayArrow';
import Refresh from '@mui/icons-material/Refresh';
import { Messages } from './RealtimeOverview.messages';
import { createRealtimeSessionsUrl } from 'utils/link.utils';
import Stack from '@mui/material/Stack';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import FormControlLabel from '@mui/material/FormControlLabel';
import Switch from '@mui/material/Switch';
import Tooltip from '@mui/material/Tooltip';
import { ServicesAutocompleteInput } from '../components/services-autocomplete-input';
import { AutoRefreshSelect } from './auto-refresh-select';
import { exportRtaQueriesToCsv } from './export/exportRtaQueriesToCsv';
import { ServiceType } from 'types/services.types';
import {
  blockedOnlyTooltip,
  resolveSelection,
  sessionErrorsMessage,
} from './RealtimeOverview.utils';

const EMPTY_QUERIES: QueryData[] = [];

// Narrowest width that fits the paused MySQL toolbar on one row with the sidebar expanded.
const TOOLBAR_ONE_ROW = '@media (min-width: 1600px)';

const RealtimeOverviewPage: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  // "?serviceIds=" yields an empty string, which the API rejects.
  const requestedServiceIds = useMemo(
    () => searchParams.getAll('serviceIds').filter(Boolean),
    [searchParams]
  );
  // Polled because a session can fail after the view opens, and the empty state shows why.
  const { data: sessions = [], isLoading } = useRealtimeSessions({
    refetchInterval: 5000,
  });
  // A URL can name services of both technologies; the first service's technology wins.
  const { serviceIds, serviceType } = useMemo(
    () => resolveSelection(requestedServiceIds, sessions),
    [requestedServiceIds, sessions]
  );
  const [fetching, setFetching] = useState(serviceIds.length > 0);
  const [refreshInterval, setRefreshInterval] = useState(2000);
  const { data: queries, refetch } = useRealtimeQueries(
    { serviceIds },
    {
      // The API rejects a request that names no service.
      enabled: fetching && serviceIds.length > 0,
      refetchInterval: refreshInterval,
    }
  );
  const [hideCommit, setHideCommit] = useState(false);
  const [blockedOnly, setBlockedOnly] = useState(false);
  const isMySqlSelection = serviceType === ServiceType.mysql;
  // Synced from the table after filters; details-pane arrows use this list, not the full API result.
  const [navigableQueries, setNavigableQueries] = useState<QueryData[]>([]);
  const [selectedQuery, setSelectedQuery] = useState<QueryData>();
  // We need to store the previous fetching state to restore it when the details pane is closed
  const previousFetchingState = useRef<boolean>(fetching);
  // Gated on the toggle being on screen, so a hidden filter cannot hide rows or shrink the CSV.
  const hideTransactionControl = hideCommit && isMySqlSelection;
  // Gated the same way: lock waits are reported for MySQL only.
  const showBlockedOnly = blockedOnly && isMySqlSelection;
  // The blocked count excludes hidden transaction-control rows, but not MRT's column filters,
  // so a Database or User filter can leave the label higher than the row count.
  const visibleQueries = useMemo(() => {
    const allQueries = queries ?? EMPTY_QUERIES;

    return hideTransactionControl
      ? allQueries.filter((query) => !isTransactionControl(query))
      : allQueries;
  }, [queries, hideTransactionControl]);
  // "Blocked only (0)" here would present a monitoring gap as a healthy server.
  const blockingUnknown = useMemo(
    () => visibleQueries.length > 0 && visibleQueries.every(isBlockingUnknown),
    [visibleQueries]
  );
  // Only one lock source answered, as on a stock MariaDB without the metadata-lock instrument.
  const blockingPartial = useMemo(
    () => !blockingUnknown && visibleQueries.some(isBlockingUnknown),
    [blockingUnknown, visibleQueries]
  );
  // Only confirmed waits are counted: an undecided row is not evidence of waiting.
  const blockedQueries = useMemo(
    () => visibleQueries.filter(isBlocked),
    [visibleQueries]
  );
  // Counted apart from blockingPartial so the tooltip asks for a refresh, not a config change.
  const unattributedCount = useMemo(
    () => visibleQueries.filter(isBlockingUnattributed).length,
    [visibleQueries]
  );
  // Undecided rows stay visible, so the filter never hides a statement that may be waiting.
  const filteredQueries = useMemo(
    () =>
      visibleQueries.filter(
        (query) =>
          isBlocked(query) ||
          isBlockingUnknown(query) ||
          isBlockingUnattributed(query)
      ),
    [visibleQueries]
  );
  const tableQueries = showBlockedOnly ? filteredQueries : visibleQueries;
  const noDataMessage = useMemo(
    () => sessionErrorsMessage(serviceIds, sessions),
    [serviceIds, sessions]
  );
  const blockedCount = blockedQueries.length;

  const handleQuerySelected = (query: QueryData) => {
    // Previous and next also select here while the pane has paused the view; saving then would
    // leave the view paused on close.
    if (!selectedQuery) {
      previousFetchingState.current = fetching;
    }
    setSelectedQuery(query);
    setFetching(false);
  };

  const handleCloseDetails = () => {
    setSelectedQuery(undefined);
    // The selection may have been emptied while the pane was open.
    setFetching(previousFetchingState.current && serviceIds.length > 0);
  };

  // The pane reports the opened statement as finished rather than showing what its connection
  // ran next.
  const selectedFinished = useMemo(
    () =>
      !!selectedQuery &&
      !(queries ?? EMPTY_QUERIES).some((query) =>
        isSameStatement(selectedQuery, query)
      ),
    [selectedQuery, queries]
  );

  const { isFirst, isLast, next, previous } = useStatementNavigation({
    rows: navigableQueries,
    selected: selectedQuery,
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

    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete('serviceIds');
      newServiceIds.forEach((id) => next.append('serviceIds', id));
      return next;
    });
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
        serviceType={serviceType}
        noDataMessage={noDataMessage}
        onQuerySelected={handleQuerySelected}
        onNavigableQueriesChange={setNavigableQueries}
        actions={({ table }) => (
          <Stack
            flex={1}
            direction="row"
            alignItems="flex-start"
            alignContent="flex-start"
            rowGap={0}
            columnGap={1}
            sx={{
              width: '100%',
              minWidth: 0,
              // Narrower than TOOLBAR_ONE_ROW, the row filters wrap to a second row so
              // "All sessions" keeps its place at the right end of the first.
              flexWrap: 'wrap',
              [TOOLBAR_ONE_ROW]: { flexWrap: 'nowrap' },
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
                singleTechnology
                onServiceIdsChange={handleServiceIdsChange}
                inputProps={{
                  size: 'small',
                }}
              />
            </Box>
            <Stack
              direction="row"
              alignItems="center"
              gap={1}
              sx={{ mt: 1, minWidth: 0, flex: '0 0 auto' }}
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
                startIcon={fetching ? <Pause /> : <PlayArrow />}
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
              {/* Icons, not labelled buttons, so the toolbar fits one line at common widths. */}
              {!fetching && serviceIds.length !== 0 && (
                <Tooltip title={Messages.refresh} arrow>
                  <IconButton
                    data-testid="overview-table-refresh-button"
                    aria-label={Messages.refresh}
                    onClick={() => refetch()}
                    color="inherit"
                  >
                    <Refresh />
                  </IconButton>
                </Tooltip>
              )}
              {!fetching && (
                <Tooltip title={Messages.exportTooltip} arrow>
                  {/* A disabled button fires no events, so the tooltip hangs off a span. */}
                  <span>
                    <IconButton
                      data-testid="overview-table-export-button"
                      aria-label={Messages.exportTooltip}
                      disabled={
                        serviceIds.length === 0 ||
                        table.getPrePaginationRowModel().rows.length === 0
                      }
                      onClick={() =>
                        exportRtaQueriesToCsv(
                          table
                            .getPrePaginationRowModel()
                            .rows.map((row) => row.original),
                          serviceType
                        )
                      }
                      color="inherit"
                    >
                      <FileDownloadOutlined />
                    </IconButton>
                  </span>
                </Tooltip>
              )}
            </Stack>
            {/* Row filters, wrapped as one unit so a narrow toolbar never splits them. */}
            {isMySqlSelection && (
              <Stack
                direction="row"
                alignItems="center"
                gap={1}
                sx={{
                  mt: 1,
                  flex: '0 0 auto',
                  order: 3,
                  flexBasis: '100%',
                  [TOOLBAR_ONE_ROW]: { order: 0, flexBasis: 'auto' },
                }}
              >
                <Divider
                  orientation="vertical"
                  flexItem
                  sx={{
                    my: 1,
                    mx: 0.5,
                    display: 'none',
                    [TOOLBAR_ONE_ROW]: { display: 'block' },
                  }}
                />
                <Tooltip
                  title={
                    blockingUnknown
                      ? Messages.blockedUnknownTooltip
                      : blockedOnlyTooltip(blockingPartial, unattributedCount)
                  }
                  arrow
                >
                  <FormControlLabel
                    data-testid="overview-table-blocked-only-toggle"
                    disabled={blockingUnknown}
                    control={
                      <Switch
                        size="small"
                        checked={blockedOnly && !blockingUnknown}
                        onChange={(event) =>
                          setBlockedOnly(event.target.checked)
                        }
                      />
                    }
                    label={
                      blockingUnknown
                        ? Messages.blockedUnknown
                        : Messages.blockedOnly(blockedCount)
                    }
                    // ml: 0 lines a wrapped row up with the service picker; see below.
                    sx={{ whiteSpace: 'nowrap', ml: 0, mr: 1 }}
                  />
                </Tooltip>
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
                    // ml resets FormControlLabel's negative margin, which would pull this
                    // toggle flush against the previous label.
                    sx={{ whiteSpace: 'nowrap', ml: 0, mr: 0 }}
                  />
                </Tooltip>
              </Stack>
            )}
            <Box
              sx={{
                flex: '0 0 auto',
                ml: { md: 'auto' },
                my: 1,
                whiteSpace: 'nowrap',
                order: 2,
                [TOOLBAR_ONE_ROW]: { order: 0 },
              }}
            >
              <Button
                color="inherit"
                data-testid="overview-table-all-sessions-button"
                startIcon={<DynamicFeed />}
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
        finished={selectedFinished}
        onClose={handleCloseDetails}
        isFirstQuery={isFirst}
        isLastQuery={isLast}
        onNext={next}
        onPrevious={previous}
        // The pane covers the toolbar, so it carries its own refresh.
        onRefresh={serviceIds.length > 0 ? () => refetch() : undefined}
      />
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
