import type { FC } from 'react';
import { type MaterialReactTableProps,type MRT_Row } from 'material-react-table';
import { Table, useNavigableRows } from '@percona/peak-ui';
import type { QueryData } from 'types/rta.types';
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
  },
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
  const { tableProps: navigableTableProps, refresh } =
    useNavigableRows<QueryData>({
      data: queries,
      onChange: onNavigableQueriesChange,
    });
  const { tableProps: urlStateTableProps } = useTableUrlState(
    OVERVIEW_TABLE_URL_STATE_OPTIONS
  );

  return (
    <RealtimeTableWrapper>
      <Table
        tableName="realtime-overview-table"
        columns={OVERVIEW_TABLE_COLUMNS}
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
