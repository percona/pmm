import Typography from '@mui/material/Typography';
import { QueryCell } from './query-cell';
import { StateCell } from './state-cell';
import { RealTimeQuery } from 'types/real-time.types';
import { type MRT_ColumnDef } from 'material-react-table';
import { Messages } from './RealTimeTable.messages';

export const REAL_TIME_TABLE_MOCK_DATE: RealTimeQuery[] = [
  {
    query: "db.orders.updateOne({ _id: ObjectId('...') }, { $push: { n...",
    service: 'mc-analytics-s02-primary',
    duration: '1987 ms',
    state: 'Blocked',
  },
  {
    query: "db.sessions.find({ 'user_id': 12345 })",
    service: 'mc-analytics-s01-primary',
    duration: '1876 ms',
    state: 'Running',
  },
  {
    query: "db.users.find({ country: 'USA' }).sort({ 'profile.last_log...",
    service: 'mc-analytics-s01-primary',
    duration: '1765 ms',
    state: 'Sorting result',
  },
  {
    query: "db.logs.aggregate([{ $group: { _id: '$ip_address', count:...",
    service: 'mc-analytics-s02-second...',
    duration: '1732 ms',
    state: 'Running',
  },
  {
    query: 'db.analytics.aggregate([{ $sort: { timestamp: -1 } }])',
    service: 'mc-analytics-s01-primary',
    duration: '1654 ms',
    state: 'Sorting result',
  },
  {
    query: "db.orders.updateMany({ state:'processing' }, { $set: { p...",
    service: 'mc-analytics-s02-primary',
    duration: '1543 ms',
    state: 'Blocked',
  },
  {
    query: "db.articles.find().sort({ 'published_at': -1 }).skip(50000...",
    service: 'mc-analytics-s01-primary',
    duration: '1321 ms',
    state: 'Running',
  },
  {
    query: "db.sessions.deleteMany({ 'lastActivity': { $lt: new Date('...",
    service: 'mc-analytics-s03-primary',
    duration: '1210 ms',
    state: 'Running',
  },
  {
    query: "db.user_events.find({ event_type: 'login' }).limit(10)",
    service: 'mc-analytics-s01-primary',
    duration: '1109 ms',
    state: 'Running',
  },
  {
    query: "db.analytics.aggregate([{ $match: { event: 'pageView' } },...",
    service: 'mc-analytics-s01-primary',
    duration: '987 ms',
    state: 'Waiting',
  },
];

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
