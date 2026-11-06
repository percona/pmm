import { useMemo, useState, type FC, type ReactNode } from 'react';
import Typography from '@mui/material/Typography';
import {
  Table,
  Tooltip,
  type MRT_ColumnDef,
  type MRT_SortingState,
} from '@percona/peak-ui';
import type { MRT_ExpandedState, MRT_Updater } from 'material-react-table';
import { Messages } from '../AmazonRds.messages';
import type {
  GroupColumnId,
  InstanceGroup,
  WritersState,
} from '../AmazonRds.types';
import {
  formatStorage,
  formatThroughput,
  getEngineLabel,
  getGroupType,
  getGroupTypeLabel,
  getWriters,
  GROUP_SORT_VALUES,
  sortGroups,
} from '../AmazonRds.utils';
import { AlertsBadge } from './AlertsBadge';
import { InstancesTable } from './InstancesTable';

interface GroupsTableProps {
  groups: InstanceGroup[];
  autoExpanded: Set<string>;
  // Rows the user expanded or collapsed are forgotten when this changes (the search text).
  expansionKey: string;
}

// Table's own size for the expand buttons, which these props replace.
const EXPAND_BUTTON_SX = { height: 40, width: 40 };

// Content-sized cells with the compact padding of PMM's other tables; long headers wrap.
const BODY_CELL_SX = { width: 'auto', minWidth: 'auto', px: 1 };
const HEAD_CELL_SX = {
  ...BODY_CELL_SX,
  '& .Mui-TableHeadCell-Content-Wrapper': { whiteSpace: 'normal' },
};

const WRITERS_COLORS: Record<WritersState, string> = {
  good: 'success.main',
  warning: 'warning.main',
  unknown: 'text.secondary',
};

// Tooltips describe their text (describeChild) rather than replace its accessible name.
const column = (
  id: GroupColumnId,
  header: string,
  render: (group: InstanceGroup) => ReactNode,
  description?: string
): MRT_ColumnDef<InstanceGroup> => ({
  id,
  header,
  accessorFn: GROUP_SORT_VALUES[id],
  Cell: ({ row }) => render(row.original),
  ...(description
    ? {
        Header: () => (
          <Tooltip title={description} describeChild>
            <span>{header}</span>
          </Tooltip>
        ),
      }
    : {}),
});

const COLUMNS: MRT_ColumnDef<InstanceGroup>[] = [
  column('region', Messages.columns.region, (group) => group.region),
  column(
    'awsAccountId',
    Messages.columns.awsAccount,
    (group) => group.awsAccountId
  ),
  column('name', Messages.columns.identifier, (group) =>
    group.kind === 'unknown' ? (
      <Tooltip title={Messages.groupTypeTitles.unavailable} describeChild>
        <Typography
          component="span"
          variant="inherit"
          color="text.secondary"
          sx={{ fontStyle: 'italic' }}
        >
          {group.name}
        </Typography>
      </Tooltip>
    ) : (
      <Typography component="span" variant="inherit" fontWeight="medium">
        {group.name}
      </Typography>
    )
  ),
  column('type', Messages.columns.type, (group) => (
    <Tooltip
      title={Messages.groupTypeTitles[getGroupType(group)]}
      describeChild
    >
      <span>{getGroupTypeLabel(group)}</span>
    </Tooltip>
  )),
  column('engine', Messages.columns.engine, (group) =>
    getEngineLabel(group.engine)
  ),
  column(
    'alerts',
    Messages.columns.alerts,
    (group) => <AlertsBadge alerts={group.alerts} />,
    Messages.columns.alertsDescription
  ),
  column(
    'instances',
    Messages.columns.instances,
    (group) => (
      <Typography
        component="span"
        variant="inherit"
        color={
          group.exportersConnected < group.monitoredInstances
            ? 'warning.main'
            : undefined
        }
      >
        {Messages.instances(group.exportersConnected, group.monitoredInstances)}
      </Typography>
    ),
    Messages.columns.instancesDescription
  ),
  column('writers', Messages.columns.writers, (group) => {
    const { text, state } = getWriters(group);

    return (
      <Typography
        component="span"
        variant="inherit"
        color={WRITERS_COLORS[state]}
      >
        {text}
      </Typography>
    );
  }),
  column(
    'throughput',
    Messages.columns.throughput,
    (group) => formatThroughput(group.throughput, group.engine),
    Messages.columns.throughputDescription
  ),
  column(
    'storage',
    Messages.columns.storage,
    (group) => formatStorage(group.storage),
    Messages.columns.storageDescription
  ),
];

// TanStack's expanded state is either `true` (expand all) or the expanded row IDs; the rows the
// user toggled override the ones the search expanded.
const useGroupExpansion = (
  groups: InstanceGroup[],
  autoExpanded: Set<string>,
  expansionKey: string
) => {
  const [expansion, setExpansion] = useState({
    key: expansionKey,
    overrides: {} as Record<string, boolean>,
  });
  const overrides = expansion.key === expansionKey ? expansion.overrides : {};
  const expanded = Object.fromEntries(
    groups
      .filter((group) => overrides[group.key] ?? autoExpanded.has(group.key))
      .map((group) => [group.key, true])
  );

  const onExpandedChange = (updater: MRT_Updater<MRT_ExpandedState>) => {
    const next = typeof updater === 'function' ? updater(expanded) : updater;

    setExpansion({
      key: expansionKey,
      overrides: {
        ...overrides,
        ...Object.fromEntries(
          groups.map((group) => [group.key, next === true || !!next[group.key]])
        ),
      },
    });
  };

  return { expanded, onExpandedChange };
};

export const GroupsTable: FC<GroupsTableProps> = ({
  groups,
  autoExpanded,
  expansionKey,
}) => {
  const [sorting, setSorting] = useState<MRT_SortingState>([]);
  const { expanded, onExpandedChange } = useGroupExpansion(
    groups,
    autoExpanded,
    expansionKey
  );
  const data = useMemo(() => sortGroups(groups, sorting), [groups, sorting]);

  return (
    <Table
      tableName="amazon-rds-groups"
      columns={COLUMNS}
      data={data}
      getRowId={(group) => group.key}
      // Sorted here, so missing values stay last in both directions.
      manualSorting
      state={{ sorting, expanded }}
      onSortingChange={setSorting}
      onExpandedChange={onExpandedChange}
      renderDetailPanel={({ row }) => <InstancesTable group={row.original} />}
      // MRT names every toggle "Expand" whatever its state, so the names say what a click does.
      muiExpandAllButtonProps={({ table }) => ({
        'aria-label': table.getIsAllRowsExpanded()
          ? Messages.collapseAll
          : Messages.expandAll,
        sx: EXPAND_BUTTON_SX,
      })}
      muiExpandButtonProps={({ row }) => ({
        'aria-expanded': row.getIsExpanded(),
        'aria-label': row.getIsExpanded()
          ? Messages.collapseGroup(row.original.name)
          : Messages.expandGroup(row.original.name),
        sx: EXPAND_BUTTON_SX,
      })}
      // A plain table sized to its content, so the ten columns fit beside the menu: MRT gives
      // every column a fixed width otherwise.
      layoutMode="semantic"
      muiTableHeadCellProps={{ sx: HEAD_CELL_SX }}
      muiTableBodyCellProps={{ sx: BODY_CELL_SX }}
      enablePagination={false}
      enableTopToolbar={false}
      enableBottomToolbar={false}
      enableColumnActions={false}
      enableColumnFilters={false}
      enableGlobalFilter={false}
      enableHiding={false}
      enableStickyHeader
      muiTableContainerProps={{
        sx: {
          borderRadius: '8px',
          border: '1px solid',
          borderColor: 'divider',
        },
      }}
    />
  );
};
