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
}

const OverviewTable: FC<Props> = ({ queries, onQuerySelected, actions }) => (
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
    />
  </RealtimeTableWrapper>
);

export default OverviewTable;
