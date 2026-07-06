import { FC } from 'react';
import { type MRT_Row, MaterialReactTableProps } from 'material-react-table';
import { Table, useNavigableRows } from '@percona/percona-ui';
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
  const { tableProps: navigableTableProps, refresh } = useNavigableRows<QueryData>({
    data: queries,
    onChange: onNavigableQueriesChange,
  });
  const { tableProps: urlStateTableProps } = useTableUrlState(OVERVIEW_TABLE_URL_STATE_OPTIONS);

  return (
    <RealtimeTableWrapper>
      <Table
        tableName="realtime-overview-table"
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
        {...navigableTableProps}
        {...urlStateTableProps}
        enableStickyHeader
        enableGlobalFilter={false}
        enableHiding={false}
        enableRowHoverAction
        rowHoverAction={(row) => {
          refresh();
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
