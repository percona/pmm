import { type MRT_ColumnDef } from 'material-react-table';
import { Messages } from './SessionsTable.messages';
import { SessionStatus } from './cell-session-status';
import { SessionRow } from './SessionsTable.types';
import { SessionName } from './cell-session-name';
import { Technology, technologyLabel } from 'pages/rta/components/technology';

// The accessor returns the displayed name so sorting and filtering work on that
// rather than on the SERVICE_TYPE_* enum value; the id is set explicitly because
// an accessorFn column has no accessorKey to derive it from.
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

// A module constant so the reference is stable; the sessions table renders on a
// 5s poll.
export const SESSIONS_TABLE_COLUMNS: MRT_ColumnDef<SessionRow>[] = [
  SESSION_NAME_COLUMN,
  TECHNOLOGY_COLUMN,
  STATUS_COLUMN,
];

// columnId is what MRT orders columns by: the explicit id when a column has one,
// the accessor key otherwise.
export const columnId = (column: MRT_ColumnDef<SessionRow>): string =>
  column.id ?? column.accessorKey ?? '';
