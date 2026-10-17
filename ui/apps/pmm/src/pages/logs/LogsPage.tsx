import {
  Alert,
  Box,
  LinearProgress,
  Link,
  Stack,
  Tab,
  Tabs,
  ThemeProvider,
} from '@mui/material';
import { FC, useCallback, useMemo, useState } from 'react';
import {
  Link as RouterLink,
  useLocation,
  useNavigate,
  useParams,
  useSearchParams,
} from 'react-router-dom';
import { LogFilter } from 'api/logs';
import { Page } from 'components/page';
import { useLogFacets, useLogVolume } from 'hooks/api/useLogs';
import { PMM_LOGS_PATH } from 'lib/constants';
import { LogsErrorAlert } from './components/LogsErrorAlert';
import { LogsFilterBar } from './components/LogsFilterBar';
import { LogsLinesTab } from './components/LogsLinesTab';
import { LogsPatternsTab } from './components/LogsPatternsTab';
import { LogsVolumeChart } from './components/LogsVolumeChart';
import { logsTheme } from './logsTheme';
import {
  filterFromSearchParams,
  hasFilters,
  QuickRange,
  rangeFromSearchParams,
  searchParamsFromFilter,
  severityClassOf,
  toggleSeverityClass,
} from './utils/logsFilter';

type LogsTab = 'lines' | 'patterns';

export const LogsPage: FC = () => {
  const { tab } = useParams<{ tab?: string }>();
  const activeTab: LogsTab = tab === 'patterns' ? 'patterns' : 'lines';
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  // A quick range ends at `now`, fixed until the filter changes or Refresh is clicked, so query keys stay stable.
  const [now, setNow] = useState(() => new Date());
  const filter = useMemo(
    () => filterFromSearchParams(searchParams, now),
    [searchParams, now]
  );
  const range = rangeFromSearchParams(searchParams);
  const volume = useLogVolume(filter);
  const total = volume.data?.buckets.reduce(
    (sum, b) => sum + b.error + b.warning + b.info + b.other,
    0
  );
  const facets = useLogFacets(filter);

  const update = useCallback(
    (next: LogFilter, nextRange?: QuickRange) => {
      setNow(new Date());
      setSearchParams(searchParamsFromFilter(next, nextRange));
    },
    [setSearchParams]
  );

  // A zoom is a history entry marked as one, so Back returns to the range it came from.
  const zoom = (fromMs: number, toMs: number) =>
    setSearchParams(
      searchParamsFromFilter({
        ...filter,
        start: new Date(fromMs).toISOString(),
        end: new Date(toMs).toISOString(),
      }),
      { state: { zoomed: true } }
    );
  const zoomed = !!(location.state as { zoomed?: boolean } | null)?.zoomed;

  return (
    <ThemeProvider theme={logsTheme}>
      <Page title="Logs" wide surface="default">
        <LogsFilterBar
          filter={filter}
          range={range}
          facets={facets.data}
          onChange={update}
          onRefresh={() => setNow(new Date())}
        />
        {volume.isLoading && <LinearProgress />}
        {volume.error && <LogsErrorAlert error={volume.error} />}
        {volume.data &&
          (total ? (
            <LogsVolumeChart
              buckets={volume.data.buckets}
              step={volume.data.step}
              onZoom={zoom}
              onBack={zoomed ? () => navigate(-1) : undefined}
              selectedClass={severityClassOf(filter)}
              onSelectClass={(cls) =>
                update(toggleSeverityClass(filter, cls), range)
              }
            />
          ) : (
            <Alert severity="info" data-testid="logs-empty-state">
              {hasFilters(filter) ? (
                'No log lines match these filters in this time range.'
              ) : (
                <>
                  No log lines in this time range. Logs are collected per node;
                  see{' '}
                  <Link component={RouterLink} to="/settings/otel">
                    Settings → OTEL
                  </Link>{' '}
                  and the <code>pmm-admin add otel</code> command.
                </>
              )}
            </Alert>
          ))}
        <Stack sx={{ flex: 1, minHeight: 0 }}>
          <Tabs
            value={activeTab}
            onChange={(_, value: LogsTab) =>
              navigate({
                pathname: `${PMM_LOGS_PATH}/${value}`,
                search: searchParams.toString(),
              })
            }
            sx={{ flexShrink: 0, borderBottom: 1, borderColor: 'divider' }}
          >
            <Tab
              data-testid="logs-tab-lines"
              value="lines"
              label={
                <Stack direction="row" spacing={1} alignItems="center">
                  <span>Lines</span>{' '}
                  {total !== undefined && (
                    <Box
                      component="span"
                      sx={{
                        px: 0.75,
                        borderRadius: 2,
                        fontSize: 12,
                        color: 'text.secondary',
                        bgcolor: 'action.hover',
                        fontVariantNumeric: 'tabular-nums',
                      }}
                    >
                      {total.toLocaleString()}
                    </Box>
                  )}
                </Stack>
              }
            />
            <Tab
              data-testid="logs-tab-patterns"
              value="patterns"
              label="Patterns"
            />
          </Tabs>
          {activeTab === 'lines' && <LogsLinesTab filter={filter} />}
          {activeTab === 'patterns' && (
            <LogsPatternsTab
              filter={filter}
              onSelect={(pattern) =>
                navigate({
                  pathname: `${PMM_LOGS_PATH}/lines`,
                  search: searchParamsFromFilter(
                    { ...filter, pattern },
                    range
                  ).toString(),
                })
              }
            />
          )}
        </Stack>
      </Page>
    </ThemeProvider>
  );
};
