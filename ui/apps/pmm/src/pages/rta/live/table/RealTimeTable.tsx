import { Table } from 'components/table';
import { FC } from 'react';
import { REAL_TIME_TABLE_COLUMNS } from './RealTimeTable.constants';
import { Messages } from './RealTimeTable.messages';
import { RealTimeTableProps } from './RealTimeTable.types';
import { useTheme } from '@mui/material/styles';
import { tableRowClasses } from '@mui/material/TableRow';

const RealTimeTable: FC<RealTimeTableProps> = ({
  queries,
  showFilters,
  setShowFilters,
  selectedQuery,
  setQuery,
}) => {
  const theme = useTheme();

  return (
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
      data={queries}
      enableRowHoverAction
      rowHoverAction={(row) => setQuery(row.original, row.index)}
      muiTableBodyRowProps={({ row }) => ({
        sx: {
          td: {
            py: 1,
          },

          [`&.${tableRowClasses.root}`]: {
            borderWidth: 2,
            boxSizing: 'border-box',
            borderStyle: 'dashed',
            borderColor: 'transparent',

            ...(row.original.query === selectedQuery?.query && {
              borderColor: theme.palette.primary.light,
              backgroundColor: theme.palette.action.focus,

              td: {
                borderWidth: 0,
              },

              '&:hover > td': {
                backgroundColor: theme.palette.action.hover,
              },
            }),
          },
        },
      })}
    />
  );
};
export default RealTimeTable;
