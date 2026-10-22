import { type MRT_ColumnDef } from 'material-react-table';
import { Messages } from './SessionsTable.messages';
import { SessionStatus } from './cell-session-status';
import { SessionRow } from './SessionsTable.types';
import { SessionName } from './cell-session-name';
import { Technology, technologyLabel } from 'pages/rta/components/technology';

// Sorts and filters on the displayed name rather than the SERVICE_TYPE_* value;
// an accessorFn column needs an explicit id.
const TECHNOLOGY_COLUMN: MRT_ColumnDef<SessionRow> = {
  id: 'serviceType',
  header: Messages.table.columns.technology,
  size: 140,
  accessorFn: (row) => technologyLabel(row.serviceType),
  Cell: ({ row }) => <Technology serviceType={row.original.serviceType} />,
};

const SESSION_NAME_COLUMN: MRT_ColumnDef<SessionRow> = {
  accessorKey: 'sessionName',
  header: Messages.table.columns.sessionName,
  Cell: ({ row }) => <SessionName session={row.original} />,
};

const STATUS_COLUMN: MRT_ColumnDef<SessionRow> = {
  accessorKey: 'status',
  Cell: ({ row }) => <SessionStatus session={row.original} />,
  header: Messages.table.columns.status,
  sortingFn: (rowA, rowB) => {
    if (rowA.original.status === rowB.original.status) {
      return rowA.original.startTime.localeCompare(rowB.original.startTime);
    }

    return rowA.original.status.localeCompare(rowB.original.status);
  },
};

// Module constant for a stable reference across the 5s poll.
export const SESSIONS_TABLE_COLUMNS: MRT_ColumnDef<SessionRow>[] = [
  SESSION_NAME_COLUMN,
  TECHNOLOGY_COLUMN,
  STATUS_COLUMN,
];

// MRT orders columns by the explicit id, falling back to the accessor key.
export const columnId = (column: MRT_ColumnDef<SessionRow>): string =>
  column.id ?? column.accessorKey ?? '';
