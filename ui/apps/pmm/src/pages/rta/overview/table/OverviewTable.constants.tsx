import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { StateCell } from './state-cell';
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
    accessorKey: 'executionDuration',
    filterVariant: 'range',
    filterFn: 'between',
    sortingFn: (rowA, rowB) =>
      parseDuration(rowA.original.executionDuration) -
      parseDuration(rowB.original.executionDuration),
    Cell: ({ cell }) => `${parseDuration(cell.getValue() as string).toFixed(5)} ms`,
  },
  {
    header: Messages.columns.state,
    accessorKey: 'state',
    size: 100,
    Cell: ({ row }) => <StateCell state={row.original.state} />,
  },
];
