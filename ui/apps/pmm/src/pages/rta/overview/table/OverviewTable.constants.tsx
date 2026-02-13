import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { QueryCell } from './query-cell';
import { parseDuration } from 'utils/duration.utils';

export const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    minSize: 400,
    header: Messages.columns.queryText,
    accessorKey: 'queryText',
    Cell: ({ row }) => <QueryCell query={row.original.queryText} />,
  },
  {
    header: Messages.columns.service,
    accessorKey: 'serviceName',
  },
  {
    size: 100,
    header: Messages.columns.elapsedTime,
    accessorKey: 'queryExecutionDuration',
    filterVariant: 'range',
    filterFn: 'between',
    sortingFn: (rowA, rowB) =>
      parseDuration(rowA.original.queryExecutionDuration ?? '0s') -
      parseDuration(rowB.original.queryExecutionDuration ?? '0s'),
    Cell: ({ cell }) => `${cell.getValue() ? `${parseDuration(cell.getValue() as string).toFixed(2)} ms` : 'N/A'}`,
  },
];
