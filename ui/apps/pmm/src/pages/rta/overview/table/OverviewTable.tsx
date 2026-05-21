import {
  type MRT_ColumnFiltersState,
  type MRT_Row,
  type MRT_TableInstance,
  MaterialReactTableProps,
} from 'material-react-table';
import { Table } from '@percona/percona-ui';
import { FC, useEffect, useRef, useState } from 'react';
import { QueryData } from 'types/rta.types';
import { OVERVIEW_TABLE_COLUMNS } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import { Messages } from './OverviewTable.messages';
import { filterElapsedTime } from './OverviewTable.utils';

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
  const tableRef = useRef<MRT_TableInstance<QueryData> | null>(null);
  // Controlled filter state is required to read the filtered row model via tableInstanceRef.
  const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
  const [globalFilter, setGlobalFilter] = useState('');

  // Pre-pagination so navigation covers all filtered rows, not only the current page.
  const getNavigableQueries = () =>
    tableRef.current?.getPrePaginationRowModel().rows.map((row) => row.original) ??
    queries;

  useEffect(() => {
    onNavigableQueriesChange(getNavigableQueries());
  }, [queries, columnFilters, globalFilter, onNavigableQueriesChange]);

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
        columns={OVERVIEW_TABLE_COLUMNS}
        data={queries}
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
        state={{ columnFilters, globalFilter }}
        onColumnFiltersChange={setColumnFilters}
        onGlobalFilterChange={setGlobalFilter}
        enableGlobalFilter={true}
        enableHiding={false}
        enableRowHoverAction
        tableInstanceRef={tableRef}
        rowHoverAction={(row) => onQuerySelected(row.original)}
        renderTopToolbarCustomActions={actions}
        filterFns={{
          // default 'betweenInclusive' filter fails on values like '1.50', discarding the row that has 1.5 seconds
          timeRangeFilterFn: (row, id, filterValue) =>
            filterElapsedTime(row as MRT_Row<QueryData>, id, filterValue),
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
