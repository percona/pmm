import { Table } from 'components/table';
import { FC } from 'react';
import {
  REAL_TIME_TABLE_COLUMNS,
  REAL_TIME_TABLE_MOCK_DATA,
} from './RealTimeTable.constants';
import { Messages } from './RealTimeTable.messages';
import { RealTimeTableProps } from './RealTimeTable.types';

const RealTimeTable: FC<RealTimeTableProps> = ({
  showFilters,
  setShowFilters,
  selectedQuery,
  setQuery,
}) => (
  <Table
    tableName="real-time-queries"
    getRowId={(row) => row.query}
    noDataMessage={Messages.noData}
    columns={REAL_TIME_TABLE_COLUMNS}
    state={{
      showColumnFilters: showFilters,
    }}
    initialState={{
      pagination: {
        pageSize: 25,
        pageIndex: 0,
      },
    }}
    enableStickyHeader={true}
    enableTopToolbar={false}
    onShowColumnFiltersChange={() => setShowFilters(!showFilters)}
    data={REAL_TIME_TABLE_MOCK_DATA}
    enableRowHoverAction
    rowHoverAction={(row) => setQuery(row.original)}
    // rowCount={25}
  />
);
export default RealTimeTable;
