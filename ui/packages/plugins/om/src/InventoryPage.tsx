/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Stack,
  Tab,
  Tabs,
  Tooltip,
  Typography,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import { Table, type MRT_ColumnDef } from '@percona/percona-ui';
import {
  useInvalidateEstateOnRefreshEnd,
  useOmInventoryRuns,
  useRefreshInventory,
} from './inventoryHooks';
import { isRunActive, OmApiError } from './hooks';
import { ConfigForm } from './components/ConfigForm';
import { RunStatusBadge } from './components/HealthBadge';
import { RunEntities } from './components/RunEntities';
import { OmHeader } from './components/OmHeader';
import {
  formatCompactDuration,
  formatRunDuration,
  formatTimestamp,
  runDurationSeconds,
} from './format';
import { ageSeconds } from './inventory';
import type { OmInventoryRun } from './types';

const RUN_COLUMNS: MRT_ColumnDef<OmInventoryRun>[] = [
  {
    accessorKey: 'status',
    header: 'Status',
    Cell: ({ row: { original } }) => (
      <RunStatusBadge status={original.status} />
    ),
  },
  {
    accessorKey: 'start_time',
    header: 'Started',
    Cell: ({ row: { original } }) => formatTimestamp(original.start_time),
  },
  {
    // Sorts on elapsed seconds, not the formatted string -- lexicographically "9s"
    // lands after "10m". HostsPage's age column already does it this way.
    accessorFn: (row) => runDurationSeconds(row.start_time, row.end_time),
    id: 'duration',
    header: 'Duration',
    Cell: ({ row: { original } }) =>
      formatRunDuration(original.start_time, original.end_time) || '—',
  },
  {
    // Empty rather than zero for a full sweep: "the whole estate" is the ordinary
    // case, and a column that said "all" on nineteen rows out of twenty would be
    // noise. What matters is spotting the scoped one among them.
    accessorFn: (row) => row.scope.length,
    id: 'scope',
    header: 'Scope',
    Cell: ({ row: { original } }) =>
      original.scope.length === 0 ? (
        <Tooltip title="The whole estate">
          <Box component="span" sx={{ color: 'text.disabled' }}>
            all
          </Box>
        </Tooltip>
      ) : (
        <Tooltip title={original.scope.join(', ')}>
          <Box component="span">
            {original.scope.length} host
            {original.scope.length === 1 ? '' : 's'}
          </Box>
        </Tooltip>
      ),
  },
  {
    accessorFn: (row) => row.counts.answered_hosts,
    id: 'hosts',
    header: 'Hosts',
    Cell: ({ row: { original } }) => (
      // total / probeable / answered in one cell. The gap between the first two is
      // the estate nothing can be dispatched to, which is an onboarding fact rather
      // than a failed run, and the gap between the last two is what actually failed.
      <Tooltip
        title={`${original.counts.total_hosts} in scope, ${original.counts.probeable_hosts} with somewhere to run a probe, ${original.counts.answered_hosts} answered`}
      >
        <Box component="span">
          {original.counts.answered_hosts}/{original.counts.probeable_hosts}
          {original.counts.probeable_hosts === original.counts.total_hosts
            ? ''
            : ` of ${original.counts.total_hosts}`}
        </Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (row) => row.counts.total_services,
    id: 'services',
    header: 'Services',
    Cell: ({ row: { original } }) => original.counts.total_services,
  },
  {
    accessorFn: (row) => row.counts.resolved_services,
    id: 'resolved',
    header: 'Resolved',
    Cell: ({ row: { original } }) => (
      <Tooltip title="Services that mapped to a live executor host">
        <Box component="span">{original.counts.resolved_services}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (row) => row.counts.answered_services,
    id: 'answered',
    header: 'Answered',
    Cell: ({ row: { original } }) => (
      // The diagnostic pair: resolved says the mapping worked, answered says the
      // node ran the payload. resolved=9 / answered=0 is a healthy mapping and
      // broken executors — a distinction a single "failed" count would hide.
      <Tooltip title="Services whose node ran the probe payload">
        <Box component="span">{original.counts.answered_services}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (row) => row.counts.orphaned_services,
    id: 'orphaned',
    header: 'Orphaned',
    Cell: ({ row: { original } }) => (
      <Tooltip title="Services with no live executor host — not an error">
        <Box component="span">{original.counts.orphaned_services}</Box>
      </Tooltip>
    ),
  },
];

/**
 * Ask for a refresh, and say honestly what came back.
 *
 * Separate from `SyncButton`, which rebuilds pmm-managed's snapshot in about a tenth
 * of a second and never touches a host. This queues a job per host and takes tens of
 * seconds, so the app answers `running` and the list below polls until it lands. Two
 * triggers that different must not look alike, which is most of why they live on
 * different pages.
 */
function RefreshButton() {
  const { data: runs } = useOmInventoryRuns();
  const trigger = useRefreshInventory();
  // Any active run, not just the newest. Refreshes are host-scoped, so two can overlap
  // and a narrow one started later can reach a terminal status while a broader one is
  // still probing - `runs[0]` would re-enable this button into a sweep that must
  // conflict with it. Same reason `useOmInventoryRuns` polls on the whole collection.
  const running = (runs ?? []).some((run) => isRunActive(run.status));
  // The rows this page's sibling tabs render change when the refresh finishes, not when
  // it is accepted, so the estate is invalidated on that edge rather than on the mutation.
  useInvalidateEstateOnRefreshEnd(runs);
  // A 409 is an expected answer rather than a failure, and since conflict is judged
  // per host the message names what is in flight instead of saying "a sweep is
  // already running" - which was true of anything and useful for nothing.
  const conflict =
    trigger.error instanceof OmApiError && trigger.error.status === 409
      ? trigger.error
      : null;
  const failure = trigger.error && !conflict ? trigger.error : null;

  return (
    <Stack direction="row" alignItems="center" gap={1}>
      <Tooltip title="Probe every host in the estate and collect what no metric carries">
        <span>
          <Button
            variant="contained"
            startIcon={
              running ? <CircularProgress size={16} /> : <PlayArrowIcon />
            }
            disabled={running || trigger.isPending}
            onClick={() => trigger.mutate(undefined)}
          >
            {running ? 'Refreshing…' : 'Refresh estate'}
          </Button>
        </span>
      </Tooltip>
      {conflict && (
        <Typography variant="body2" color="text.secondary">
          {conflict.message}
        </Typography>
      )}
      {failure && (
        <Typography variant="body2" color="error">
          Could not start a refresh: {failure.message}
        </Typography>
      )}
    </Stack>
  );
}

/**
 * What OM currently knows, from the newest run.
 *
 * The table answers "has this been working". This answers "what does OM know right
 * now", which is the more common question and otherwise needs reading the first row
 * of a table and knowing that the first row is the newest.
 *
 * A run stuck in `running` is a real state with a reaper behind it, so its age is
 * shown: that is what distinguishes a refresh that is working from one that is
 * wedged.
 */
function LastRun({ run }: { run: OmInventoryRun | undefined }) {
  if (!run) {
    return (
      <Alert severity="info">
        No refresh has run yet. OM has nothing to show until one does.
      </Alert>
    );
  }
  const age = ageSeconds(run.start_time);
  const active = isRunActive(run.status);
  return (
    <Stack
      direction="row"
      gap={3}
      alignItems="center"
      sx={{ flexWrap: 'wrap' }}
    >
      <RunStatusBadge status={run.status} />
      <Typography variant="body2" color="text.secondary">
        {active
          ? `started ${age == null ? 'just now' : `${formatCompactDuration(age)} ago`}`
          : `${age == null ? '' : `${formatCompactDuration(age)} ago`}, took ${
              formatRunDuration(run.start_time, run.end_time) || '—'
            }`}
      </Typography>
      <Typography variant="body2">
        <strong>{run.counts.answered_hosts}</strong> of{' '}
        {run.counts.probeable_hosts} hosts answered
      </Typography>
      <Typography variant="body2">
        <strong>{run.counts.answered_services}</strong> of{' '}
        {run.counts.resolved_services} services answered
      </Typography>
      {run.scope.length > 0 && (
        <Tooltip title={run.scope.join(', ')}>
          <Typography variant="body2" color="warning.main">
            scoped to {run.scope.length} host
            {run.scope.length === 1 ? '' : 's'}
          </Typography>
        </Tooltip>
      )}
      {run.error && (
        <Typography variant="body2" color="error">
          {run.error}
        </Typography>
      )}
    </Stack>
  );
}

/**
 * OM's refresh history, and the schedule that drives it.
 *
 * These are the app's refreshes, not pmm-managed's collection pass: one runs a payload
 * on every host over Nomad and takes tens of seconds, the other recomputes a document
 * from data PMM already holds. They are two different things called a "run", which is
 * why they live at two different paths and on two different pages.
 *
 * Read through pmm-managed rather than from SEP directly, which is what lets this page
 * render its own error when SEP is unwell instead of being blanked by a gate that
 * fails closed.
 */
/** The tabs, and the query-parameter values that address them. */
const TABS = ['runs', 'settings'] as const;
type TabId = (typeof TABS)[number];

/**
 * OM's refresh history, and the schedule that drives it.
 *
 * These are the app's refreshes, not pmm-managed's collection pass: one runs a payload
 * on every host over Nomad and takes tens of seconds, the other recomputes a document
 * from data PMM already holds. They are two different things called a "run", which is
 * why they live at two different paths and on two different pages.
 *
 * Read through pmm-managed rather than from SEP directly, which is what lets this page
 * render its own error when SEP is unwell instead of being blanked by a gate that
 * fails closed.
 *
 * The two halves are tabs rather than one column because they answer different
 * questions on different clocks: "did the last refresh work" is asked often and
 * skimmed, "how often should it run" is asked rarely and read carefully. Stacked, the
 * second sat below a table of twenty-five rows and was found by scrolling.
 */
export function InventoryPage() {
  const { data: runs, isLoading, error } = useOmInventoryRuns();
  const rows = useMemo(() => runs ?? [], [runs]);
  // In the query string rather than component state, so a link to the settings tab is
  // shareable and a reload does not silently put the reader back on Runs.
  const [params, setParams] = useSearchParams();
  const requested = params.get('tab');
  const tab: TabId = TABS.includes(requested as TabId)
    ? (requested as TabId)
    : 'runs';

  return (
    <Stack gap={2}>
      <OmHeader
        title="Inventory"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every refresh probes each host for what no metric carries, and
            stores it against the estate.
          </Typography>
        }
        // Stays in the header rather than inside the Runs tab: it is the page's
        // action, and hiding it while someone reads the schedule would mean going
        // back a tab to act on what they just changed.
        actions={<RefreshButton />}
      />

      <Tabs
        value={tab}
        onChange={(_event, next: TabId) =>
          setParams((current) => {
            const updated = new URLSearchParams(current);
            updated.set('tab', next);
            return updated;
          })
        }
      >
        <Tab value="runs" label="Runs" />
        <Tab value="settings" label="Settings" />
      </Tabs>

      {tab === 'runs' ? (
        <>
          {error && (
            <Alert severity="error">
              {/* Rendered inside the page rather than replacing it: SEP being
                  unwell is a fact about the estate, and the settings tab still
                  reads. */}
              Could not load refreshes: {(error as Error).message}
            </Alert>
          )}

          {/* Only once the query has actually answered. LastRun reads an absent run as
              "no refresh has run yet", which is a claim about the estate - not something
              to assert while the first request is still in flight or has failed with no
              cached rows to fall back on. */}
          {(!isLoading || runs) && !error && <LastRun run={rows[0]} />}

          {isLoading && !runs ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
              <CircularProgress />
            </Box>
          ) : (
            <Table
              tableName="om-inventory-runs"
              columns={RUN_COLUMNS}
              data={rows}
              getRowId={(row) => row.run_id}
              enableGlobalFilter={false}
              enableColumnFilters={false}
              enableHiding={false}
              enablePagination={false}
              enableStickyHeader
              enableExpanding
              renderDetailPanel={({ row }) => (
                <RunEntities run={row.original} />
              )}
            />
          )}
        </>
      ) : (
        <ConfigForm />
      )}
    </Stack>
  );
}
