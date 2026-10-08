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

import { useEffect, useMemo, useState } from 'react';
import {
  Box,
  Chip,
  LinearProgress,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import { EmptyState } from './components/EmptyState';
import {
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
} from 'material-react-table';
import { FLEET_NOT_COLLECTED, PROCESS_ROLE_LABEL } from './constants';
import { SnapshotBar } from './components/SnapshotBar';
import { StatusBadge } from './components/HealthBadge';
import { MemberState } from './components/MemberState';
import { ServiceLink } from './components/ServiceLink';
import { Duration, Percent } from './components/Metric';
import { Unavailable } from './components/Unavailable';
import { useOmTopology } from './topologyHooks';
import { serviceStatusRank, toServiceRows } from './topology';
import { useOmInventoryServices } from './inventoryHooks';
import {
  ageSeconds,
  isFailing,
  joinServiceInventory,
  missingRowReason,
  type OmEstateStatus,
} from './inventory';
import { formatCompactDuration, pluralize } from './format';
import { ProbeValue } from './components/ProbeValue';
import { ServiceDetailDrawer } from './components/ServiceDetailDrawer';
import type {
  OmInventoryService,
  OmServiceInventoryRow,
  OmServiceStatus,
} from './types';
import { OmError } from './components/OmError';

/** A full `mongod` command line is a paragraph; the cell shows it on hover. */
const TRUNCATED = {
  display: 'block',
  maxWidth: 320,
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
} as const;

/**
 * The columns a row opens with: what it is, where it runs, whether it is healthy, and
 * the two numbers a DBA checks first. Everything else is a column-chooser away, or in
 * the row's own detail panel.
 *
 * Eighteen of twenty-four used to be visible -- about four screen widths at 1440px,
 * which is wide enough that a row cannot be read as a row, so the page could not
 * answer "which one needs attention" at all (design review P17).
 */
const HIDDEN_BY_DEFAULT = {
  env_name: false,
  cluster_name: false,
  replication_set: false,
  process_role: false,
  installed_version: false,
  probe_status: false,
  vendor: false,
  endpoint: false,
  cpu_usage_percent: false,
  connections_free_percent: false,
  // Hidden to make the table fit, not because they do not matter: eight columns,
  // two of them 24-character names, cannot sit in the ~980px this page gets at
  // 1440 with the nav open. Both are a click away in the chooser and both are in
  // the detail drawer, where a per-service read belongs. Repl. lag stays as the
  // one replication signal on the row.
  version: false,
  oplog_window_seconds: false,
  service_id: false,
  service_type: false,
  edition: false,
  config_path: false,
  argv: false,
  node_id: false,
};

/** Columns for the flat service table. Grouping keys lead, then identity, then load. */
function useColumns(
  estate: OmEstateStatus
): MRT_ColumnDef<OmServiceInventoryRow>[] {
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
      {
        accessorKey: 'service_name',
        size: 200,
        header: 'Service',
        Cell: ({ row: { original } }) => (
          <ServiceLink serviceName={original.service_name} />
        ),
      },
      {
        accessorKey: 'host',
        size: 200,
        header: 'Node',
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
        // Wide enough for "for 23h 59m" beside a Down chip.
        size: 190,
        header: 'Status',
        // Worst first ascending, by rank rather than by the enum's spelling.
        sortingFn: (a, b, columnId) =>
          serviceStatusRank(a.getValue<OmServiceStatus>(columnId)) -
          serviceStatusRank(b.getValue<OmServiceStatus>(columnId)),
        Cell: ({ row: { original } }) => (
          <StatusBadge
            status={original.status}
            lastUpAt={original.last_up_at}
          />
        ),
      },
      {
        accessorKey: 'process_role',
        header: 'Process',
        Cell: ({ row: { original } }) =>
          PROCESS_ROLE_LABEL[original.process_role] ?? original.process_role,
      },
      {
        accessorKey: 'state',
        size: 185,
        header: 'Member state',
        Cell: ({ row: { original } }) => <MemberState service={original} />,
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
            estate={estate}
            value={original.inventory?.installed_version}
          />
        ),
      },
      {
        id: 'probe_status',
        accessorFn: (row) => row.inventory?.probe_status ?? null,
        header: 'Scan',
        Cell: ({ row: { original } }) =>
          original.inventory ? (
            <ProbeStatus inventory={original.inventory} />
          ) : (
            <Unavailable reason={missingRowReason(estate)} />
          ),
      },
      {
        id: 'last_success_at',
        size: 140,
        // Sorted on the age in seconds, not the timestamp string: the column is read
        // as "how stale", and a lexicographic sort of ISO strings puts a row that has
        // never answered next to the oldest one rather than at the end.
        accessorFn: (row) =>
          ageSeconds(row.inventory?.freshness.last_success_at) ?? Infinity,
        header: 'Collected',
        Cell: ({ row: { original } }) => {
          if (!original.inventory) {
            return <Unavailable reason={missingRowReason(estate)} />;
          }
          const age = ageSeconds(original.inventory.freshness.last_success_at);
          return age == null ? (
            <Unavailable reason="probe_never_succeeded" />
          ) : (
            <>{formatCompactDuration(age)} ago</>
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
        size: 145,
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
            estate={estate}
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
            <Unavailable reason={missingRowReason(estate)} />
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
            <ProbeValue
              inventory={original.inventory}
              estate={estate}
              value={null}
            />
          ),
      },
    ],
    [estate]
  );
}

/**
 * Counts above the table, so the headline numbers need no reading of rows.
 *
 * `failing` counts probes, not services: a service can be UP to PMM and failing its
 * probe, which is the pair of facts this page exists to put side by side. It doubles
 * as the filter, because a count nobody can act on is decoration.
 */
const Counts = ({
  total,
  up,
  down,
  failing,
  estate,
  failingOnly,
  onToggleFailing,
}: {
  total: number;
  up: number;
  down: number;
  /** null whenever the estate has not answered, so no count can honestly be shown. */
  failing: number | null;
  /** Which of the two silences it is, so the placeholder says the right one. */
  estate: OmEstateStatus;
  failingOnly: boolean;
  onToggleFailing: () => void;
}) => {
  return (
    <Stack direction="row" spacing={3} sx={{ mb: 2, alignItems: 'center' }}>
      <Typography variant="body2">
        <strong>{total}</strong> {pluralize(total, 'service')}
      </Typography>
      <Typography variant="body2">
        <strong>{up}</strong> up
      </Typography>
      <Typography variant="body2" color={down ? 'error.main' : undefined}>
        <strong>{down}</strong> down
      </Typography>
      {failing === null ? (
        <Typography variant="body2" color="text.secondary">
          {estate === 'pending'
            ? 'reading scan status…'
            : 'scan status unavailable'}
        </Typography>
      ) : failing > 0 || failingOnly ? (
        <Chip
          size="small"
          color={failingOnly ? 'error' : 'default'}
          variant={failingOnly ? 'filled' : 'outlined'}
          label={`${failing} failing a scan`}
          onClick={onToggleFailing}
        />
      ) : null}
    </Stack>
  );
};

/**
 * A service's probe outcome, as a word rather than a status string.
 *
 * Failing is shown with how long it has been failing, because that is the difference
 * between a blip and something to act on: a table of fifteen rows where one has been
 * failing for three days looks identical to a healthy one otherwise.
 */
const ProbeStatus = ({ inventory }: { inventory: OmInventoryService }) => {
  const since = ageSeconds(inventory.freshness.failing_since);
  if (since == null) {
    return <>{inventory.probe_status ?? 'ok'}</>;
  }
  return (
    <Tooltip
      title={
        inventory.freshness.last_error ??
        'The last scan of this service failed.'
      }
    >
      <Box component="span" sx={{ color: 'error.main', cursor: 'help' }}>
        failing {formatCompactDuration(since)}
        {inventory.freshness.consecutive_failures > 0
          ? ` (${inventory.freshness.consecutive_failures}x)`
          : ''}
      </Box>
    </Tooltip>
  );
};

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
export const FleetServicesTab = () => {
  const { data, isPending, isError, error } = useOmTopology();
  // Deliberately not gated on the estate loading or failing. The snapshot is PMM's own
  // and always available; the estate is a second service that may be unwell, and a
  // page that blanked when it was would be exactly what proxying it through
  // pmm-managed was meant to stop.
  const {
    data: inventory,
    isPending: inventoryPending,
    isError: inventoryFailed,
    error: inventoryError,
  } = useOmInventoryServices();
  // Three states, not two. The estate being *in flight* is not the estate being empty:
  // the topology document answers in a tenth of a second, so on first paint every row
  // would otherwise be labelled "not in the inventory" and the chip would read
  // "0 failing" - two claims about an estate that has not answered yet.
  const estate: OmEstateStatus = inventoryFailed
    ? 'unavailable'
    : inventoryPending
      ? 'pending'
      : 'ready';
  const [failingOnly, setFailingOnly] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const columns = useColumns(estate);
  const joined = useMemo(
    () => joinServiceInventory(toServiceRows(data), inventory),
    [data, inventory]
  );
  // Looked up by id so the drawer follows the poll, and closes when the service
  // leaves the snapshot.
  const selected = useMemo(
    () =>
      selectedId === null
        ? null
        : (joined.find((row) => row.service_id === selectedId) ?? null),
    [joined, selectedId]
  );
  useEffect(() => {
    if (selected === null) {
      setSelectedId(null);
    }
  }, [selected]);
  const rows = useMemo(
    () =>
      failingOnly
        ? joined.filter((row) => row.inventory && isFailing(row.inventory))
        : joined,
    [joined, failingOnly]
  );
  // null rather than 0 whenever the estate has not answered - failed or still in
  // flight. "0 failing" is a claim, and an empty join is not evidence for it either way.
  const failingCount = useMemo(
    () =>
      estate === 'ready'
        ? joined.filter((row) => row.inventory && isFailing(row.inventory))
            .length
        : null,
    [joined, estate]
  );

  const table = useMaterialReactTable({
    columns,
    data: rows,
    enableGrouping: true,
    // See FleetClustersTab: without these MRT sizes every column to its header's
    // chrome rather than its content, and the table overflows the width the page
    // actually gets. The per-column menu's only verb beyond sorting is "hide this
    // column", which the chooser in the toolbar already does.
    layoutMode: 'grid',
    enableColumnActions: false,
    enablePagination: false,
    enableDensityToggle: false,
    // The short table cannot answer "tell me everything about this row", and the
    // column chooser answers it for every row at once. Clicking opens the drawer
    // instead. Grouping headers are rows too and carry no service, so they are
    // left alone.
    muiTableBodyRowProps: ({ row }) => ({
      hover: true,
      sx: row.getIsGrouped() ? undefined : { cursor: 'pointer' },
      onClick: row.getIsGrouped()
        ? undefined
        : (event) => {
            // A link or button in the row has its own job, e.g. a Cmd-click on
            // the service name opening its dashboard in a new tab.
            if ((event.target as HTMLElement).closest('a, button')) {
              return;
            }
            setSelectedId(row.original.service_id ?? null);
          },
    }),
    initialState: {
      density: 'compact',
      columnVisibility: HIDDEN_BY_DEFAULT,
      // Down services first, so a failure is the first row a reader sees.
      sorting: [
        { id: 'status', desc: false },
        { id: 'cluster_name', desc: false },
        { id: 'service_name', desc: false },
      ],
    },
  });

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    // Just the alert now: the page's header, and the Sync action that is the way out
    // of the expected first-run 503, belong to FleetPage and are rendered whichever
    // tab is open. A branch that named the fix without offering it was the bug here,
    // and hoisting the header is what fixes it for both tabs at once.
    return (
      <OmError
        placement="load"
        title="Could not load the fleet"
        messages={(error as Error)?.message}
      />
    );
  }

  return (
    <Box>
      <SnapshotBar envelope={data.snapshot} />
      {inventoryFailed && (
        <Box sx={{ mb: 2 }}>
          <OmError
            placement="load"
            severity="warning"
            title="Could not read scan results"
            messages={[
              "The scan columns are blank and no scan count is shown. The monitoring columns come from PMM's own data and are unaffected.",
              inventoryError instanceof Error ? inventoryError.message : null,
            ]}
          />
        </Box>
      )}
      <Counts
        total={data.summary.total_services}
        up={data.summary.up_services}
        down={data.summary.down_services}
        failing={failingCount}
        estate={estate}
        failingOnly={failingOnly}
        onToggleFailing={() => setFailingOnly((on) => !on)}
      />
      {rows.length > 0 ? (
        <MaterialReactTable table={table} />
      ) : joined.length > 0 ? (
        <EmptyState title="No services failing a scan">
          No service is failing a scan right now. Turn off the failing filter to
          see them all.
        </EmptyState>
      ) : (
        <EmptyState title="No MongoDB services yet">
          The same fleet as the Clusters tab, one row per MongoDB service.{' '}
          {data.snapshot.generated_at
            ? 'It is empty because PMM has no MongoDB services registered yet - add one, and it appears here on the next refresh.'
            : FLEET_NOT_COLLECTED}
        </EmptyState>
      )}
      <ServiceDetailDrawer
        row={selected}
        estate={estate}
        onClose={() => setSelectedId(null)}
      />
    </Box>
  );
};
