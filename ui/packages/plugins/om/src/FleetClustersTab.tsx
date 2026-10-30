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
import { useSearchParams } from 'react-router-dom';
import {
  Box,
  Button,
  InputAdornment,
  LinearProgress,
  Link,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
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
import {
  CLUSTER_HEALTH_LABEL,
  CLUSTER_TYPE_LABEL,
  FLEET_NOT_COLLECTED,
  OPERATIONS_DOC_URL,
  PROCESS_ROLE_LABEL,
} from './constants';
import { NESTED_TABLE_WRAPPER } from './nestedTable';
import { SnapshotBar } from './components/SnapshotBar';
import { ClusterHealthBadge, StatusBadge } from './components/HealthBadge';
import { MemberState } from './components/MemberState';
import { ServiceLink } from './components/ServiceLink';
import { Duration, Percent } from './components/Metric';
import { Age } from './components/Age';
import {
  MultiSelectFilter,
  type FilterOption,
} from './components/MultiSelectFilter';
import { Unavailable } from './components/Unavailable';
import {
  GroupBySelect,
  NO_GROUPING,
  useSessionGrouping,
  type GroupByOption,
} from './components/GroupBySelect';
import { useOmTopology } from './topologyHooks';
import { useOmInventoryServices } from './inventoryHooks';
import { pluralize } from './format';
import {
  clusterHealthRank,
  compareVersions,
  downFirst,
  toClusterRows,
} from './topology';
import {
  HEALTH_FILTER_VALUES,
  TYPE_FILTER_VALUES,
  filterClusterRows,
  hasClusterFilters,
  parseClusterFilters,
  withClusterFilters,
  type ClusterFilters,
} from './clusterFilters';
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

type GroupingColumn = Exclude<ClusterGrouping, typeof NO_GROUPING>;

/**
 * A cluster's value in each grouping column, as the column's accessor reads it.
 *
 * The raw name, never the label: an environment really called "Unassigned
 * environment" must not fold into the unnamed one.
 */
const GROUP_VALUE: Record<GroupingColumn, (row: OmClusterRow) => string> = {
  env_name: (row) => row.env_name ?? '',
  health: (row) => row.health,
};

const groupLabel = (row: OmClusterRow, grouping: GroupingColumn) =>
  grouping === 'health' ? row.health : environmentLabel(row);

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
 *
 * Type is a label beside the cluster's name, so its own column only earns a place for
 * sorting on it; Last seen, CPU and Connections are a click away for the same width
 * reason (Pedro, PMM-15806).
 */
const HIDDEN_BY_DEFAULT = {
  total_services: false,
  roles: false,
  versions: false,
  type: false,
  last_seen_at: false,
  max_cpu_percent: false,
  min_connections_free_percent: false,
};

const typeLabel = (row: OmClusterRow) => CLUSTER_TYPE_LABEL[row.type];

/** Element by element, oldest first, so a cluster still on an old build leads. */
const compareVersionLists = (a: string[], b: string[]) => {
  for (let i = 0; i < Math.min(a.length, b.length); i++) {
    const order = compareVersions(a[i], b[i]);
    if (order !== 0) {
      return order;
    }
  }
  return a.length - b.length;
};

const ClusterVersions = ({ cluster }: { cluster: OmClusterRow }) => {
  if (!cluster.versions.length) {
    return <Unavailable reason="service_not_observed" />;
  }
  if (cluster.version_drift) {
    // A replica set running two versions is worth reading as a finding, not as a
    // longer cell.
    return (
      <Tooltip
        title={
          <Box data-testid="om-version-sets">
            {cluster.version_sets.map((set) => (
              <div key={set.replication_set}>
                {set.replication_set}: {set.versions.join(', ')}
              </div>
            ))}
          </Box>
        }
      >
        <Typography variant="body2" component="span" color="warning.main">
          Mixed
        </Typography>
      </Tooltip>
    );
  }
  if (cluster.versions.length === 1) {
    return <>{cluster.versions[0]}</>;
  }
  // Shards or routers on different versions, each set consistent: not drift.
  return (
    <Tooltip title={cluster.versions.join(', ')}>
      <span>{cluster.versions.length} versions</span>
    </Tooltip>
  );
};

function useColumns(): MRT_ColumnDef<OmClusterRow>[] {
  return useMemo(
    () => [
      {
        accessorFn: GROUP_VALUE.env_name,
        id: 'env_name',
        header: 'Environment',
        size: 160,
        Cell: ({ row: { original } }) => environmentLabel(original),
      },
      {
        accessorKey: 'cluster_name',
        header: 'Cluster',
        size: 200,
        Cell: ({ row: { original } }) => (
          <Stack
            direction="row"
            alignItems="baseline"
            columnGap={1}
            flexWrap="wrap"
          >
            {original.cluster_name ?? <Unavailable reason="not_applicable" />}
            {typeLabel(original) && (
              <Typography
                variant="caption"
                color="text.secondary"
                data-testid="om-cluster-type"
              >
                {typeLabel(original)}
              </Typography>
            )}
          </Stack>
        ),
      },
      {
        accessorFn: (row) => typeLabel(row),
        id: 'type',
        header: 'Type',
        size: 120,
        sortUndefined: 'last',
        Cell: ({ row: { original } }) =>
          typeLabel(original) ?? <Unavailable reason="not_applicable" />,
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
        accessorFn: (row) => row.versions.join(', ') || undefined,
        id: 'versions',
        header: 'Versions',
        size: 130,
        sortUndefined: 'last',
        // As versions, not strings: 7.0.9-1 is older than 7.0.24-13.
        sortingFn: (a, b) =>
          compareVersionLists(a.original.versions, b.original.versions),
        Cell: ({ row: { original } }) => <ClusterVersions cluster={original} />,
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
      {
        accessorFn: (row) =>
          row.last_seen_at ? Date.parse(row.last_seen_at) : undefined,
        id: 'last_seen_at',
        header: 'Last seen',
        size: 120,
        sortUndefined: 'last',
        Header: () => (
          <Tooltip title="When the least recently observed member was last observed">
            <span>Last seen</span>
          </Tooltip>
        ),
        Cell: ({ row: { original } }) =>
          original.last_seen_at ? (
            <Age value={original.last_seen_at} />
          ) : (
            <Unavailable reason="service_not_observed" />
          ),
      },
      {
        accessorFn: (row) => row.max_cpu_percent ?? undefined,
        id: 'max_cpu_percent',
        header: 'Max CPU',
        size: 110,
        sortUndefined: 'last',
        Cell: ({ row: { original } }) =>
          original.max_cpu_percent == null ? (
            <Unavailable reason="metric_not_collected" />
          ) : (
            <Percent value={original.max_cpu_percent} />
          ),
      },
      {
        accessorFn: (row) => row.min_connections_free_percent ?? undefined,
        id: 'min_connections_free_percent',
        header: 'Min conn. free',
        size: 140,
        sortUndefined: 'last',
        Header: () => (
          <Tooltip title="The lowest percentage of free connections among the members">
            <span>Min conn. free</span>
          </Tooltip>
        ),
        Cell: ({ row: { original } }) =>
          original.min_connections_free_percent == null ? (
            <Unavailable reason="metric_not_collected" />
          ) : (
            <Percent value={original.min_connections_free_percent} />
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
      // Its own width, not the expand column's: squeezed into that, the counts
      // collapse to nothing.
      sx={{ width: 'max-content' }}
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

const NO_FILTERS: ClusterFilters = {
  q: '',
  health: [],
  type: [],
  environment: [],
};

/**
 * The search and filters, kept in the URL.
 *
 * Each change is a history entry, so Back undoes a filter - except while a search is
 * being refined, where one entry per keystroke would make Back retype it backwards.
 */
function useClusterFilters() {
  const [params, setParams] = useSearchParams();
  const filters = useMemo(() => parseClusterFilters(params), [params]);
  const setFilters = (next: ClusterFilters) => {
    const refiningSearch =
      !!filters.q.trim() &&
      !!next.q.trim() &&
      (['health', 'type', 'environment'] as const).every(
        (key) => filters[key].join('\n') === next[key].join('\n')
      );
    setParams((current) => withClusterFilters(current, next), {
      replace: refiningSearch,
    });
  };
  return [filters, setFilters] as const;
}

const HEALTH_OPTIONS: FilterOption[] = HEALTH_FILTER_VALUES.map((value) => ({
  value,
  label: CLUSTER_HEALTH_LABEL[value],
}));

const TYPE_OPTIONS: FilterOption[] = Object.entries(TYPE_FILTER_VALUES).map(
  ([value, type]) => ({ value, label: CLUSTER_TYPE_LABEL[type] ?? value })
);

/** Search and filters, above the table so they stay put when nothing matches. */
const ClusterFilterBar = ({
  rows,
  shown,
  filters,
  onChange,
}: {
  rows: OmClusterRow[];
  shown: number;
  filters: ClusterFilters;
  onChange: (filters: ClusterFilters) => void;
}) => {
  // From the fleet, not a fixed list, with the unnamed one under its label.
  const environments = useMemo(
    () =>
      [...new Set(rows.map((row) => row.env_name ?? ''))]
        .sort((a, b) => a.localeCompare(b))
        .map((value) => ({ value, label: value || UNNAMED_ENVIRONMENT })),
    [rows]
  );
  return (
    <Stack direction="row" alignItems="center" gap={1.5} flexWrap="wrap">
      <TextField
        size="small"
        placeholder="Search clusters"
        value={filters.q}
        onChange={(event) => onChange({ ...filters, q: event.target.value })}
        slotProps={{
          htmlInput: { 'aria-label': 'Search clusters' },
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          },
        }}
        sx={{ width: 240 }}
      />
      <MultiSelectFilter
        label="Health"
        options={HEALTH_OPTIONS}
        value={filters.health}
        onChange={(health) =>
          onChange({ ...filters, health: health as ClusterFilters['health'] })
        }
      />
      <MultiSelectFilter
        label="Type"
        options={TYPE_OPTIONS}
        value={filters.type}
        onChange={(type) => onChange({ ...filters, type })}
      />
      <MultiSelectFilter
        label="Environment"
        options={environments}
        // One the fleet no longer has is ignored by the filter, so not shown as picked.
        value={filters.environment.filter((name) =>
          environments.some((option) => option.value === name)
        )}
        onChange={(environment) => onChange({ ...filters, environment })}
      />
      {hasClusterFilters(filters) && (
        <>
          <Typography
            variant="body2"
            color="text.secondary"
            data-testid="om-cluster-match-count"
          >
            {shown} of {rows.length} {pluralize(rows.length, 'cluster')}
          </Typography>
          <Button size="small" onClick={() => onChange(NO_FILTERS)}>
            Clear filters
          </Button>
        </>
      )}
    </Stack>
  );
};

/**
 * Every cluster in one table, grouped by what the reader picks in the toolbar.
 *
 * One table rather than one per environment, so the clusters can be read by
 * environment, by health or flat without the page changing shape (PMM-15712).
 */
const ClustersTable = ({ rows }: { rows: OmClusterRow[] }) => {
  const [grouping, setGrouping] = useSessionGrouping<ClusterGrouping>(
    'om.fleet.clusters.grouping',
    GROUP_BY_OPTIONS,
    DEFAULT_GROUPING
  );
  const [filters, setFilters] = useClusterFilters();
  const shown = useMemo(
    () => filterClusterRows(rows, filters),
    [rows, filters]
  );
  // The whole fleet, not the matches: a group folded away stays folded while a
  // filter hides it and brings it back.
  const { expanded, onExpandedChange } = useGroupedExpansion(rows, grouping);
  const columns = useColumns();
  const table = useMaterialReactTable({
    columns,
    data: shown,
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
    // else it can hold is off. Search is the filter bar's, so it can live in the URL.
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
    renderEmptyRowsFallback: () => (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        No cluster matches the search and filters.
      </Typography>
    ),
    // MRT labels every toggle "Expand" whatever its state; a group's says which
    // group and whether it is open.
    muiExpandButtonProps: ({ row }) =>
      row.getIsGrouped() && grouping !== NO_GROUPING
        ? {
            'aria-expanded': row.getIsExpanded(),
            'aria-label': `${row.getIsExpanded() ? 'Collapse' : 'Expand'} ${groupLabel(row.original, grouping)}`,
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
        // the other cells, so it runs across them - above them, or their backgrounds
        // paint over it.
        muiTableBodyCellProps: ({ row }) =>
          row.getIsGrouped()
            ? {
                sx: {
                  overflow: 'visible',
                  whiteSpace: 'nowrap',
                  position: 'relative',
                  zIndex: 1,
                },
              }
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

  return (
    <Stack gap={1.5}>
      <ClusterFilterBar
        rows={rows}
        shown={shown.length}
        filters={filters}
        onChange={setFilters}
      />
      <MaterialReactTable table={table} />
    </Stack>
  );
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
  // Read only to say when it fails. This tab shows no scan results, but the Services
  // tab beside it does, and the reader should learn they are missing here rather
  // than from a column of blanks one click away. Same query, so it is shared.
  const { isError: inventoryFailed, error: inventoryError } =
    useOmInventoryServices();
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
        {inventoryFailed && (
          <OmError
            placement="load"
            severity="warning"
            title="Could not read scan results"
            messages={[
              "Until Operations can read them again, the Services tab shows no scan status, or the status from its last successful read. The clusters and their health come from PMM's own data and are unaffected.",
              inventoryError instanceof Error ? inventoryError.message : null,
            ]}
          />
        )}
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
            : FLEET_NOT_COLLECTED}{' '}
          <Link href={OPERATIONS_DOC_URL} target="_blank" rel="noopener">
            Read about Operations for MongoDB
          </Link>
        </EmptyState>
      ) : (
        <ClustersTable rows={rows} />
      )}
    </Stack>
  );
};
