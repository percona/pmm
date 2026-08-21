import { type MRT_ColumnDef } from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
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

const QUERY_TEXT_COLUMN: MRT_ColumnDef<QueryData> = {
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
};

const HOST_COLUMN: MRT_ColumnDef<QueryData> = {
  header: Messages.columns.host,
  accessorKey: 'serviceName',
  // without this the column falls back to MRT's 'fuzzy' default, which
  // matches any host containing the typed characters in order
  filterFn: 'contains',
  // @ts-expect-error - muiTableBodyCellProps is not typed correctly
  muiTableBodyCellProps: ({ row }) => ({
    'data-testid': `query-${row.original.queryId}-host-cell`,
  }),
};

const DATABASE_COLUMN: MRT_ColumnDef<QueryData> = {
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
};

const USER_COLUMN: MRT_ColumnDef<QueryData> = {
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
};

const OPERATION_ID_COLUMN: MRT_ColumnDef<QueryData> = {
  header: Messages.columns.operationId,
  accessorKey: 'queryId',
  enableColumnFilter: false,
  enableSorting: false,
  // @ts-expect-error - muiTableBodyCellProps is not typed correctly
  muiTableBodyCellProps: ({ row }) => ({
    'data-testid': `query-${row.original.queryId}-operation-id-cell`,
  }),
};

const ELAPSED_TIME_COLUMN: MRT_ColumnDef<QueryData> = {
  header: Messages.columns.elapsedTime,
  accessorKey: 'queryExecutionDurationMs',
  // Pinned to the right edge, so every pixel here is taken from the query
  // text; the compact value format is what keeps the column this narrow.
  size: 120,
  filterVariant: 'range',
  filterFn: 'timeRangeFilterFn',
  muiFilterTextFieldProps: {
    type: 'text',
    inputProps: { inputMode: 'decimal' },
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
};

// Both sets are module constants so the reference stays stable across the
// polling re-renders of the overview.
const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  QUERY_TEXT_COLUMN,
  HOST_COLUMN,
  OPERATION_ID_COLUMN,
  ELAPSED_TIME_COLUMN,
];

// Database and User are MySQL-only. MongoDB reports values for both, but they
// carry little meaning there (admin/local, __system), so they are not offered
// when MongoDB services are being watched.
const OVERVIEW_TABLE_COLUMNS_MYSQL: MRT_ColumnDef<QueryData>[] = [
  QUERY_TEXT_COLUMN,
  HOST_COLUMN,
  DATABASE_COLUMN,
  USER_COLUMN,
  OPERATION_ID_COLUMN,
  ELAPSED_TIME_COLUMN,
];

export const getOverviewTableColumns = (
  serviceType?: ServiceType
): MRT_ColumnDef<QueryData>[] =>
  serviceType === ServiceType.mysql
    ? OVERVIEW_TABLE_COLUMNS_MYSQL
    : OVERVIEW_TABLE_COLUMNS;
