import {
  type MRT_ColumnDef,
  type MRT_Row,
  type MRT_VisibilityState,
  type MaterialReactTableProps,
} from 'material-react-table';
import { Table, useNavigableRows } from '@percona/peak-ui';
import { useMemo, useState, type FC } from 'react';
import type { QueryData } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import { getOverviewTableColumns } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import { Messages } from './OverviewTable.messages';
import { filterCommaSeparated, filterElapsedTime } from './OverviewTable.utils';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';

const OVERVIEW_TABLE_URL_STATE_OPTIONS = {
  paramPrefix: 'overview',
  defaults: {
    pagination: { pageIndex: 0, pageSize: 25 },
  },
};

// Database and User are opt-in columns: showing them by default pushes the
// query text and Elapsed time out of view, so they start hidden and users
// reveal the ones they need from the Show/Hide columns menu.
// The visibility state is held here rather than in initialState because the
// peak-ui Table controls columnVisibility from its own localStorage state,
// which cannot express a column that is hidden by default.
const DEFAULT_COLUMN_VISIBILITY: MRT_VisibilityState = {
  databaseName: false,
  username: false,
};

// Pinning stays enabled for the sticky rendering of Elapsed time - MRT only
// draws a pinned column's background while the feature is on - but no column may
// be pinned by hand, which takes the pin buttons out of the column and
// Show/Hide menus. Hoisted so the reference is stable: MRT memoizes the merged
// default column on it, and this table re-renders on every poll.
const DEFAULT_COLUMN: Partial<MRT_ColumnDef<QueryData>> = {
  enablePinning: false,
};

// Elapsed time is the key live metric; keep it visible even when the other
// columns overflow into a horizontal scroll. Held as controlled state rather
// than initialState so the Show/Hide menu's "Unpin all" - which MRT renders
// unconditionally while pinning is enabled - cannot take it away.
const COLUMN_PINNING = { right: ['queryExecutionDurationMs'] };

interface Props {
  queries: QueryData[];
  // Technology of the services being watched; the selection cannot mix them.
  serviceType?: ServiceType;
  onQuerySelected: (query: QueryData) => void;
  onNavigableQueriesChange: (queries: QueryData[]) => void;
  actions?: MaterialReactTableProps<QueryData>['renderTopToolbarCustomActions'];
  onRowHover?: () => void;
}

const OverviewTable: FC<Props> = ({
  queries,
  serviceType,
  onQuerySelected,
  onNavigableQueriesChange,
  actions,
  onRowHover,
}) => {
  const { tableProps: navigableTableProps, refresh } =
    useNavigableRows<QueryData>({
      data: queries,
      onChange: onNavigableQueriesChange,
    });
  const { tableProps: urlStateTableProps } = useTableUrlState(
    OVERVIEW_TABLE_URL_STATE_OPTIONS
  );
  const [columnVisibility, setColumnVisibility] = useState<MRT_VisibilityState>(
    DEFAULT_COLUMN_VISIBILITY
  );
  const columns = useMemo(
    () => getOverviewTableColumns(serviceType),
    [serviceType]
  );

  return (
    <RealtimeTableWrapper>
      <Table
        tableName="realtime-overview-table"
        columns={columns}
        data={queries}
        noDataMessage={Messages.noData}
        muiTopToolbarProps={{
          sx: {
            mb: 0.5,
            [`& > .${boxClasses.root}`]: {
              alignItems: 'flex-start',
              alignContent: 'flex-start',
              flexDirection: 'row-reverse',
            },
          },
        }}
        {...navigableTableProps}
        {...urlStateTableProps}
        // Both spreads above carry a `state`, so the merge has to be explicit:
        // the URL-driven pagination/sorting/filters would otherwise be dropped.
        state={{
          ...navigableTableProps.state,
          ...urlStateTableProps.state,
          columnVisibility,
          columnPinning: COLUMN_PINNING,
        }}
        onColumnVisibilityChange={setColumnVisibility}
        enableStickyHeader
        enableColumnPinning
        defaultColumn={DEFAULT_COLUMN}
        enableGlobalFilter={false}
        enableHiding
        enableRowHoverAction
        rowHoverAction={(row) => {
          refresh();
          onQuerySelected(row.original);
        }}
        renderTopToolbarCustomActions={actions}
        filterFns={{
          // comma-separated list of lazy (substring) matches for Database/User
          commaSeparatedFilterFn: (row, id, filterValue) =>
            filterCommaSeparated(row as MRT_Row<QueryData>, id, filterValue),
          // default 'betweenInclusive' filter fails on values like '1.50', discarding the row that has 1.5 seconds
          timeRangeFilterFn: (row, id, filterValue) =>
            filterElapsedTime(row as MRT_Row<QueryData>, id, filterValue),
        }}
        muiTableContainerProps={{
          sx: {
            flex: 1,
            // TODO: use theme.shape.borderRadiusMd (8px) once percona-ui
            // publishes the Shape tokens (percona-ui#37, not in 1.0.23)
            borderRadius: '8px',
            border: '1px solid',
            borderColor: 'divider',
          },
        }}
        muiTableBodyRowProps={({ row }) => ({
          onMouseEnter: onRowHover,
          'data-testid': `query-${row.original.queryId}-row`,
        })}
        muiTableBodyCellProps={{
          sx: { py: 1, px: 1 },
        }}
        muiTableHeadCellProps={{
          sx: { px: 1 },
        }}
      />
    </RealtimeTableWrapper>
  );
};

export default OverviewTable;
