import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { StateCell } from './state-cell';
import { QueryCell } from './query-cell';
import { parseDuration } from 'utils/duration.utils';

export const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    minSize: 500,
    header: Messages.columns.queryText,
    accessorKey: 'queryText',
    Cell: ({ row }) => <QueryCell query={row.original.queryText} />,
  },
  {
    header: Messages.columns.service,
    accessorKey: 'serviceName',
  },
  {
    header: Messages.columns.elapsedTime,
    accessorKey: 'executionDuration',
    filterVariant: 'range',
    filterFn: 'between',
    sortingFn: (rowA, rowB) =>
      parseDuration(rowA.original.executionDuration) -
      parseDuration(rowB.original.executionDuration),
  },
  {
    header: Messages.columns.state,
    accessorKey: 'state',
    Cell: ({ row }) => <StateCell state={row.original.state} />,
  },
];
