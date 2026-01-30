import { MaterialReactTableProps } from 'material-react-table';
import { Table } from '@percona/ui-lib';
import { FC } from 'react';
import { QueryData } from 'types/rta.types';
import { OVERVIEW_TABLE_COLUMNS } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import { Icon } from 'components/icon';
import Stack from '@mui/material/Stack';
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
      enableRowActions
      renderRowActions={({ row }) => (
        <Stack
          className="row-actions"
          justifyContent="center"
          alignItems="center"
          sx={{
            flex: 1,
            height: '100%',
            display: 'none',
          }}
        >
          <IconButton
            color="inherit"
            data-testid="open-query-details"
            aria-label={Messages.actions.openDetails}
            onClick={() => onQuerySelected(row.original, row.index)}
          >
            <Icon name="bottom-panel-open" />
          </IconButton>
        </Stack>
      )}
      // Show the row actions only on hover
      muiTableBodyRowProps={{
        sx: {
          '&:hover .row-actions': {
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          },
        },
      }}
      displayColumnDefOptions={{
        'mrt-row-actions': {
          header: '',
          size: 56,
        },
      }}
    />
  </RealtimeTableWrapper>
);

export default OverviewTable;
