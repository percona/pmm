import { type MRT_ColumnDef } from 'material-react-table';
import Tooltip from '@mui/material/Tooltip';
import Chip from '@mui/material/Chip';

import { QueryData, hasLockChain, isIdleInTransaction } from 'types/rta.types';
import { Messages } from './OverviewTable.messages';
import { QueryCell } from './query-cell';
import { formatDuration } from 'date-fns';
import UnavailableText from 'components/unavailable-text';

export const OVERVIEW_TABLE_COLUMNS: MRT_ColumnDef<QueryData>[] = [
  {
    size: 500,
    header: Messages.columns.queryText,
    accessorKey: 'queryText',
    filterFn: 'contains',
    Cell: ({ row }) => (
      <QueryCell
        query={row.original.queryText}
        truncated={row.original.postgresqlPayload?.queryTruncated}
        language={row.original.postgresqlPayload ? 'text' : 'mongodb'}
      />
    ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-query-text-cell`,
    }),
  },
  {
    header: Messages.columns.state,
    accessorKey: 'postgresqlPayload.state',
    enableSorting: false,
    Cell: ({ row }) => {
      const state = row.original.postgresqlPayload?.state;
      if (!state) {
        return <UnavailableText />;
      }

      if (isIdleInTransaction(row.original)) {
        return (
          <Chip
            size="small"
            color="warning"
            label={Messages.idleInTransaction}
            data-testid={`query-${row.original.queryId}-idle-in-transaction`}
          />
        );
      }

      return state;
    },
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-state-cell`,
    }),
  },
  {
    header: Messages.columns.waitEvent,
    accessorFn: (row) =>
      row.postgresqlPayload
        ? [row.postgresqlPayload.waitEventType, row.postgresqlPayload.waitEvent]
            .filter(Boolean)
            .join(' / ')
        : '',
    enableSorting: false,
    Cell: ({ cell }) =>
      cell.getValue<string>() ? (
        cell.getValue<string>()
      ) : (
        <UnavailableText />
      ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-wait-event-cell`,
    }),
  },
  {
    header: Messages.columns.locks,
    accessorFn: (row) => row.postgresqlPayload?.lockChain?.length ?? 0,
    enableSorting: false,
    Cell: ({ row }) =>
      hasLockChain(row.original) ? (
        <Tooltip title={Messages.lockChainHint}>
          <Chip
            size="small"
            color="error"
            label={Messages.blocked}
            data-testid={`query-${row.original.queryId}-lock-indicator`}
          />
        </Tooltip>
      ) : (
        <UnavailableText />
      ),
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-locks-cell`,
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
    Cell: ({ row, cell }) => {
      const durationMs = isIdleInTransaction(row.original)
        ? row.original.transactionDurationMs
        : cell.getValue<number>();

      return durationMs ? (
        `${formatDuration({ seconds: durationMs }, { format: ['seconds'] })}`
      ) : (
        <UnavailableText />
      );
    },
    // @ts-expect-error - muiTableBodyCellProps is not typed correctly
    muiTableBodyCellProps: ({ row }) => ({
      'data-testid': `query-${row.original.queryId}-elapsed-time-cell`,
    }),
  },
];
