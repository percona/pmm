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

// Hidden by default to keep the query text in view; held as state because
// peak-ui's stored columnVisibility cannot express hidden-by-default.
const DEFAULT_COLUMN_VISIBILITY: MRT_VisibilityState = {
  databaseName: false,
  username: false,
};

// MRT draws a pinned column's background only with pinning on, so it stays on
// but nothing is pinnable by hand. Hoisted: MRT memoizes the default column on it.
const DEFAULT_COLUMN: Partial<MRT_ColumnDef<QueryData>> = {
  enablePinning: false,
};

// Controlled rather than initialState so the Show/Hide menu's "Unpin all"
// cannot unpin Elapsed time.
const COLUMN_PINNING = { right: ['queryExecutionDurationMs'] };

interface Props {
  queries: QueryData[];
  // Technology of the services being watched; the selection cannot mix them.
  serviceType?: ServiceType;
  // Replaces the default empty-state text, e.g. with why a session failed.
  noDataMessage?: string;
  onQuerySelected: (query: QueryData) => void;
  onNavigableQueriesChange: (queries: QueryData[]) => void;
  actions?: MaterialReactTableProps<QueryData>['renderTopToolbarCustomActions'];
  onRowHover?: () => void;
}

const OverviewTable: FC<Props> = ({
  queries,
  serviceType,
  noDataMessage = Messages.noData,
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
        noDataMessage={noDataMessage}
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
        // Both spreads carry a `state`; merged explicitly so neither is dropped.
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
