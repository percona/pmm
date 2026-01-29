import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { StateCell } from './state-cell';
import { QueryCell } from './query-cell';

export const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    minSize: 500,
    header: Messages.queryText,
    accessorKey: 'queryText',
    Cell: ({ row }) => <QueryCell query={row.original.queryText} />,
  },
  {
    header: Messages.service,
    accessorKey: 'serviceName',
  },
  {
    header: Messages.elapsedTime,
    accessorKey: 'executionDuration',
  },
  {
    header: Messages.state,
    accessorKey: 'state',
    Cell: ({ row }) => <StateCell state={row.original.state} />,
  },
];

export const sampleQueries: QueryData[] = Array.from({ length: 25 }).map(
  (_, i) => {
    const isMongo = i % 3 === 0 || true; // Every 3rd entry is MongoDB
    const states = ['RUNNING', 'SUCCESS', 'SUCCESS', 'FAILED', 'KILLED'];
    const services = ['mongo1', 'mongo2', 'mongo3', 'mongo4'];

    return {
      serviceId: `svc-${(i % 4) + 1}`,
      serviceName: services[i % 4],
      queryId: `uuid-${1000 + i}`,
      queryText: isMongo
        ? `db.collection('orders').find({ status: 'active' }).limit(${10 + i})`
        : `SELECT * FROM users WHERE last_login > '2023-01-0${(i % 9) + 1}' LIMIT ${20 + i};`,
      state: states[i % states.length],
      executionDuration: `${(Math.random() * 2).toFixed(2)}s`,
      rowsExamined: Math.floor(Math.random() * 10000),
      rowsSent: Math.floor(Math.random() * 500),
      collectTime: new Date(Date.now() - i * 1000 * 60).toISOString(),
      rawQueryJson: JSON.stringify({ plan: 'IndexScan', cost: i * 1.5 }),
      // Optional MongoDB payload
      ...(isMongo && {
        mongoDbPayload: {
          opid: `op-${5000 + i}`,
          client: `192.168.1.${100 + i}:5432`,
          waitingForLock: i % 10 === 0,
          indexUtilized: i % 6 === 0 ? 'none' : 'idx_user_status_1',
        },
      }),
    };
  }
);
