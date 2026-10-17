import {
  Alert,
  Box,
  LinearProgress,
  Link,
  Stack,
  Tab,
  Tabs,
} from '@mui/material';
import { FC, useCallback, useMemo, useState } from 'react';
import {
  Link as RouterLink,
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
import { LogsVolumeChart } from './components/LogsVolumeChart';
import {
  filterFromSearchParams,
  QuickRange,
  rangeFromSearchParams,
  searchParamsFromFilter,
} from './utils/logsFilter';

type LogsTab = 'lines' | 'patterns';

const hasFilters = (f: LogFilter) =>
  !!(
    f.nodeNames ||
    f.sources ||
    f.presets ||
    f.minSeverity ||
    f.text ||
    f.pattern
  );

export const LogsPage: FC = () => {
  const { tab } = useParams<{ tab?: string }>();
  const activeTab: LogsTab = tab === 'patterns' ? 'patterns' : 'lines';
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  // A quick range ends at `now`, fixed until the filter changes or Refresh is clicked, so query keys stay stable.
  const [now, setNow] = useState(() => new Date());
  const filter = useMemo(
    () => filterFromSearchParams(searchParams, now),
    [searchParams, now]
  );
  const range = rangeFromSearchParams(searchParams);
  const volume = useLogVolume(filter);
  const facets = useLogFacets(filter);

  const update = useCallback(
    (next: LogFilter, nextRange?: QuickRange) => {
      setNow(new Date());
      setSearchParams(searchParamsFromFilter(next, nextRange));
    },
    [setSearchParams]
  );

  const zoom = (fromMs: number, toMs: number) =>
    update({
      ...filter,
      start: new Date(fromMs).toISOString(),
      end: new Date(toMs).toISOString(),
    });

  return (
    <Page title="Logs" wide>
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
        (volume.data.buckets.length ? (
          <LogsVolumeChart
            buckets={volume.data.buckets}
            step={volume.data.step}
            onZoom={zoom}
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
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab data-testid="logs-tab-lines" value="lines" label="Lines" />
          <Tab
            data-testid="logs-tab-patterns"
            value="patterns"
            label="Patterns"
          />
        </Tabs>
        {activeTab === 'lines' && <LogsLinesTab filter={filter} />}
        {activeTab === 'patterns' && <Box data-testid="logs-patterns-tab" />}
      </Stack>
    </Page>
  );
};
