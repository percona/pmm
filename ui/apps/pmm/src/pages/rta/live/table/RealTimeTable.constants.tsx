import Typography from '@mui/material/Typography';
import { QueryCell } from './query-cell';
import { StateCell } from './state-cell';
import { RealTimeQuery } from 'types/real-time.types';
import { type MRT_ColumnDef } from 'material-react-table';
import { Messages } from './RealTimeTable.messages';

export const REAL_TIME_TABLE_MOCK_DATA: RealTimeQuery[] = Array.from(
  { length: 100 },
  (_, idx) => ({
    query: idx + `\tdb.logs.aggregate([
  {
    $group: {
      _id: '$ip_address',
      count: {
        $sum: 1
      }
    }
  },
  {
    $sort: {
      count: -1
    }
  }
])`,
    service: 'mc-analytics-s02-primary',
    duration: '1987 ms',
    state: 'Blocked',
  })
);

export const REAL_TIME_TABLE_COLUMNS: MRT_ColumnDef<RealTimeQuery>[] = [
  {
    header: Messages.queryText,
    accessorKey: 'query',
    Cell: ({ row }) => <QueryCell query={row.original.query} />,
  },
  {
    muiTableHeadCellProps: {
      sx: {
        width: 250,
        minWidth: 250,
        maxWidth: 250,
      },
    },
    muiTableBodyCellProps: {
      sx: {
        width: 250,
        minWidth: 250,
        maxWidth: 250,
      },
    },
    header: Messages.service,
    accessorKey: 'service',
  },
  {
    header: Messages.elapsedTime,
    accessorKey: 'duration',
    Cell: ({ row }) => (
      <Typography align="right">{row.original.duration}</Typography>
    ),
    muiTableHeadCellProps: {
      sx: {
        width: 176,
        minWidth: 176,
        maxWidth: 176,
      },
    },
    muiTableBodyCellProps: {
      sx: {
        width: 176,
        minWidth: 176,
        maxWidth: 176,
      },
    },
  },
  {
    header: Messages.state,
    accessorKey: 'state',
    Cell: ({ row }) => <StateCell state={row.original.state} />,
    muiTableHeadCellProps: {
      sx: {
        width: 176,
        minWidth: 176,
        maxWidth: 176,
      },
    },
    muiTableBodyCellProps: {
      sx: {
        width: 176,
        minWidth: 176,
        maxWidth: 176,
      },
    },
  },
];
