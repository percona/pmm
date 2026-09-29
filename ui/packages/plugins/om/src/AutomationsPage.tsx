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
  Chip,
  CircularProgress,
  Stack,
  Typography,
} from '@mui/material';
import { Table, type MRT_ColumnDef } from '@percona/percona-ui';
import { bootstrapRunDisplayStatus } from './api';
import { BOOTSTRAP_RUN_COLOR, BOOTSTRAP_RUN_LABEL } from './constants';
import { OmHeader } from './components/OmHeader';
import { RunProgress } from './components/RunProgress';
import {
  formatRunDuration,
  formatTimestamp,
  runDurationSeconds,
} from './format';
import { useOmBootstrapRuns } from './inventoryHooks';
import type { OmGetBootstrapRunResponse } from './types';

/**
 * How many runs to ask for.
 *
 * Asked for explicitly because this page has no pagination: left unset, the
 * server's own default of 20 would silently cut the history off, and 100 is the
 * most it will serve (ListInventoryRunsRequest.limit is validated lte: 100, and
 * pmm-managed's maxInventoryRunLimit caps it again).
 */
const RUN_HISTORY_LIMIT = 100;

const RUN_COLUMNS: MRT_ColumnDef<OmGetBootstrapRunResponse>[] = [
  {
    accessorFn: (row) => bootstrapRunDisplayStatus(row),
    id: 'status',
    header: 'Status',
    Cell: ({ row: { original } }) => {
      const status = bootstrapRunDisplayStatus(original);
      return (
        <Chip
          size="small"
          label={BOOTSTRAP_RUN_LABEL[status] ?? status}
          color={BOOTSTRAP_RUN_COLOR[status] ?? 'default'}
        />
      );
    },
  },
  {
    accessorKey: 'replica_set_name',
    header: 'Replica set',
  },
  {
    accessorKey: 'mongodb_version',
    header: 'MongoDB version',
  },
  {
    accessorFn: (row) => row.environment || '—',
    id: 'environment',
    header: 'Environment',
  },
  {
    accessorFn: (row) => row.cluster || '—',
    id: 'cluster',
    header: 'Cluster',
  },
  {
    accessorFn: (row) => row.hosts.length,
    id: 'hosts',
    header: 'Hosts',
  },
  {
    accessorKey: 'started_at',
    header: 'Started',
    Cell: ({ row: { original } }) => formatTimestamp(original.started_at),
  },
  {
    // Sorts on elapsed seconds, not the formatted string -- see InventoryPage's own
    // duration column for why.
    accessorFn: (row) => runDurationSeconds(row.started_at, row.finished_at),
    id: 'duration',
    header: 'Duration',
    Cell: ({ row: { original } }) =>
      formatRunDuration(original.started_at, original.finished_at),
  },
];

/**
 * Bootstrap run history: every run this PMM server has driven, newest first,
 * each expandable in place to the same step-by-step progress {@link BootstrapPage}
 * shows live.
 *
 * Exists so a run started from {@link BootstrapPage} is never lost by
 * navigating away -- starting a run is one thing, and reading its progress
 * back later is another that has to survive leaving the page it started on.
 *
 * `?expand=<run_id>` unfolds that run's row on landing -- BootstrapPage
 * navigates here with it set the moment a run is accepted, so triggering a
 * bootstrap goes straight to watching it live rather than to a page that
 * still requires an extra click to find the run just started.
 */
export const AutomationsPage = () => {
  const {
    data: runs,
    isLoading,
    error,
  } = useOmBootstrapRuns(RUN_HISTORY_LIMIT);
  const rows = useMemo(() => runs ?? [], [runs]);
  const [params] = useSearchParams();
  const expandRunId = params.get('expand');

  if (isLoading && !runs) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
        <CircularProgress />
      </Box>
    );
  }

  // Returns rather than rendering above the table: a failed load has no rows to
  // show, and an empty table under the error reads as "there are no runs", which
  // is a different thing from "we could not find out". Same shape as HostsPage's
  // own isError branch.
  if (error) {
    return (
      <Alert severity="error">
        Could not load bootstrap runs: {(error as Error).message}
      </Alert>
    );
  }

  return (
    <Stack gap={2}>
      <OmHeader
        title="Automations"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every bootstrap run this server has driven, newest first. Expand a
            row to watch it live or review what happened.
          </Typography>
        }
      />

      {rows.length === 0 ? (
        <Alert severity="info">No bootstrap runs yet.</Alert>
      ) : (
        <Table
          tableName="om-automations-runs"
          columns={RUN_COLUMNS}
          data={rows}
          getRowId={(row) => row.run_id}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableHiding={false}
          enablePagination={false}
          enableStickyHeader
          enableExpanding
          initialState={
            expandRunId ? { expanded: { [expandRunId]: true } } : undefined
          }
          renderDetailPanel={({ row }) => <RunProgress run={row.original} />}
        />
      )}
    </Stack>
  );
};
