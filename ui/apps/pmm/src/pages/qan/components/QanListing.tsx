import { Table } from '@percona/percona-ui';
import { Box, CircularProgress, Stack, Typography } from '@mui/material';
import { FC, useMemo } from 'react';
import type { MRT_ColumnDef } from 'material-react-table';
import { useQanReport } from 'hooks/api/useQan';
import { useQanPanelActions, useQanPanelState } from '../hooks/useQanPanelState';
import { getLabelQueryParams, DEFAULT_QAN_COLUMNS } from '../utils/qanTools';
import { qanMetricDisplayValue, qanMetricSparkline } from '../utils/qanMetrics';
import { formatQanMetricFigure, qanColumnLabel } from '../utils/qanDisplay';
import { orderByFromSorting, sortingFromOrderBy } from '../utils/qanOrderBy';
import type { QanReportRow } from 'types/qan.types';
import { QanMetricSparkline } from './QanMetricSparkline';
import { QanQueryCell } from './QanQueryCell';

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
        size: 40,
        enableSorting: false,
        Cell: ({ row }) =>
          row.index === 0 && tableRows.length > 1 ? '' : row.index,
      },
      {
        id: 'main',
        header: 'Query Fingerprint',
        accessorFn: (row) => row.fingerprint || row.dimension || '',
        Cell: ({ row }) => {
          const isTotalsRow = row.index === 0 && tableRows.length > 1;
          return (
            <QanQueryCell
              fingerprint={row.original.fingerprint}
              dimension={row.original.dimension}
              isTotals={isTotalsRow}
            />
          );
        },
        size: 360,
        enableSorting: false,
      },
    ];
    metricColumns.forEach((col) => {
      base.push({
        id: col,
        header: qanColumnLabel(col),
        accessorFn: (row) => qanMetricDisplayValue(col, row),
        Cell: ({ row }) => {
          const isTotalsRow = row.index === 0 && tableRows.length > 1;
          const value = qanMetricDisplayValue(col, row.original);
          const sparkline = qanMetricSparkline(row.original.metrics, col);
          return (
            <Stack
              spacing={1}
              alignItems="flex-end"
              sx={{ py: 0.5, minWidth: 120 }}
            >
              <Typography
                sx={{
                  fontFamily: '"Roboto Mono", monospace',
                  fontSize: 16,
                  fontWeight: 500,
                  lineHeight: 1.5,
                  textAlign: 'right',
                }}
              >
                {formatQanMetricFigure(col, value)}
              </Typography>
              {!isTotalsRow ? <QanMetricSparkline points={sparkline} /> : null}
            </Stack>
          );
        },
        size: 145,
        enableSorting: true,
      });
    });
    return base;
  }, [metricColumns, tableRows.length]);

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
                ...(isTotalsRow
                  ? {
                      bgcolor: 'background.paper',
                      '& td': { borderTop: 1, borderColor: 'divider' },
                    }
                  : {}),
                ...(selected
                  ? {
                      bgcolor: 'rgba(32, 68, 147, 0.12)',
                      outline: 2,
                      outlineStyle: 'dashed',
                      outlineColor: 'primary.light',
                      outlineOffset: -2,
                    }
                  : {}),
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
          muiTableHeadCellProps={{
            sx: {
              fontWeight: 500,
              fontSize: 16,
              '&:not(:first-of-type):not(:nth-of-type(2))': { textAlign: 'right' },
            },
          }}
          muiBottomToolbarProps={{
            sx: { borderTop: 1, borderColor: 'divider', minHeight: 48 },
          }}
          noDataMessage="No queries found for the selected filters and time range."
        />
      )}
    </Box>
  );
};
