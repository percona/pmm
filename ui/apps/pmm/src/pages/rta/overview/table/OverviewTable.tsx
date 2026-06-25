import {
  type MRT_ColumnFiltersState,
  type MRT_Row,
  type MRT_SortingState,
  type MRT_TableInstance,
  MaterialReactTableProps,
} from 'material-react-table';
import { Table } from '@percona/percona-ui';
import { FC, useCallback, useEffect, useRef, useState } from 'react';
import { QueryData } from 'types/rta.types';
import { getOverviewTableColumns } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import { Messages } from './OverviewTable.messages';
import { filterElapsedTime, prepareOverviewQueries } from './OverviewTable.utils';

interface Props {
  queries: QueryData[];
  onQuerySelected: (query: QueryData) => void;
  onNavigableQueriesChange: (queries: QueryData[]) => void;
  actions?: MaterialReactTableProps<QueryData>['renderTopToolbarCustomActions'];
  onRowHover?: () => void;
}

const OverviewTable: FC<Props> = ({
  queries,
  onQuerySelected,
  onNavigableQueriesChange,
  actions,
  onRowHover,
}) => {
  const tableQueries = prepareOverviewQueries(queries);
  const columns = getOverviewTableColumns(tableQueries);
  const tableRef = useRef<MRT_TableInstance<QueryData> | null>(null);
  // Controlled table state is required to read the filtered/sorted row model via tableInstanceRef.
  const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
  const [sorting, setSorting] = useState<MRT_SortingState>([]);

  // Pre-pagination so navigation covers all filtered rows, not only the current page.
  const getNavigableQueries = useCallback(
    () =>
      tableRef.current?.getPrePaginationRowModel().rows.map((row) => row.original) ??
      tableQueries,
    [tableQueries]
  );

  const syncNavigableQueries = useCallback(() => {
    onNavigableQueriesChange(getNavigableQueries());
  }, [getNavigableQueries, onNavigableQueriesChange]);

  useEffect(() => {
    syncNavigableQueries();
  }, [columnFilters, sorting, syncNavigableQueries]);

  return (
    <RealtimeTableWrapper>
      <Table
        tableName="realtime-overview-table"
        initialState={{
          pagination: {
            pageSize: 25,
            pageIndex: 0,
          },
        }}
        columns={columns}
        data={tableQueries}
        noDataMessage={Messages.noData}
        muiTopToolbarProps={{
          sx: {
            // vertically center the buttons
            [`& > .${boxClasses.root}`]: {
              alignItems: 'center',
              flexDirection: 'row-reverse',
            },
          },
        }}
        state={{ columnFilters, sorting }}
        onColumnFiltersChange={setColumnFilters}
        onSortingChange={setSorting}
        enableStickyHeader
        enableGlobalFilter={false}
        enableHiding={false}
        enableRowHoverAction
        tableInstanceRef={tableRef}
        rowHoverAction={(row) => {
          syncNavigableQueries();
          onQuerySelected(row.original);
        }}
        renderTopToolbarCustomActions={actions}
        filterFns={{
          // default 'betweenInclusive' filter fails on values like '1.50', discarding the row that has 1.5 seconds
          timeRangeFilterFn: (row, id, filterValue) =>
            filterElapsedTime(row as MRT_Row<QueryData>, id, filterValue),
        }}
        muiTableContainerProps={{
          sx: {
            flex: 1,
            borderRadius: 2,
            border: '1px solid',
            borderColor: 'divider',
          },
        }}
        muiTableBodyRowProps={({ row }) => ({
          onMouseEnter: onRowHover,
          'data-testid': `query-${row.original.queryId}-row`,
        })}
      />
    </RealtimeTableWrapper>
  );
};

export default OverviewTable;
