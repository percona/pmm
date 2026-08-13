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
  LinearProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import {
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
} from 'material-react-table';
import { PROCESS_ROLE_LABEL } from './constants';
import { PomHeader } from './components/PomHeader';
import { SnapshotBar } from './components/SnapshotBar';
import { StatusBadge } from './components/HealthBadge';
import { SyncButton } from './components/SyncButton';
import { Duration, Percent } from './components/Metric';
import { Unavailable } from './components/Unavailable';
import { toServiceRows, usePomTopology } from './hooks';
import type { PomServiceRow } from './types';

/** A full `mongod` command line is a paragraph; the cell shows it on hover. */
const TRUNCATED = {
  display: 'block',
  maxWidth: 320,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
} as const;

/**
 * Columns the table carries but does not open with.
 *
 * The page's job is to show everything the snapshot stores, which is not the same as
 * showing it all at once: these five are either constant across a PSMDB estate
 * (`edition`), internal identifiers, or long enough to push the load columns off
 * screen. They stay one click away in the column-visibility menu.
 */
const HIDDEN_BY_DEFAULT = {
  service_id: false,
  service_type: false,
  edition: false,
  config_path: false,
  argv: false,
};

/** Columns for the flat service table. Grouping keys lead, then identity, then load. */
function useColumns(): MRT_ColumnDef<PomServiceRow>[] {
  return useMemo(
    () => [
      {
        accessorKey: 'env_name',
        header: 'Environment',
        Cell: ({ row: { original } }) =>
          original.env_name ?? <Unavailable reason="not_applicable" />,
      },
      {
        accessorKey: 'cluster_name',
        header: 'Cluster',
        Cell: ({ row: { original } }) =>
          original.cluster_name ?? <Unavailable reason="not_applicable" />,
      },
      { accessorKey: 'service_name', header: 'Service' },
      {
        accessorKey: 'host',
        header: 'Host',
        Cell: ({ row: { original } }) =>
          original.host ?? <Unavailable reason="service_not_observed" />,
      },
      {
        accessorKey: 'replication_set',
        header: 'Replication set',
        Cell: ({ row: { original } }) =>
          // A standalone belongs to no set, which is not a gap in what we saw.
          original.replication_set ?? <Unavailable reason="not_applicable" />,
      },
      {
        accessorKey: 'status',
        header: 'Status',
        Cell: ({ row: { original } }) => (
          <StatusBadge status={original.status} />
        ),
      },
      {
        accessorKey: 'process_role',
        header: 'Role',
        Cell: ({ row: { original } }) =>
          PROCESS_ROLE_LABEL[original.process_role] ?? original.process_role,
      },
      {
        accessorKey: 'state',
        header: 'Member state',
        Cell: ({ row: { original } }) =>
          // A router and a standalone are not replica-set members at all.
          original.state ?? <Unavailable reason="not_applicable" />,
      },
      {
        accessorKey: 'version',
        header: 'Version',
        Cell: ({ row: { original } }) =>
          original.version ?? <Unavailable reason="service_not_observed" />,
      },
      {
        accessorKey: 'installed_version',
        header: 'Installed',
        // Reads "not collected" rather than "not observed": unlike every other
        // column this one needs an on-host probe, so its absence says the discovery
        // app has not run here -- not that the service is unreachable.
        Cell: ({ row: { original } }) =>
          original.installed_version ?? (
            <Unavailable reason="metric_not_collected" />
          ),
      },
      {
        accessorKey: 'vendor',
        header: 'Vendor',
        Cell: ({ row: { original } }) =>
          original.vendor ?? <Unavailable reason="service_not_observed" />,
      },
      {
        accessorKey: 'endpoint',
        header: 'Endpoint',
        Cell: ({ row: { original } }) =>
          original.endpoint ?? <Unavailable reason="service_not_observed" />,
      },
      {
        accessorKey: 'cpu_usage_percent',
        header: 'CPU',
        Cell: ({ row: { original } }) => (
          <Percent value={original.cpu_usage_percent} />
        ),
      },
      {
        accessorKey: 'connections_free_percent',
        header: 'Conn. free',
        Cell: ({ row: { original } }) => (
          <Percent value={original.connections_free_percent} />
        ),
      },
      {
        accessorKey: 'replication_lag_seconds',
        header: 'Repl. lag',
        Cell: ({ row: { original } }) => (
          <Duration value={original.replication_lag_seconds} />
        ),
      },
      {
        accessorKey: 'oplog_window_seconds',
        header: 'Oplog window',
        Cell: ({ row: { original } }) => (
          <Duration value={original.oplog_window_seconds} />
        ),
      },
      {
        accessorKey: 'service_id',
        header: 'Service ID',
        Cell: ({ row: { original } }) =>
          original.service_id ?? <Unavailable reason="service_not_observed" />,
      },
      {
        accessorKey: 'service_type',
        header: 'Service type',
        Cell: ({ row: { original } }) =>
          original.service_type ?? (
            <Unavailable reason="service_not_observed" />
          ),
      },
      {
        accessorKey: 'edition',
        header: 'Edition',
        Cell: ({ row: { original } }) =>
          original.edition ?? <Unavailable reason="service_not_observed" />,
      },
      {
        accessorKey: 'config_path',
        header: 'Config path',
        // Probe-only, like `installed_version`: its absence says the discovery app
        // has not run on this host, not that the service is unreachable.
        Cell: ({ row: { original } }) =>
          original.config_path ?? <Unavailable reason="metric_not_collected" />,
      },
      {
        accessorKey: 'argv',
        header: 'Command line',
        Cell: ({ row: { original } }) =>
          original.argv ? (
            <Tooltip title={original.argv}>
              <Box component="span" sx={TRUNCATED}>
                {original.argv}
              </Box>
            </Tooltip>
          ) : (
            <Unavailable reason="metric_not_collected" />
          ),
      },
    ],
    []
  );
}

/** Counts above the table, so the headline numbers need no reading of rows. */
function Counts({
  total,
  up,
  down,
}: {
  total: number;
  up: number;
  down: number;
}) {
  return (
    <Stack direction="row" spacing={3} sx={{ mb: 2 }}>
      <Typography variant="body2">
        <strong>{total}</strong> services
      </Typography>
      <Typography variant="body2" color="success.main">
        <strong>{up}</strong> up
      </Typography>
      <Typography
        variant="body2"
        color={down ? 'error.main' : 'text.secondary'}
      >
        <strong>{down}</strong> down
      </Typography>
    </Stack>
  );
}

/**
 * The estate as one table, one row per service.
 *
 * The document is a nested `environments -> clusters -> services` tree, but it renders
 * flat: sorting and filtering are only useful across the whole estate, and the nesting
 * survives as the two leading columns. Grouping is available on them if a reader wants
 * the tree back, and Overview is the same snapshot already read that way.
 */
export function TopologyPage() {
  const { data, isPending, isError, error } = usePomTopology();
  const columns = useColumns();
  const rows = useMemo(() => toServiceRows(data), [data]);

  const table = useMaterialReactTable({
    columns,
    data: rows,
    enableGrouping: true,
    enablePagination: false,
    enableDensityToggle: false,
    initialState: {
      density: 'compact',
      columnVisibility: HIDDEN_BY_DEFAULT,
      sorting: [
        { id: 'cluster_name', desc: false },
        { id: 'service_name', desc: false },
      ],
    },
  });

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    return (
      <Alert severity="error">
        {/* A 503 here is the expected first-run state, not a fault: the API says so
            when no discovery has completed, and Sync is the way out. */}
        {(error as Error)?.message ?? 'Could not load the topology.'}
      </Alert>
    );
  }

  return (
    <Box>
      <PomHeader
        title="Topology"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every monitored MongoDB service, and every field the snapshot stores
            about it.
          </Typography>
        }
        actions={<SyncButton />}
      />
      <SnapshotBar envelope={data.snapshot} />
      <Counts
        total={data.summary.services_total}
        up={data.summary.services_up}
        down={data.summary.services_down}
      />
      <MaterialReactTable table={table} />
    </Box>
  );
}
