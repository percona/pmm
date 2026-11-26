import type { MRT_ColumnDef } from 'material-react-table';
import type { Dump } from 'types/dump.types';
import { CreatedAt } from './components/created-at';
import { DataWindow } from './components/data-window';
import { DumpServices } from './components/dump-services';
import { DumpStatus } from './components/dump-status';
import { Messages } from './DumpPage.messages';

export const getDumpColumns = (
  onShowLogs: (dump: Dump) => void
): MRT_ColumnDef<Dump>[] => [
  {
    accessorKey: 'dumpId',
    header: Messages.columns.id,
  },
  {
    accessorKey: 'status',
    header: Messages.columns.status,
    Cell: ({ row: { original } }) => (
      <DumpStatus dump={original} onShowLogs={() => onShowLogs(original)} />
    ),
  },
  {
    accessorKey: 'serviceNames',
    header: Messages.columns.services,
    enableSorting: false,
    Cell: ({ row: { original } }) => (
      <DumpServices serviceNames={original.serviceNames} />
    ),
  },
  {
    accessorKey: 'startTime',
    header: Messages.columns.dataFrom,
    Cell: ({ row: { original } }) => <DataWindow dump={original} />,
  },
  {
    accessorKey: 'createdAt',
    header: Messages.columns.created,
    Cell: ({ row: { original } }) => <CreatedAt value={original.createdAt} />,
  },
];
