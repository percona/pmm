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

import { useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Chip,
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
import { OmHeader } from './components/OmHeader';
import { SnapshotBar } from './components/SnapshotBar';
import { StatusBadge } from './components/HealthBadge';
import { SyncButton } from './components/SyncButton';
import { Duration, Percent } from './components/Metric';
import { Unavailable } from './components/Unavailable';
import { toServiceRows, useOmTopology } from './hooks';
import { useOmInventoryServices } from './inventoryHooks';
import { ageSeconds, isFailing, joinServiceInventory } from './inventory';
import { formatDuration } from './format';
import { ProbeValue } from './components/ProbeValue';
import type { OmInventoryService, OmServiceInventoryRow } from './types';

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
  node_id: false,
};

/** Columns for the flat service table. Grouping keys lead, then identity, then load. */
function useColumns(): MRT_ColumnDef<OmServiceInventoryRow>[] {
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
        // Read off the estate rather than the snapshot: the merge that used to put it
        // there is gone, and this is now first-hand.
        //
        // Deliberately beside `version` and never merged with it. That column is what
        // the running mongod reports over the wire; this is what the package database
        // on the host says. They disagree exactly when a package has been upgraded and
        // the process not restarted, which is a state OM exists to find - and one
        // column showing "whichever we have" could not express it.
        id: 'installed_version',
        accessorFn: (row) => row.inventory?.installed_version ?? null,
        header: 'Installed',
        Cell: ({ row: { original } }) => (
          <ProbeValue
            inventory={original.inventory}
            value={original.inventory?.installed_version}
          />
        ),
      },
      {
        id: 'probe_status',
        accessorFn: (row) => row.inventory?.probe_status ?? null,
        header: 'Probe',
        Cell: ({ row: { original } }) =>
          original.inventory ? (
            <ProbeStatus inventory={original.inventory} />
          ) : (
            <Unavailable reason="not_in_inventory" />
          ),
      },
      {
        id: 'last_success_at',
        // Sorted on the age in seconds, not the timestamp string: the column is read
        // as "how stale", and a lexicographic sort of ISO strings puts a row that has
        // never answered next to the oldest one rather than at the end.
        accessorFn: (row) =>
          ageSeconds(row.inventory?.freshness.last_success_at) ?? Infinity,
        header: 'Collected',
        Cell: ({ row: { original } }) => {
          if (!original.inventory) {
            return <Unavailable reason="not_in_inventory" />;
          }
          const age = ageSeconds(original.inventory.freshness.last_success_at);
          return age == null ? (
            <Unavailable reason="probe_never_succeeded" />
          ) : (
            <>{formatDuration(age)} ago</>
          );
        },
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
        id: 'config_path',
        accessorFn: (row) => row.inventory?.config_path ?? null,
        header: 'Config path',
        Cell: ({ row: { original } }) => (
          <ProbeValue
            inventory={original.inventory}
            value={original.inventory?.config_path}
          />
        ),
      },
      {
        id: 'node_id',
        accessorFn: (row) => row.inventory?.node_id ?? null,
        header: 'Node ID',
        // The link to the Hosts page, and the key OM's estate is built on. Hidden by
        // default like the other identifiers, but it is what makes "which host is this
        // on" answerable without a second lookup.
        Cell: ({ row: { original } }) =>
          original.inventory?.node_id ?? (
            <Unavailable reason="not_in_inventory" />
          ),
      },
      {
        id: 'argv',
        accessorFn: (row) => row.inventory?.argv ?? null,
        header: 'Command line',
        Cell: ({ row: { original } }) =>
          original.inventory?.argv ? (
            <Tooltip title={original.inventory.argv}>
              <Box component="span" sx={TRUNCATED}>
                {original.inventory.argv}
              </Box>
            </Tooltip>
          ) : (
            <ProbeValue inventory={original.inventory} value={null} />
          ),
      },
    ],
    []
  );
}

/**
 * Counts above the table, so the headline numbers need no reading of rows.
 *
 * `failing` counts probes, not services: a service can be UP to PMM and failing its
 * probe, which is the pair of facts this page exists to put side by side. It doubles
 * as the filter, because a count nobody can act on is decoration.
 */
function Counts({
  total,
  up,
  down,
  failing,
  failingOnly,
  onToggleFailing,
}: {
  total: number;
  up: number;
  down: number;
  failing: number;
  failingOnly: boolean;
  onToggleFailing: () => void;
}) {
  return (
    <Stack direction="row" spacing={3} sx={{ mb: 2, alignItems: 'center' }}>
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
      {failing > 0 || failingOnly ? (
        <Chip
          size="small"
          color={failingOnly ? 'error' : 'default'}
          variant={failingOnly ? 'filled' : 'outlined'}
          label={`${failing} failing a probe`}
          onClick={onToggleFailing}
        />
      ) : null}
    </Stack>
  );
}

/**
 * A service's probe outcome, as a word rather than a status string.
 *
 * Failing is shown with how long it has been failing, because that is the difference
 * between a blip and something to act on: a table of fifteen rows where one has been
 * failing for three days looks identical to a healthy one otherwise.
 */
function ProbeStatus({ inventory }: { inventory: OmInventoryService }) {
  const since = ageSeconds(inventory.freshness.failing_since);
  if (since == null) {
    return <>{inventory.probe_status ?? 'ok'}</>;
  }
  return (
    <Tooltip
      title={
        inventory.freshness.last_error ??
        'The last probe against this service failed.'
      }
    >
      <Box component="span" sx={{ color: 'error.main', cursor: 'help' }}>
        failing {formatDuration(since)}
        {inventory.freshness.consecutive_failures > 0
          ? ` (${inventory.freshness.consecutive_failures}x)`
          : ''}
      </Box>
    </Tooltip>
  );
}

/**
 * Every monitored service, with what PMM sees and what the probe found.
 *
 * Two sources joined on PMM's service id, which costs nothing: the estate is keyed on
 * the same id, so there is no matching rule and nothing to get wrong. Rows come from
 * the snapshot, so a service PMM registered since the last sweep still appears - with
 * its probe columns saying why they are empty rather than the row being missing.
 *
 * The document is a nested `environments -> clusters -> services` tree, but it renders
 * flat: sorting and filtering are only useful across the whole estate, and the nesting
 * survives as the two leading columns. Grouping is available on them if a reader wants
 * the tree back, and Overview is the same snapshot already read that way.
 */
export function ServicesPage() {
  const { data, isPending, isError, error } = useOmTopology();
  // Deliberately not gated on the estate loading or failing. The snapshot is PMM's own
  // and always available; the estate is a second service that may be unwell, and a
  // page that blanked when it was would be exactly what proxying it through
  // pmm-managed was meant to stop.
  const { data: inventory } = useOmInventoryServices();
  const [failingOnly, setFailingOnly] = useState(false);
  const columns = useColumns();
  const joined = useMemo(
    () => joinServiceInventory(toServiceRows(data), inventory),
    [data, inventory]
  );
  const rows = useMemo(
    () =>
      failingOnly
        ? joined.filter((row) => row.inventory && isFailing(row.inventory))
        : joined,
    [joined, failingOnly]
  );
  const failingCount = useMemo(
    () =>
      joined.filter((row) => row.inventory && isFailing(row.inventory)).length,
    [joined]
  );

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
      <OmHeader
        title="Services"
        subtitle={
          <Typography variant="body2" color="text.secondary">
            Every monitored MongoDB service: what PMM sees over the wire, and
            what OM&apos;s probe found on the host.
          </Typography>
        }
        actions={<SyncButton />}
      />
      <SnapshotBar envelope={data.snapshot} />
      <Counts
        total={data.summary.services_total}
        up={data.summary.services_up}
        down={data.summary.services_down}
        failing={failingCount}
        failingOnly={failingOnly}
        onToggleFailing={() => setFailingOnly((on) => !on)}
      />
      <MaterialReactTable table={table} />
    </Box>
  );
}
