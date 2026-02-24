import { MaterialReactTableProps } from 'material-react-table';
import { Table } from '@percona/ui-lib';
import { FC } from 'react';
import { QueryData } from 'types/rta.types';
import { OVERVIEW_TABLE_COLUMNS } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import { Messages } from './OverviewTable.messages';

interface Props {
  queries: QueryData[];
  onQuerySelected: (query: QueryData, idx: number) => void;
  actions?: MaterialReactTableProps<QueryData>['renderTopToolbarCustomActions'];
  onRowHover?: () => void;
}

const OverviewTable: FC<Props> = ({
  queries,
  onQuerySelected,
  actions,
  onRowHover,
}) => (
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
      enableGlobalFilter={false}
      enableHiding={false}
      enableRowHoverAction
      rowHoverAction={(row) => onQuerySelected(row.original, row.index)}
      renderTopToolbarCustomActions={actions}
      filterFns={{
        // default 'betweenInclusive' filter fails on values like '1.50', discarding the row that has 1.5 seconds
        timeRangeFilterFn: (row, id, filterValue) => {
          const [min, max] = filterValue;
          if (
            min === '' ||
            max === '' ||
            min === null ||
            max === null ||
            min === undefined ||
            max === undefined
          ) {
            return true;
          }

          if (Number.isNaN(min) || Number.isNaN(max)) {
            return false;
          }

          const minSeconds = parseFloat(min);
          const maxSeconds = parseFloat(max);

          const valueSeconds = row.getValue<number>(id);
          if (valueSeconds === null || valueSeconds === undefined) {
            return false;
          }

          return valueSeconds >= minSeconds && valueSeconds <= maxSeconds;
        },
      }}
      muiTableBodyRowProps={{
        onMouseEnter: onRowHover,
      }}
    />
  </RealtimeTableWrapper>
);

export default OverviewTable;
