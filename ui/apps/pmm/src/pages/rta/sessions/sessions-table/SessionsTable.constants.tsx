import { type MRT_ColumnDef } from 'material-react-table';
import { RealTimeSession } from 'types/rta.types';
import { Messages } from './SessionsTable.messages';
import { SessionStatus } from './session-status';

export const SESSIONS_TABLE_COLUMNS: MRT_ColumnDef<RealTimeSession>[] = [
  {
    accessorKey: 'sessionName',
    header: Messages.table.columns.sessionName,
  },
  {
    accessorKey: 'status',
    Cell: ({ row }) => <SessionStatus session={row.original} />,
    header: Messages.table.columns.status,
    sortingFn: (rowA, rowB) => {
      if (rowA.original.status === rowB.original.status) {
        return rowA.original.startedAt.localeCompare(rowB.original.startedAt);
      }

      return rowA.original.status.localeCompare(rowB.original.status);
    },
  },
];
