import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { type MRT_ColumnDef } from 'material-react-table';
import { formatDuration } from 'date-fns';

import UnavailableText from 'components/unavailable-text';
import { isPostgresQuery, QueryData } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { QueryCell } from './query-cell';

const elapsedCell = ({ cell }: { cell: { getValue: () => unknown } }) =>
  cell.getValue() ? (
    `${formatDuration(
      {
        seconds: cell.getValue<number>(),
      },
      {
        format: ['seconds'],
      }
    )}`
  ) : (
    <UnavailableText />
  );

const BASE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    size: 500,
    header: Messages.columns.queryText,
    accessorKey: 'queryText',
    filterFn: 'contains',
    Cell: ({ row }) => <QueryCell query={row.original.queryText} />,
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
    filterVariant: 'range',
    filterFn: 'timeRangeFilterFn',
    muiTableHeadCellFilterTextFieldProps: {
      inputProps: { step: 0.25, type: 'number' },
    },
    Cell: elapsedCell,
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-elapsed-time-cell`,
    }),
  },
];

const POSTGRES_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    header: Messages.columns.sessionState,
    id: 'sessionState',
    accessorFn: (row) => row.postgresPayload?.sessionState ?? '',
    Cell: ({ row }) => {
      const payload = row.original.postgresPayload;
      if (!payload) {
        return <UnavailableText />;
      }

      const isIdleInTransaction = payload.sessionState === 'idle in transaction';

      return (
        <Stack direction="row" spacing={0.5} alignItems="center">
          <span>{payload.sessionState}</span>
          {isIdleInTransaction && (
            <Chip label="idle in tx" color="warning" size="small" variant="outlined" />
          )}
          {(payload.parallelWorkerCount ?? 0) > 0 && (
            <Tooltip title={`${payload.parallelWorkerCount} parallel worker(s) hidden`}>
              <Chip
                label={`+${payload.parallelWorkerCount} workers`}
                size="small"
                variant="outlined"
              />
            </Tooltip>
          )}
        </Stack>
      );
    },
  },
  {
    header: Messages.columns.database,
    id: 'databaseName',
    accessorFn: (row) => row.postgresPayload?.databaseName ?? '',
  },
  {
    header: Messages.columns.waitEvent,
    id: 'waitEvent',
    accessorFn: (row) => {
      const payload = row.postgresPayload;
      if (!payload) {
        return '';
      }

      return [payload.waitEventType, payload.waitEvent].filter(Boolean).join(' / ');
    },
  },
];

export const getOverviewTableColumns = (
  queries: QueryData[]
): MRT_ColumnDef<QueryData>[] => {
  if (!queries.some(isPostgresQuery)) {
    return BASE_COLUMNS;
  }

  const [queryText, host, operationId, elapsedTime] = BASE_COLUMNS;

  return [queryText, host, ...POSTGRES_COLUMNS, operationId, elapsedTime];
};
