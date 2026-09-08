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
  Chip,
  CircularProgress,
  Stack,
  Typography,
} from '@mui/material';
import { Table, type MRT_ColumnDef } from '@percona/percona-ui';
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

const RUN_COLUMNS: MRT_ColumnDef<OmGetBootstrapRunResponse>[] = [
  {
    accessorKey: 'status',
    header: 'Status',
    Cell: ({ row: { original } }) => (
      <Chip
        size="small"
        label={BOOTSTRAP_RUN_LABEL[original.status] ?? original.status}
        color={BOOTSTRAP_RUN_COLOR[original.status] ?? 'default'}
      />
    ),
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
 * each expandable in place to the same step-by-step progress the wizard shows
 * live.
 *
 * Exists so a run started from {@link BootstrapWizardDialog} is never lost by
 * closing the dialog -- the wizard is a launcher, not the only place a run's
 * progress can be read back.
 */
export const OperationsPage = () => {
  const { data: runs, isLoading, error } = useOmBootstrapRuns();
  const rows = useMemo(() => runs ?? [], [runs]);

  return (
    <Stack gap={2}>
      <OmHeader
        title="Operations"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every bootstrap run this server has driven, newest first. Expand a
            row to watch it live or review what happened.
          </Typography>
        }
      />

      {error && (
        <Alert severity="error">
          Could not load bootstrap runs: {(error as Error).message}
        </Alert>
      )}

      {isLoading && !runs ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      ) : rows.length === 0 && !error ? (
        <Alert severity="info">No bootstrap runs yet.</Alert>
      ) : (
        <Table
          tableName="om-operations-runs"
          columns={RUN_COLUMNS}
          data={rows}
          getRowId={(row) => row.run_id}
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableHiding={false}
          enablePagination={false}
          enableStickyHeader
          enableExpanding
          renderDetailPanel={({ row }) => <RunProgress run={row.original} />}
        />
      )}
    </Stack>
  );
};
