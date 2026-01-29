import { Table } from '@percona/ui-lib';
import { FC, ReactElement } from 'react';
import { QueryData } from 'types/rta.types';
import { OVERVIEW_TABLE_COLUMNS } from './OverviewTable.constants';
import { RealtimeTableWrapper } from 'pages/rta/components/rta-table-wrapper';
import { boxClasses } from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import { Icon } from 'components/icon';
import { Messages } from './OverviewTable.messages';
import { Link as RouterLink } from 'react-router-dom';

interface Props {
  queries: QueryData[];
  onQuerySelected: (query: QueryData, idx: number) => void;
  controls?: ReactElement;
}

const OverviewTable: FC<Props> = ({
  queries,
  onQuerySelected,
  controls: Controls,
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
      renderTopToolbarCustomActions={() => (
        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          sx={{
            flex: 1,
          }}
        >
          <Stack>{Controls}</Stack>
          <Button
            color="inherit"
            data-testid="open-new-modal"
            startIcon={<Icon name="dynamic-feed" />}
            component={RouterLink}
            to="/rta/sessions?fromOverview=true"
          >
            {Messages.allSessions}
          </Button>
        </Stack>
      )}
    />
  </RealtimeTableWrapper>
);

export default OverviewTable;
