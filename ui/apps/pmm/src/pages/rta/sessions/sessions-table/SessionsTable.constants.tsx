import { type MRT_ColumnDef } from 'material-react-table';
// import { RealTimeSession } from 'types/rta.types';
import { Messages } from './SessionsTable.messages';
import { SessionStatus } from './session-status';
import { SessionRow } from './SessionsTable.types';

export const SESSIONS_TABLE_COLUMNS: MRT_ColumnDef<SessionRow>[] = [
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
        return rowA.original.startTime.localeCompare(rowB.original.startTime);
      }

      return rowA.original.status.localeCompare(rowB.original.status);
    },
  },
];
