import { FC, useCallback, useEffect, useRef } from 'react';
import { type MRT_Row, type MRT_TableInstance, MaterialReactTableProps } from 'material-react-table';
import { Table } from '@percona/percona-ui';
import { QueryData } from 'types/rta.types';
import { OVERVIEW_TABLE_COLUMNS } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import { Messages } from './OverviewTable.messages';
import { filterElapsedTime } from './OverviewTable.utils';
import { useTableUrlState } from 'hooks/utils/useTableUrlState';

const OVERVIEW_TABLE_URL_STATE_OPTIONS = {
  paramPrefix: 'overview',
  defaults: {
    pagination: { pageIndex: 0, pageSize: 25 },
  }
};

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
  const { tableProps } = useTableUrlState(OVERVIEW_TABLE_URL_STATE_OPTIONS);
  const { columnFilters, sorting } = tableProps.state;

  // Pre-pagination so navigation covers all filtered rows, not only the current page.
  const getNavigableQueries = useCallback(
    () =>
      tableRef.current?.getPrePaginationRowModel().rows.map((row) => row.original) ??
      queries,
    [queries]
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
        {...tableProps}
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
        enableStickyHeader
        enableGlobalFilter={false}
        enableHiding={false}
        enableRowHoverAction
        tableInstanceRef={tableRef}
        rowHoverAction={(row) => {
          syncNavigableQueries();
          onQuerySelected(row.original as QueryData);
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
