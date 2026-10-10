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

// The widest the paused MySQL toolbar needs to sit on one row, with the navigation sidebar
// expanded: the service picker, playback controls with Refresh and Export, both row filters and
// "All sessions". Measured at 1680px with 16px to spare.
const TOOLBAR_ONE_ROW = '@media (min-width: 1600px)';

const RealtimeOverviewPage: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  // Blanks are dropped: "?serviceIds=" yields one empty string, which is not a selection
  // and which the API rejects outright.
  const requestedServiceIds = useMemo(
    () => searchParams.getAll('serviceIds').filter(Boolean),
    [searchParams]
  );
  // Polled like the sessions list: a session that fails to start does so after the
  // view has opened, and its reason is what the empty state shows.
  const { data: sessions = [], isLoading } = useRealtimeSessions({
    refetchInterval: 5000,
  });
  // One view of live queries shows one technology. The picker enforces that, but
  // a URL can still name services of both (starting sessions is not restricted),
  // so the first service's technology wins and the rest are ignored.
  const { serviceIds, serviceType } = useMemo(
    () => resolveSelection(requestedServiceIds, sessions),
    [requestedServiceIds, sessions]
  );
  const [fetching, setFetching] = useState(serviceIds.length > 0);
  const [refreshInterval, setRefreshInterval] = useState(2000);
  const { data: queries, refetch } = useRealtimeQueries(
    { serviceIds },
    {
      // Belt and braces. handleCloseDetails is where fetching and the selection can fall out
      // of step, and it now reconciles them, but the request is rejected outright when it
      // names no service so the invariant is asserted here too.
      enabled: fetching && serviceIds.length > 0,
      refetchInterval: refreshInterval,
    }
  );
  const [hideCommit, setHideCommit] = useState(false);
  const [blockedOnly, setBlockedOnly] = useState(false);
  // Lock waits are reported for MySQL and PostgreSQL. Hiding transaction-control statements
  // is MySQL-only: PostgreSQL lists a session idle in transaction by its last statement,
  // often BEGIN, and hiding it would hide a session holding a transaction open.
  const isMySqlSelection = serviceType === ServiceType.mysql;
  const isSqlSelection =
    isMySqlSelection || serviceType === ServiceType.posgresql;
  // Synced from the table after filters; details-pane arrows use this list, not the full API result.
  const [navigableQueries, setNavigableQueries] = useState<QueryData[]>([]);
  const [selectedQuery, setSelectedQuery] = useState<QueryData>();
  // We need to store the previous fetching state to restore it when the details pane is closed
  const previousFetchingState = useRef<boolean>(fetching);
  // Gated on the toggle being on screen: when the selection stops being MySQL
  // the control unmounts, and a filter nobody can see must not keep hiding rows
  // (nor silently shrink the CSV export, which exports the filtered rows).
  const hideTransactionControl = hideCommit && isMySqlSelection;
  // Gated the same way as the transaction-control toggle: a filter that has left the screen
  // must not keep hiding rows.
  const showBlockedOnly = blockedOnly && isSqlSelection;
  // Split out so the toggle label counts the rows switching it on would leave, rather than
  // every blocked row in the response. It is still counted before the table's own column
  // filters, which live inside MRT and are not visible here, so a Database or User filter can
  // leave the label higher than the row count.
  const visibleQueries = useMemo(() => {
    const allQueries = queries ?? EMPTY_QUERIES;

    return hideTransactionControl
      ? allQueries.filter((query) => !isTransactionControl(query))
      : allQueries;
  }, [queries, hideTransactionControl]);
  // The agent could not read the lock graph at all, so nothing is known about waiting.
  // Reporting "Blocked only (0)" here would present a monitoring gap as a verified healthy
  // server, which is the one thing this must not do during an incident.
  const blockingUnknown = useMemo(
    () => visibleQueries.length > 0 && visibleQueries.every(isBlockingUnknown),
    [visibleQueries]
  );
  // One of the two lock sources answered and the other did not, so some rows have a verdict
  // and some do not. This is the normal state on a stock MariaDB, where the metadata-lock
  // instrument ships disabled while row locks are readable. Filtering to the known blocked
  // rows would then hide the statements queued behind a DDL -- stuck, and silently dropped
  // from the one view meant to show them.
  const blockingPartial = useMemo(
    () => !blockingUnknown && visibleQueries.some(isBlockingUnknown),
    [blockingUnknown, visibleQueries]
  );
  // Confirmed blocked, which is what the label counts: a row nobody could judge is not
  // evidence of waiting and must not inflate the number.
  const blockedQueries = useMemo(
    () => visibleQueries.filter(isBlocked),
    [visibleQueries]
  );
  // The connection was waiting, but for a later statement than the one sampled. Counted apart
  // from blockingPartial: every lock source answered, so the reader needs a refresh, not a
  // configuration change, and the tooltip must not send them to fix one.
  const unattributedCount = useMemo(
    () => visibleQueries.filter(isBlockingUnattributed).length,
    [visibleQueries]
  );
  // What the filter shows. Undecided rows stay visible, so the filter never hides a statement
  // that may be waiting.
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
    // Only on opening the pane. Previous and next select through here too, while the view is
    // already paused by the pane, and saving then would make closing it leave the view paused.
    if (!selectedQuery) {
      previousFetchingState.current = fetching;
    }
    setSelectedQuery(query);
    setFetching(false);
  };

  const handleCloseDetails = () => {
    setSelectedQuery(undefined);
    // The selection can be emptied while the pane is open, which makes the state captured on
    // open stale: restoring it unchecked resumes polling with nothing selected, and the API
    // rejects a request that names no service.
    setFetching(previousFetchingState.current && serviceIds.length > 0);
  };

  // The pane keeps showing the statement that was opened. When a later read no longer has it
  // running, the pane says so rather than swapping in whatever its connection ran next (a
  // MySQL row is keyed by connection id) -- the same outcome as for a MongoDB operation, whose
  // row simply disappears when it ends.
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
              // One row where everything fits, paused or live. Narrower, the row filters
              // take a second row of their own (see TOOLBAR_ONE_ROW), so "All sessions"
              // keeps its place at the right end of the first.
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
              {/* Refresh and Export only appear while paused. As labelled buttons they
                  pushed the toolbar past one line at common widths, dropping "All
                  sessions" to a second row, so they are icons with tooltips. */}
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
                            .rows.map((row) => row.original)
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
            {/* These filter the rows, they do not drive live updates: kept out of the
                auto-refresh / playback group so that group reads as one control, and
                wrapped as one unit so a narrow toolbar never splits them. */}
            {isSqlSelection && (
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
                    // ml: see the toggle below. On a row of its own the group lines
                    // up with the service picker above it.
                    sx={{ whiteSpace: 'nowrap', ml: 0, mr: 1 }}
                  />
                </Tooltip>
                {isMySqlSelection && (
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
                      // ml resets the negative margin FormControlLabel applies to align a
                      // standalone switch; left in place it pulls this control flush against
                      // the previous label, so the two toggles read as one run of text.
                      sx={{ whiteSpace: 'nowrap', ml: 0, mr: 0 }}
                    />
                  </Tooltip>
                )}
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
        // The pane covers the toolbar, so it carries its own refresh. The view is paused while
        // the pane is open; this reads once, as the toolbar's Refresh does.
        onRefresh={serviceIds.length > 0 ? () => refetch() : undefined}
      />
    </RealtimePage>
  );
};

export default RealtimeOverviewPage;
