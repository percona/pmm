import {
  type MRT_ColumnDef,
  type MRT_TableInstance,
} from 'material-react-table';

import { QueryData } from 'types/rta.types';
import { ServiceType } from 'types/services.types';
import { Messages } from './OverviewTable.messages';
import { QueryCell } from './query-cell';
import UnavailableText from 'components/unavailable-text';
import Stack from '@mui/material/Stack';
import { Chip } from '@percona/peak-ui';
import { darken, type Theme } from '@mui/material/styles';
import {
  BlockedChip,
  BlockedUnknownChip,
} from 'pages/rta/components/blocked-chip';
import { TruncatedChip } from 'pages/rta/components/truncated-chip';
import {
  formatElapsedTime,
  isBlocked,
  sqlPayload,
  isBlockingUnattributed,
  isLockWaitUnreadable,
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
    // The chip sits with the statement rather than in a column of its own: most rows are
    // never blocked, so a dedicated column would be mostly empty and cost horizontal space
    // that the query text uses better.
    <Stack direction="row" alignItems="center" gap={1} sx={{ minWidth: 0 }}>
      {isBlocked(row.original) && (
        <BlockedChip blockers={sqlPayload(row.original)?.blockedBy ?? []} />
      )}
      {isBlockingUnattributed(row.original) && (
        <BlockedUnknownChip reason="unattributed" />
      )}
      {isLockWaitUnreadable(row.original) && (
        <BlockedUnknownChip reason="unreadable" />
      )}
      <QueryCell
        query={row.original.queryText}
        language={queryLanguage(row.original)}
      />
      {row.original.postgresqlPayload?.state.startsWith(
        'idle in transaction'
      ) && (
        <Chip
          size="small"
          color="warning"
          label={row.original.postgresqlPayload.state}
          data-testid={`query-${row.original.queryId}-idle-in-transaction-chip`}
          sx={{ flexShrink: 0 }}
        />
      )}
      {sqlPayload(row.original)?.queryTextTruncated && (
        <TruncatedChip
          dataTestId={`query-${row.original.queryId}-truncated-chip`}
          postgresql={!!row.original.postgresqlPayload}
        />
      )}
    </Stack>
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

// MRT draws a pinned cell at 0.97 opacity over a 0.97-alpha background, so the
// columns scrolling underneath it -- User, Database -- show through faintly.
// This is MRT's own shade without the alpha. "&&" outranks MRT's row-level
// "tr td[data-pinned]::before" rule, which sets the same background. The
// column's own props replace the table-wide cell sx, hence the padding is
// repeated where this is used.
const opaquePinnedCellSx =
  (table: MRT_TableInstance<QueryData>) => (theme: Theme) => ({
    opacity: 1,
    '&&[data-pinned="true"]:before': {
      backgroundColor: darken(
        table.options.mrtTheme.baseBackgroundColor,
        theme.palette.mode === 'dark' ? 0.05 : 0.01
      ),
    },
  });

const ELAPSED_TIME_COLUMN: MRT_ColumnDef<QueryData> = {
  header: Messages.columns.elapsedTime,
  accessorKey: 'queryExecutionDurationMs',
  // Pinned to the right edge, so every pixel here is taken from the query text.
  // The width is set by the header, not the compact value: the cell padding and
  // the sort and column-menu icons take about 80px, leaving the ~95px "Elapsed
  // time" needs only from 180 up (170 cut it to "Elapsed ti…"). minSize is what
  // MRT holds as the floor (min-width is max(size, minSize)), so it stays
  // readable when the Database and User columns compete for the row; grow is
  // off so it takes no more.
  size: 190,
  minSize: 190,
  grow: false,
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
  muiTableHeadCellProps: ({ table }) => ({
    sx: (theme: Theme) => ({ px: 1, ...opaquePinnedCellSx(table)(theme) }),
  }),
  muiTableBodyCellProps: ({ row, table }) => ({
    'data-testid': `query-${row.original.queryId}-elapsed-time-cell`,
    sx: (theme: Theme) => ({
      py: 1,
      px: 1,
      ...opaquePinnedCellSx(table)(theme),
    }),
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

// Database and User are for MySQL and PostgreSQL. MongoDB reports values for both, but they
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
  serviceType === ServiceType.mysql || serviceType === ServiceType.posgresql
    ? OVERVIEW_TABLE_COLUMNS_MYSQL
    : OVERVIEW_TABLE_COLUMNS;
