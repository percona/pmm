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
  isProbeConflict,
  isProbeRunActive,
  usePomProbeRuns,
  useTriggerPomProbe,
} from './probeHooks';
import { RunStatusBadge } from './components/HealthBadge';
import { RunNodes } from './components/RunNodes';
import { PomHeader } from './components/PomHeader';
import { formatRunDuration, formatTimestamp } from './format';
import type { PomProbeRun } from './types';

const RUN_COLUMNS: MRT_ColumnDef<PomProbeRun>[] = [
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
  {
    accessorKey: 'facts_collected',
    header: 'Facts',
    Cell: ({ row: { original } }) => (
      <Tooltip title="On-host facts stored by this sweep, merged into the topology document">
        <Box component="span">{original.facts_collected}</Box>
      </Tooltip>
    ),
  },
];

/**
 * Trigger a probe sweep and reflect its progress.
 *
 * Separate from `SyncButton`, which rebuilds pmm-managed's topology document in a
 * tenth of a second. This queues work on database hosts that takes tens of seconds,
 * so the app answers `202` and the list below polls until it lands.
 */
function RunProbeButton() {
  const { data: runs } = usePomProbeRuns();
  const trigger = useTriggerPomProbe();
  const running = isProbeRunActive(runs?.[0]?.status);
  const conflict = isProbeConflict(trigger.error);
  const failure = trigger.error && !conflict ? trigger.error : null;

  return (
    <Stack direction="row" alignItems="center" gap={1}>
      <Tooltip title="Probe every database host and collect the facts no metric carries">
        <span>
          <Button
            variant="contained"
            startIcon={
              running ? <CircularProgress size={16} /> : <PlayArrowIcon />
            }
            disabled={running || trigger.isPending}
            onClick={() => trigger.mutate()}
          >
            {running ? 'Running…' : 'Run discovery'}
          </Button>
        </span>
      </Tooltip>
      {conflict && (
        <Typography variant="body2" color="text.secondary">
          A sweep is already in flight.
        </Typography>
      )}
      {failure && (
        <Typography variant="body2" color="error">
          Could not start a sweep: {failure.message}
        </Typography>
      )}
    </Stack>
  );
}

/**
 * The `pom_discovery` sweep history.
 *
 * These are SEP's runs, not pmm-managed's: a sweep runs a payload on every database
 * host over Nomad and stores the on-host facts — installed version, config path,
 * command line — that pmm-managed then merges into the topology document. The list
 * polls itself while the newest sweep is active, which is what makes the trigger
 * button's progress visible here without extra wiring.
 */
export function RunsPage() {
  const { data: runs, isLoading, error } = usePomProbeRuns();
  const rows = useMemo(() => runs ?? [], [runs]);

  return (
    <Stack gap={2}>
      <PomHeader
        title="Discovery"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every sweep probes each database host for the facts no metric
            carries, and POM merges them into the topology.
          </Typography>
        }
        actions={<RunProbeButton />}
      />

      {error && (
        <Alert severity="error">
          {/* SEP owns this half of POM, so its being unreachable is the answer
              here rather than an empty table. */}
          Could not load sweeps from SEP: {(error as Error).message}
        </Alert>
      )}

      {isLoading && !runs ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      ) : (
        <Table
          tableName="pom-probe-runs"
          columns={RUN_COLUMNS}
          data={rows}
          getRowId={(row) => row.run_id}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableHiding={false}
          enablePagination={false}
          enableStickyHeader
          enableExpanding
          renderDetailPanel={({ row }) => <RunNodes run={row.original} />}
        />
      )}
    </Stack>
  );
}
