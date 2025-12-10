import Stack from '@mui/material/Stack';
import { Table } from 'components/table';
import { FC } from 'react';
import {
  REAL_TIME_TABLE_COLUMNS,
  REAL_TIME_TABLE_MOCK_DATE,
} from './RealTimeTable.constants';
import { Messages } from './RealTimeTable.messages';
import { RealTimeTableProps } from './RealTimeTable.types';

const RealTimeTable: FC<RealTimeTableProps> = ({
  showFilters,
  setShowFilters,
}) => (
  <Stack
    sx={{
      px: 2,
    }}
  >
    <Table
      tableName="real-time-queries"
      noDataMessage={Messages.noData}
      columns={REAL_TIME_TABLE_COLUMNS}
      state={{
        showColumnFilters: showFilters,
      }}
      enableTopToolbar={false}
      onShowColumnFiltersChange={() => setShowFilters(!showFilters)}
      data={REAL_TIME_TABLE_MOCK_DATE}
    />
  </Stack>
);

export default RealTimeTable;
