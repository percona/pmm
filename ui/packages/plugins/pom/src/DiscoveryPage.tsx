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
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import { Table, type MRT_ColumnDef } from '@percona/percona-ui';
import {
  isRefreshActive,
  usePomInventoryRuns,
  useRefreshInventory,
} from './inventoryHooks';
import { PomApiError } from './hooks';
import { ConfigForm } from './components/ConfigForm';
import { RunStatusBadge } from './components/HealthBadge';
import { RunEntities } from './components/RunEntities';
import { PomHeader } from './components/PomHeader';
import { formatDuration, formatRunDuration, formatTimestamp } from './format';
import { ageSeconds } from './inventory';
import type { PomInventoryRun } from './types';

const RUN_COLUMNS: MRT_ColumnDef<PomInventoryRun>[] = [
  {
    accessorKey: 'status',
    header: 'Status',
    Cell: ({ row: { original } }) => (
      <RunStatusBadge status={original.status} />
    ),
  },
  {
    accessorKey: 'started_at',
    header: 'Started',
    Cell: ({ row: { original } }) => formatTimestamp(original.started_at),
  },
  {
    accessorFn: (row) => formatRunDuration(row.started_at, row.finished_at),
    id: 'duration',
    header: 'Duration',
    Cell: ({ row: { original } }) =>
      formatRunDuration(original.started_at, original.finished_at) || '—',
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
    accessorFn: (row) => row.counts.hosts_answered,
    id: 'hosts',
    header: 'Hosts',
    Cell: ({ row: { original } }) => (
      // total / probeable / answered in one cell. The gap between the first two is
      // the estate nothing can be dispatched to, which is an onboarding fact rather
      // than a failed run, and the gap between the last two is what actually failed.
      <Tooltip
        title={`${original.counts.hosts_total} in scope, ${original.counts.hosts_probeable} with somewhere to run a probe, ${original.counts.hosts_answered} answered`}
      >
        <Box component="span">
          {original.counts.hosts_answered}/{original.counts.hosts_probeable}
          {original.counts.hosts_probeable === original.counts.hosts_total
            ? ''
            : ` of ${original.counts.hosts_total}`}
        </Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (row) => row.counts.services_total,
    id: 'services',
    header: 'Services',
    Cell: ({ row: { original } }) => original.counts.services_total,
  },
  {
    accessorFn: (row) => row.counts.services_resolved,
    id: 'resolved',
    header: 'Resolved',
    Cell: ({ row: { original } }) => (
      <Tooltip title="Services that mapped to a live executor host">
        <Box component="span">{original.counts.services_resolved}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (row) => row.counts.services_answered,
    id: 'answered',
    header: 'Answered',
    Cell: ({ row: { original } }) => (
      // The diagnostic pair: resolved says the mapping worked, answered says the
      // node ran the payload. resolved=9 / answered=0 is a healthy mapping and
      // broken executors — a distinction a single "failed" count would hide.
      <Tooltip title="Services whose node ran the probe payload">
        <Box component="span">{original.counts.services_answered}</Box>
      </Tooltip>
    ),
  },
  {
    accessorFn: (row) => row.counts.services_orphaned,
    id: 'orphaned',
    header: 'Orphaned',
    Cell: ({ row: { original } }) => (
      <Tooltip title="Services with no live executor host — not an error">
        <Box component="span">{original.counts.services_orphaned}</Box>
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
  const { data: runs } = usePomInventoryRuns();
  const trigger = useRefreshInventory();
  const running = isRefreshActive(runs?.[0]?.status);
  // A 409 is an expected answer rather than a failure, and since conflict is judged
  // per host the message names what is in flight instead of saying "a sweep is
  // already running" - which was true of anything and useful for nothing.
  const conflict =
    trigger.error instanceof PomApiError && trigger.error.status === 409
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
 * What POM currently knows, from the newest run.
 *
 * The table answers "has this been working". This answers "what does POM know right
 * now", which is the more common question and otherwise needs reading the first row
 * of a table and knowing that the first row is the newest.
 *
 * A run stuck in `running` is a real state with a reaper behind it, so its age is
 * shown: that is what distinguishes a refresh that is working from one that is
 * wedged.
 */
function LastRun({ run }: { run: PomInventoryRun | undefined }) {
  if (!run) {
    return (
      <Alert severity="info">
        No refresh has run yet. POM has nothing to show until one does.
      </Alert>
    );
  }
  const age = ageSeconds(run.started_at);
  const active = isRefreshActive(run.status);
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
          ? `started ${age == null ? 'just now' : `${formatDuration(age)} ago`}`
          : `${age == null ? '' : `${formatDuration(age)} ago`}, took ${
              formatRunDuration(run.started_at, run.finished_at) || '—'
            }`}
      </Typography>
      <Typography variant="body2">
        <strong>{run.counts.hosts_answered}</strong> of{' '}
        {run.counts.hosts_probeable} hosts answered
      </Typography>
      <Typography variant="body2">
        <strong>{run.counts.services_answered}</strong> of{' '}
        {run.counts.services_resolved} services answered
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
 * POM's refresh history, and the schedule that drives it.
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
export function DiscoveryPage() {
  const { data: runs, isLoading, error } = usePomInventoryRuns();
  const rows = useMemo(() => runs ?? [], [runs]);

  return (
    <Stack gap={2}>
      <PomHeader
        title="Discovery"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every refresh probes each host for what no metric carries, and
            stores it against the estate.
          </Typography>
        }
        actions={<RefreshButton />}
      />

      {error && (
        <Alert severity="error">
          {/* Rendered inside the page rather than replacing it: SEP being unwell is
              a fact about the estate, and the schedule below is still readable. */}
          Could not load refreshes: {(error as Error).message}
        </Alert>
      )}

      <LastRun run={rows[0]} />

      {isLoading && !runs ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      ) : (
        <Table
          tableName="pom-inventory-runs"
          columns={RUN_COLUMNS}
          data={rows}
          getRowId={(row) => row.run_id}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableHiding={false}
          enablePagination={false}
          enableStickyHeader
          enableExpanding
          renderDetailPanel={({ row }) => <RunEntities run={row.original} />}
        />
      )}

      <ConfigForm />
    </Stack>
  );
}
