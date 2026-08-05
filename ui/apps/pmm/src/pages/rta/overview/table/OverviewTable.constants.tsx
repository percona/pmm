import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { QueryCell } from './query-cell';
import UnavailableText from 'components/unavailable-text';
import {
  formatElapsedTime,
  queryDatabaseName,
  queryLanguage,
  queryUsername,
  UNAVAILABLE_VALUE,
} from './OverviewTable.utils';

export const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    size: 500,
    header: Messages.columns.queryText,
    accessorKey: 'queryText',
    filterFn: 'contains',
    Cell: ({ row }) => (
      <QueryCell
        query={row.original.queryText}
        language={queryLanguage(row.original)}
      />
    ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-query-text-cell`,
    }),
  },
  {
    header: Messages.columns.host,
    accessorKey: 'serviceName',
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-host-cell`,
    }),
  },
  {
    header: Messages.columns.database,
    id: 'databaseName',
    accessorFn: queryDatabaseName,
    filterFn: 'commaSeparatedFilterFn',
    Cell: ({ cell }) =>
      cell.getValue<string>() === UNAVAILABLE_VALUE ? (
        <UnavailableText />
      ) : (
        cell.getValue<string>()
      ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-database-cell`,
    }),
  },
  {
    header: Messages.columns.user,
    id: 'username',
    accessorFn: queryUsername,
    filterFn: 'commaSeparatedFilterFn',
    Cell: ({ cell }) =>
      cell.getValue<string>() === UNAVAILABLE_VALUE ? (
        <UnavailableText />
      ) : (
        cell.getValue<string>()
      ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-user-cell`,
    }),
  },
  {
    header: Messages.columns.operationId,
    accessorKey: 'queryId',
    enableColumnFilter: false,
    enableSorting: false,
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-operation-id-cell`,
    }),
  },
  {
    header: Messages.columns.elapsedTime,
    accessorKey: 'queryExecutionDurationMs',
    // Pinned to the right edge, so every pixel here is taken from the query
    // text; the compact value format is what keeps the column this narrow.
    size: 120,
    filterVariant: 'range',
    filterFn: 'timeRangeFilterFn',
    muiTableHeadCellFilterTextFieldProps: {
      inputProps: { step: 0.25, type: 'number' },
    },
    // A statement that has just started reports 0, which is a duration like any
    // other; only a missing value is unavailable.
    Cell: ({ cell }) =>
      cell.getValue<number | null>() == null ? (
        <UnavailableText />
      ) : (
        formatElapsedTime(cell.getValue<number>())
      ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-elapsed-time-cell`,
    }),
  },
];
