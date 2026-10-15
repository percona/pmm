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
  Box,
  LinearProgress,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import { EmptyState } from './components/EmptyState';
import {
  MRT_ExpandAllButton,
  MRT_ExpandButton,
  MaterialReactTable,
  useMaterialReactTable,
  type MRT_ColumnDef,
  type MRT_ExpandedState,
  type MRT_Row,
  type MRT_TableInstance,
} from 'material-react-table';
import { FLEET_NOT_COLLECTED, PROCESS_ROLE_LABEL } from './constants';
import { NESTED_TABLE_WRAPPER } from './nestedTable';
import { SnapshotBar } from './components/SnapshotBar';
import { ClusterHealthBadge, StatusBadge } from './components/HealthBadge';
import { MemberState } from './components/MemberState';
import { ServiceLink } from './components/ServiceLink';
import { Duration } from './components/Metric';
import { Unavailable } from './components/Unavailable';
import {
  GroupBySelect,
  NO_GROUPING,
  type GroupByOption,
} from './components/GroupBySelect';
import { useOmTopology } from './topologyHooks';
import { pluralize } from './format';
import { clusterHealthRank, downFirst, toClusterRows } from './topology';
import type { OmClusterHealth, OmClusterRow, OmProcessRole } from './types';
import { OmError } from './components/OmError';

/** Label for an environment or cluster the services carry no name for. */
const UNNAMED_ENVIRONMENT = 'Unassigned environment';

const environmentLabel = (row: OmClusterRow) =>
  row.env_name ?? UNNAMED_ENVIRONMENT;

/** `3 mongod · 1 Router`, in the document's own role order. */
function describeRoles(roles: Partial<Record<OmProcessRole, number>>): string {
  return Object.entries(roles)
    .map(
      ([role, count]) =>
        `${count} ${PROCESS_ROLE_LABEL[role as OmProcessRole] ?? role}`
    )
    .join(' · ');
}

type ClusterGrouping = 'env_name' | 'health' | typeof NO_GROUPING;

/**
 * Environment by default, not Health: grouping by health would reshuffle the groups
 * every time a member flips, and the down-first sort already puts trouble at the top
 * of each environment (PMM-15712).
 */
const DEFAULT_GROUPING: ClusterGrouping = 'env_name';

const GROUP_BY_OPTIONS: GroupByOption[] = [
  { value: 'env_name', label: 'Environment' },
  { value: 'health', label: 'Health' },
  { value: NO_GROUPING, label: 'None' },
];

/** A cluster's value in each grouping column, as the column's accessor reads it. */
const GROUP_VALUE: Record<
  Exclude<ClusterGrouping, typeof NO_GROUPING>,
  (row: OmClusterRow) => string
> = {
  env_name: environmentLabel,
  health: (row) => row.health,
};

/**
 * Columns the cluster table carries but does not open with.
 *
 * Measured rather than guessed: at 1440px with the nav expanded the content column is
 * about 980px, and nine columns plus the expand control overflowed it - so the first
 * page a reader opens had a scrollbar and three columns off-screen, which is the whole
 * of P17.
 *
 * Services goes because Members states the same total as its denominator, and the
 * width it frees is what lets Environment show when the table is not grouped by it.
 *
 * Process is a roll-up that a reader asking "is this a router" answers by unfolding
 * the row, whose member table carries its own Process column.
 *
 * Versions is a roll-up of the members, and a reader who wants it is asking about one
 * cluster - which is what unfolding the row answers.
 */
const HIDDEN_BY_DEFAULT = {
  total_services: false,
  roles: false,
  versions: false,
};

function useColumns(): MRT_ColumnDef<OmClusterRow>[] {
  return useMemo(
    () => [
      {
        accessorFn: environmentLabel,
        id: 'env_name',
        header: 'Environment',
        size: 160,
      },
      {
        accessorKey: 'cluster_name',
        header: 'Cluster',
        size: 200,
        Cell: ({ row: { original } }) =>
          original.cluster_name ?? <Unavailable reason="not_applicable" />,
      },
      {
        accessorKey: 'health',
        header: 'Health',
        size: 130,
        // Worst first ascending, so one click brings trouble to the top.
        sortingFn: (a, b, columnId) =>
          clusterHealthRank(a.getValue<OmClusterHealth>(columnId)) -
          clusterHealthRank(b.getValue<OmClusterHealth>(columnId)),
        Cell: ({ row: { original } }) => (
          <ClusterHealthBadge health={original.health} />
        ),
      },
      {
        // Plain text, never a status colour: the Health column is the verdict.
        accessorFn: (row) => row.up_services,
        id: 'members',
        header: 'Members',
        size: 120,
        Header: () => (
          <Tooltip title="Members up, out of all the members in the cluster">
            <span>Members</span>
          </Tooltip>
        ),
        Cell: ({ row: { original } }) =>
          `${original.up_services}/${original.total_services}`,
      },
      { accessorKey: 'total_services', header: 'Services', size: 100 },
      {
        accessorFn: (row) => describeRoles(row.by_process_role),
        id: 'roles',
        header: 'Process',
        size: 130,
      },
      {
        accessorFn: (row) => row.versions.join(', '),
        id: 'versions',
        header: 'Versions',
        size: 130,
        Cell: ({ row: { original } }) => {
          if (!original.versions.length) {
            return <Unavailable reason="service_not_observed" />;
          }
          if (original.versions.length === 1) {
            return original.versions[0];
          }
          // More than one running version in one cluster is worth reading as a
          // finding, not as a longer cell.
          return (
            <Tooltip title={original.versions.join(', ')}>
              <Typography variant="body2" component="span" color="warning.main">
                Mixed ({original.versions.length})
              </Typography>
            </Tooltip>
          );
        },
      },
      {
        accessorKey: 'max_replication_lag_seconds',
        header: 'Max repl. lag',
        size: 130,
        Cell: ({ row: { original } }) => (
          <Duration value={original.max_replication_lag_seconds} />
        ),
      },
      {
        accessorKey: 'min_oplog_window_seconds',
        header: 'Min oplog window',
        size: 150,
        Cell: ({ row: { original } }) => (
          <Duration value={original.min_oplog_window_seconds} />
        ),
      },
    ],
    []
  );
}

/**
 * The services of one cluster, shown when its row is unfolded.
 *
 * Deliberately a plain table rather than a nested data grid: this is the roll-up
 * being shown its working, so it needs no second set of sorters and filters.
 *
 * Eight columns, not ten. At 1440px with the nav expanded this panel gets about
 * 980px, and ten overflowed it - so unfolding a cluster produced a scrollbar and
 * four columns a reader could not see. CPU and connections went: a per-member load
 * read is what the Services tab and its detail drawer are for, and this panel exists
 * to answer "are this cluster's members healthy".
 *
 * Process was cut too, on the reasoning that it says "mongod" for every member - and
 * put back, because that is only true until a cluster is sharded. A mongos has no
 * member state, so Process is the only thing identifying it. PMM-15652's test caught
 * it.
 *
 * Being a plain table, these are removals rather than hidden-by-default: there is no
 * column chooser here to bring them back. That is the trade for not making it a
 * second data grid.
 */
const ClusterServices = ({ cluster }: { cluster: OmClusterRow }) => {
  if (!cluster.services.length) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        This cluster has no services in the current snapshot.
      </Typography>
    );
  }
  return (
    <Box sx={NESTED_TABLE_WRAPPER}>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Service</TableCell>
            <TableCell>Host</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Member state</TableCell>
            <TableCell>Process</TableCell>
            <TableCell>Version</TableCell>
            <TableCell>Repl. lag</TableCell>
            <TableCell>Oplog window</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {downFirst(cluster.services).map((service) => (
            <TableRow key={service.service_name}>
              <TableCell>
                <ServiceLink serviceName={service.service_name} />
              </TableCell>
              <TableCell>
                {service.host ?? <Unavailable reason="service_not_observed" />}
              </TableCell>
              <TableCell>
                <StatusBadge
                  status={service.status}
                  lastUpAt={service.last_up_at}
                />
              </TableCell>
              <TableCell>
                <MemberState service={service} />
              </TableCell>
              <TableCell>
                {PROCESS_ROLE_LABEL[service.process_role] ??
                  service.process_role}
              </TableCell>
              <TableCell>
                {service.version ?? (
                  <Unavailable reason="service_not_observed" />
                )}
              </TableCell>
              <TableCell>
                <Duration value={service.replication_lag_seconds} />
              </TableCell>
              <TableCell>
                <Duration value={service.oplog_window_seconds} />
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </Box>
  );
};

/**
 * A group's header row: what it is grouped on, and the counts the per-environment
 * headers carried, so a folded group still says how it is doing.
 *
 * "Up" stays neutral and "down" turns red only above zero: with the Health column
 * removed from a grouped row, a red "down" is its only alarm.
 */
const GroupHeader = ({
  row,
  table,
  staticRowIndex,
  grouping,
}: {
  row: MRT_Row<OmClusterRow>;
  table: MRT_TableInstance<OmClusterRow>;
  staticRowIndex?: number;
  grouping: ClusterGrouping;
}) => {
  const clusters = row.getLeafRows().map((leaf) => leaf.original);
  const sum = (field: 'total_services' | 'up_services' | 'down_services') =>
    clusters.reduce((total, cluster) => total + cluster[field], 0);
  const services = sum('total_services');
  const down = sum('down_services');

  return (
    <Stack
      direction="row"
      alignItems="center"
      gap={2}
      data-testid="om-cluster-group"
    >
      <Stack direction="row" alignItems="center" gap={0.5}>
        <MRT_ExpandButton
          row={row}
          table={table}
          staticRowIndex={staticRowIndex}
        />
        {grouping === 'health' ? (
          <ClusterHealthBadge health={row.original.health} />
        ) : (
          <Typography variant="subtitle2" component="h2">
            {environmentLabel(row.original)}
          </Typography>
        )}
      </Stack>
      <Typography variant="body2" color="text.secondary">
        <strong>{clusters.length}</strong>{' '}
        {pluralize(clusters.length, 'cluster')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        <strong>{services}</strong> {pluralize(services, 'service')}
      </Typography>
      <Typography variant="body2" color="text.secondary">
        <strong>{sum('up_services')}</strong> up
      </Typography>
      <Typography
        variant="body2"
        color={down ? 'error.main' : 'text.secondary'}
      >
        <strong>{down}</strong> down
      </Typography>
    </Stack>
  );
};

/**
 * Group rows open, and cluster rows folded, by default.
 *
 * TanStack's expanded state cannot say "open unless closed": `true` would also unfold
 * every cluster's member table, and collapsing a row deletes its key rather than
 * storing `false`. So the two are kept apart - the groups a reader closed, and the
 * clusters they opened - and merged into the state the table reads.
 */
function useGroupedExpansion(rows: OmClusterRow[], grouping: ClusterGrouping) {
  const [closedGroups, setClosedGroups] = useState<Record<string, boolean>>({});
  const [openClusters, setOpenClusters] = useState<Record<string, boolean>>({});
  // TanStack names a top-level group row `<columnId>:<groupingValue>`.
  const groupIds = useMemo(
    () =>
      grouping === NO_GROUPING
        ? []
        : [
            ...new Set(
              rows.map((row) => `${grouping}:${GROUP_VALUE[grouping](row)}`)
            ),
          ],
    [rows, grouping]
  );
  const expanded = useMemo(() => {
    const state: Record<string, boolean> = { ...openClusters };
    for (const id of groupIds) {
      if (!closedGroups[id]) {
        state[id] = true;
      }
    }
    return state;
  }, [groupIds, closedGroups, openClusters]);

  const onExpandedChange = (
    updater: MRT_ExpandedState | ((old: MRT_ExpandedState) => MRT_ExpandedState)
  ) => {
    const next = typeof updater === 'function' ? updater(expanded) : updater;
    // `true` is "expand all": every group and every cluster.
    const isOpen = (id: string) => next === true || !!next[id];
    setClosedGroups(
      Object.fromEntries(
        groupIds.filter((id) => !isOpen(id)).map((id) => [id, true])
      )
    );
    setOpenClusters(
      Object.fromEntries(
        rows.filter((row) => isOpen(row.id)).map((row) => [row.id, true])
      )
    );
  };

  return { expanded, onExpandedChange };
}

/**
 * Every cluster in one table, grouped by what the reader picks in the toolbar.
 *
 * One table rather than one per environment, so the clusters can be read by
 * environment, by health or flat without the page changing shape (PMM-15712).
 */
const ClustersTable = ({ rows }: { rows: OmClusterRow[] }) => {
  const [grouping, setGrouping] = useState<ClusterGrouping>(DEFAULT_GROUPING);
  const { expanded, onExpandedChange } = useGroupedExpansion(rows, grouping);
  const columns = useColumns();
  const table = useMaterialReactTable({
    columns,
    data: rows,
    enablePagination: false,
    enableDensityToggle: false,
    enableExpanding: true,
    enableGrouping: true,
    // The grouped column says the same thing on every row of its group, and the
    // group header already says it once.
    groupedColumnMode: 'remove',
    // Grouping is the toolbar select's job: no drag handle on any header, and no
    // "grouped by" banner offering a second way to change it.
    enableColumnDragging: false,
    positionToolbarAlertBanner: 'none',
    // Grid layout, so the `size` on each column is honoured. The default sizes every
    // column to its header's chrome instead, which at 1440px with the nav open spent
    // the whole ~980px on six columns of mostly single digits.
    layoutMode: 'grid',
    // No per-column menu. Its only verb beyond sorting is "hide this column", which
    // the chooser does, and the icon it adds to every header is a slice of the width
    // this table does not have.
    enableColumnActions: false,
    // The toolbar carries the Group by select and the column chooser; everything
    // else it can hold is off.
    enableGlobalFilter: false,
    enableFullScreenToggle: false,
    enableHiding: true,
    // The opaque server-issued id, not the label: two clusters can share a label (a
    // sandbox's second generation reusing a name is not hypothetical here) or carry
    // none at all, and MRT uses this for expansion and selection state across
    // refetches -- an index-based fallback would misattribute that state the moment
    // the row order shifted between polls.
    getRowId: (row) => row.id,
    renderDetailPanel: ({ row }) => <ClusterServices cluster={row.original} />,
    // MRT labels every toggle "Expand" whatever its state; a group's says which
    // group and whether it is open.
    muiExpandButtonProps: ({ row }) =>
      row.getIsGrouped() && grouping !== NO_GROUPING
        ? {
            'aria-expanded': row.getIsExpanded(),
            'aria-label': `${row.getIsExpanded() ? 'Collapse' : 'Expand'} ${GROUP_VALUE[grouping](row.original)}`,
          }
        : {},
    renderTopToolbarCustomActions: () => (
      <GroupBySelect
        options={GROUP_BY_OPTIONS}
        value={grouping}
        onChange={(value) => setGrouping(value as ClusterGrouping)}
      />
    ),
    displayColumnDefOptions: {
      'mrt-row-expand': {
        size: 60,
        // Just the expand-all control: the "remove" mode's default lists the
        // grouped columns' names here, which the select already shows.
        Header: ({ table }) => <MRT_ExpandAllButton table={table} />,
        Cell: ({ row, table, staticRowIndex }) =>
          row.getIsGrouped() ? (
            <GroupHeader
              row={row}
              table={table}
              staticRowIndex={staticRowIndex}
              grouping={grouping}
            />
          ) : (
            <MRT_ExpandButton
              row={row}
              table={table}
              staticRowIndex={staticRowIndex}
            />
          ),
        // A group header is wider than the expand column; its row has nothing in
        // the other cells, so it runs across them.
        muiTableBodyCellProps: ({ row }) =>
          row.getIsGrouped()
            ? { sx: { overflow: 'visible', whiteSpace: 'nowrap' } }
            : {},
      },
    },
    state: {
      grouping: grouping === NO_GROUPING ? [] : [grouping],
      expanded,
    },
    onExpandedChange,
    initialState: {
      density: 'compact',
      columnVisibility: HIDDEN_BY_DEFAULT,
      // Trouble first: a degraded or down cluster leads its group. Groups themselves
      // keep the document's order, since they have no health of their own to sort on.
      sorting: [
        { id: 'health', desc: false },
        { id: 'cluster_name', desc: false },
      ],
    },
  });

  return <MaterialReactTable table={table} />;
};

/** Fleet-level counts, so the headline numbers need no reading of rows. */
const Counts = ({
  environments,
  clusters,
  total,
  up,
  down,
}: {
  environments: number;
  clusters: number;
  total: number;
  up: number;
  down: number;
}) => {
  return (
    <Stack direction="row" spacing={3} flexWrap="wrap">
      <Typography variant="body2">
        <strong>{environments}</strong> {pluralize(environments, 'environment')}
      </Typography>
      <Typography variant="body2">
        <strong>{clusters}</strong> {pluralize(clusters, 'cluster')}
      </Typography>
      <Typography variant="body2">
        <strong>{total}</strong> {pluralize(total, 'service')}
      </Typography>
      <Typography variant="body2">
        <strong>{up}</strong> up
      </Typography>
      <Typography variant="body2" color={down ? 'error.main' : undefined}>
        <strong>{down}</strong> down
      </Typography>
    </Stack>
  );
};

/**
 * The estate one level above the service table: a row per cluster, grouped by
 * environment unless the reader picks otherwise, and each row unfolds into the
 * services it was rolled up from.
 *
 * The document is a nested `environments -> clusters -> services` tree and this is the
 * reading that keeps the nesting whole. Topology renders the same snapshot flat, one
 * row per service and every field it carries, for anyone who needs to sort or filter
 * across the estate rather than read it by environment.
 */
export const FleetClustersTab = () => {
  const { data, isPending, isError, error } = useOmTopology();
  const rows = useMemo(() => toClusterRows(data), [data]);

  if (isPending) {
    return <LinearProgress />;
  }

  if (isError) {
    // Just the alert: the header, and the Sync action that is the way out of the
    // expected first-run 503, belong to FleetPage and render above whichever tab is
    // open. This tab is the index route, so a fresh install lands here first.
    return (
      <OmError
        placement="load"
        title="Could not load the fleet"
        messages={(error as Error)?.message}
      />
    );
  }

  return (
    <Stack gap={3}>
      <Stack gap={1}>
        <SnapshotBar envelope={data.snapshot} />
        <Counts
          environments={data.summary.environments}
          clusters={data.summary.clusters}
          total={data.summary.total_services}
          up={data.summary.up_services}
          down={data.summary.down_services}
        />
      </Stack>

      {rows.length === 0 ? (
        <EmptyState title="No MongoDB clusters yet">
          This page shows every MongoDB cluster PMM monitors, and the health of
          each one&apos;s members.{' '}
          {data.snapshot.generated_at
            ? 'It is empty because PMM has no MongoDB services registered yet - add one, and it appears here on the next refresh.'
            : FLEET_NOT_COLLECTED}
        </EmptyState>
      ) : (
        <ClustersTable rows={rows} />
      )}
    </Stack>
  );
};
