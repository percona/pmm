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
  queryTechnology,
  queryUsername,
  UNAVAILABLE_VALUE,
} from './OverviewTable.utils';
import { ElapsedTimeFilter } from './elapsed-time-filter';

const QUERY_TEXT_COLUMN: MRT_ColumnDef<QueryData> = {
  size: 500,
  header: Messages.columns.queryText,
  accessorKey: 'queryText',
  filterFn: 'contains',
  Cell: ({ row }) => (
    // Inline rather than a column of its own: most rows are never blocked.
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
          technology={queryTechnology(row.original)}
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

// MRT's pinned shade without its alpha, so scrolled columns do not show
// through; "&&" outranks MRT's "tr td[data-pinned]::before" rule.
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
  // Fits the "Elapsed time" header beside the sort and menu icons (170 cuts it);
  // minSize holds that when the Database and User columns compete for the row.
  size: 190,
  minSize: 190,
  grow: false,
  filterVariant: 'range',
  filterFn: 'timeRangeFilterFn',
  Filter: ({ column, rangeFilterIndex }) => (
    <ElapsedTimeFilter column={column} rangeFilterIndex={rangeFilterIndex} />
  ),
  // A just-started statement reports 0; only a missing value is unavailable.
  Cell: ({ cell }) =>
    cell.getValue<number | null>() == null ? (
      <UnavailableText />
    ) : (
      formatElapsedTime(cell.getValue<number>())
    ),
  // Column sx replaces the table-wide cell sx, hence the repeated padding.
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

// Module constants keep the reference stable across polling re-renders.
const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  QUERY_TEXT_COLUMN,
  HOST_COLUMN,
  OPERATION_ID_COLUMN,
  ELAPSED_TIME_COLUMN,
];

// MySQL and PostgreSQL only: MongoDB's database and user values (admin, __system) mean little.
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
