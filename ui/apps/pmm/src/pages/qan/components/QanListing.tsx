import { Table } from '@percona/percona-ui';
import { Box, CircularProgress, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import type { MRT_ColumnDef, MRT_SortingState } from 'material-react-table';
import { useQanReport } from 'hooks/api/useQan';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { getLabelQueryParams, DEFAULT_QAN_COLUMNS } from '../utils/qanTools';
import { qanMetricDisplayValue } from '../utils/qanMetrics';
import type { QanReportRow } from 'types/qan.types';

function formatMetric(value: unknown): string {
  if (value == null || Number.isNaN(value)) return '—';
  if (typeof value === 'number') {
    if (Math.abs(value) >= 1000) return value.toFixed(1);
    if (Math.abs(value) >= 1) return value.toFixed(3);
    return value.toFixed(4);
  }
  return String(value);
}

function sortingFromOrderBy(orderBy: string): MRT_SortingState {
  const id = orderBy.replace(/^-/, '');
  if (!id) return [];
  return [{ id, desc: !orderBy.startsWith('-') }];
}

function orderByFromSorting(sorting: MRT_SortingState): string {
  const first = sorting[0];
  if (!first) return '';
  return first.desc ? first.id : `-${first.id}`;
}

export const QanListing: FC = () => {
  const state = useQanPanelState();
  const actions = useQanPanelActions();
  const metricColumns =
    Array.isArray(state.columns) && state.columns.length
      ? state.columns
      : DEFAULT_QAN_COLUMNS;

  const reportParams = useMemo(
    () => ({
      columns: metricColumns,
      groupBy: state.groupBy,
      labels: getLabelQueryParams(state.labels),
      limit: state.pageSize,
      offset: (state.pageNumber - 1) * state.pageSize,
      orderBy: state.orderBy,
      mainMetric: metricColumns[0] ?? 'load',
      periodStartFrom: state.from,
      periodStartTo: state.to,
      search: state.dimensionSearchText,
    }),
    [state, metricColumns]
  );

  const { data, isLoading, isError } = useQanReport(reportParams);

  const rows = data?.rows ?? [];
  const tableRows = rows.length > 1 ? rows : rows.length === 1 ? rows : [];

  const sorting = useMemo(() => sortingFromOrderBy(state.orderBy), [state.orderBy]);
  const pagination = useMemo(
    () => ({ pageIndex: state.pageNumber - 1, pageSize: state.pageSize }),
    [state.pageNumber, state.pageSize]
  );

  const tableColumns = useMemo<MRT_ColumnDef<QanReportRow>[]>(() => {
    const base: MRT_ColumnDef<QanReportRow>[] = [
      {
        accessorKey: 'rank',
        header: '#',
        size: 48,
        enableSorting: false,
        Cell: ({ row }) =>
          row.index === 0 && tableRows.length > 1 ? '' : row.index,
      },
      {
        id: 'main',
        header: state.groupBy === 'queryid' ? 'Query' : 'Dimension',
        accessorFn: (row) => row.fingerprint || row.dimension || '',
        Cell: ({ row }) => {
          const isTotalsRow = row.index === 0 && tableRows.length > 1;
          const label = isTotalsRow
            ? 'TOTAL'
            : row.original.fingerprint || row.original.dimension || 'N/A';
          return (
            <Typography variant="body2" noWrap title={label}>
              {label}
            </Typography>
          );
        },
        size: 320,
        enableSorting: false,
      },
    ];
    metricColumns.forEach((col) => {
      base.push({
        id: col,
        header: col.replace(/_/g, ' '),
        accessorFn: (row) => qanMetricDisplayValue(col, row),
        Cell: ({ row }) => formatMetric(qanMetricDisplayValue(col, row.original)),
        size: 120,
        enableSorting: true,
      });
    });
    return base;
  }, [metricColumns, state.groupBy, tableRows.length]);

  if (isError) {
    return (
      <Typography color="error" sx={{ p: 2 }}>
        Failed to load query overview.
      </Typography>
    );
  }

  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        display: 'flex',
        flexDirection: 'column',
        border: 1,
        borderColor: 'divider',
        borderRadius: 1,
        overflow: 'hidden',
        bgcolor: 'background.paper',
      }}
    >
      {isLoading && !data ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
          <CircularProgress size={32} />
        </Box>
      ) : (
        <Table
          tableName="native-qan-overview"
          columns={tableColumns}
          data={tableRows}
          enableGlobalFilter={false}
          enablePagination
          enableBottomToolbar
          enableRowHoverAction
          rowHoverAction={(row) => {
            const isTotalsRow = row.index === 0 && tableRows.length > 1;
            const id = row.original.dimension ?? '';
            actions.selectQuery(
              id,
              row.original.fingerprint,
              row.original.database,
              isTotalsRow
            );
          }}
          muiTableBodyRowProps={({ row }) => {
            const isTotalsRow = row.index === 0 && tableRows.length > 1;
            const selected =
              state.querySelected &&
              (isTotalsRow
                ? state.totals
                : !state.totals && state.queryId === row.original.dimension);
            return {
              'data-testid': `qan-row-${row.original.dimension ?? row.index}`,
              sx: {
                cursor: 'pointer',
                ...(selected ? { backgroundColor: 'action.selected' } : {}),
              },
            };
          }}
          state={{ sorting, pagination }}
          manualSorting
          onSortingChange={(updater) => {
            const current = sortingFromOrderBy(state.orderBy);
            const next = typeof updater === 'function' ? updater(current) : updater;
            const orderBy = orderByFromSorting(next);
            if (orderBy) actions.setOrderBy(orderBy);
          }}
          manualPagination
          rowCount={data?.totalRows ?? 0}
          onPaginationChange={(updater) => {
            const current = pagination;
            const next = typeof updater === 'function' ? updater(current) : updater;
            actions.setPage(next.pageIndex + 1, next.pageSize);
          }}
          muiTableContainerProps={{ sx: { flex: 1 } }}
          muiBottomToolbarProps={{
            sx: { borderTop: 1, borderColor: 'divider' },
          }}
          noDataMessage="No queries found for the selected filters and time range."
        />
      )}
    </Box>
  );
};
