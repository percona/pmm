import { FC, useMemo, useState } from 'react';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/percona-ui';
import {
  type MRT_ColumnFiltersState,
  type MRT_PaginationState,
  type MRT_Updater,
} from 'material-react-table';
import { enqueueSnackbar } from 'notistack';
import { useSearchParams } from 'react-router-dom';
import { Page } from 'components/page';
import {
  useAdvisors,
  useCheckResultsHistory,
  useMarkCheckResultsRead,
} from 'hooks/api/useAdvisors';
import {
  AdvisorCheckResultStatus,
  CheckResultHistoryItem,
  ListCheckResultsHistoryParams,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { OrgRole } from 'types/user.types';
import { Messages } from './AdvisorInsights.messages';
import { getInsightsColumns } from './AdvisorInsights.constants';

const AdvisorInsights: FC = () => {
  const [searchParams] = useSearchParams();
  // deep link to the results of a concrete run, e.g. from the "View results" snackbar
  const runId = searchParams.get('runId') || undefined;
  const [pagination, setPagination] = useState<MRT_PaginationState>({
    pageIndex: 0,
    pageSize: 100,
  });
  const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>(
    []
  );

  const params = useMemo<ListCheckResultsHistoryParams>(() => {
    const filterValue = (id: string) =>
      (columnFilters.find((filter) => filter.id === id)?.value as string) ||
      undefined;
    const isRead = filterValue('isRead');

    return {
      pageIndex: pagination.pageIndex,
      pageSize: pagination.pageSize,
      runId,
      serviceName: filterValue('serviceName'),
      category: filterValue('category'),
      severity: filterValue('severity') as Severity | undefined,
      status: filterValue('status') as AdvisorCheckResultStatus | undefined,
      isRead: isRead === undefined ? undefined : isRead === 'true',
    };
  }, [pagination, columnFilters, runId]);

  const { data, isLoading, isFetching } = useCheckResultsHistory(params);
  const { data: advisors = [] } = useAdvisors();
  const { mutate: markRead, isPending: isMarking } = useMarkCheckResultsRead();

  const categories = useMemo(
    () => [...new Set(advisors.map((advisor) => advisor.category))].sort(),
    [advisors]
  );

  const columns = useMemo(() => getInsightsColumns(categories), [categories]);

  const handleColumnFiltersChange = (
    updater: MRT_Updater<MRT_ColumnFiltersState>
  ) => {
    setColumnFilters(updater);
    // filters change the result set, so start from the first page
    setPagination((current) => ({ ...current, pageIndex: 0 }));
  };

  const handleToggleRead = (item: CheckResultHistoryItem) =>
    markRead(
      { ids: [item.id], isRead: !item.isRead },
      {
        onSuccess: () =>
          enqueueSnackbar(
            item.isRead
              ? Messages.success.markedUnread
              : Messages.success.markedRead,
            { variant: 'success' }
          ),
      }
    );

  return (
    <Page
      title={Messages.title}
      fullWidth
      wide
      roles={[OrgRole.Editor, OrgRole.Admin]}
    >
      <Stack gap={2} sx={{ flex: 1 }}>
        <Typography variant="body2">{Messages.description}</Typography>
        <Table
          tableName="advisor-insights-table"
          columns={columns}
          data={data?.results ?? []}
          rowCount={data?.totalItems ?? 0}
          noDataMessage={Messages.noData}
          manualPagination
          manualFiltering
          enablePagination
          enableSorting={false}
          enableGlobalFilter={false}
          enableHiding={false}
          enableStickyHeader
          state={{
            pagination,
            columnFilters,
            isLoading,
            showProgressBars: isFetching && !isLoading,
          }}
          initialState={{
            showColumnFilters: true,
          }}
          onPaginationChange={setPagination}
          onColumnFiltersChange={handleColumnFiltersChange}
          enableRowActions
          positionActionsColumn="last"
          renderRowActions={({ row }) => (
            <Button
              color="inherit"
              size="small"
              disabled={isMarking}
              onClick={() => handleToggleRead(row.original)}
              data-testid={`insight-${row.original.id}-toggle-read`}
            >
              {row.original.isRead ? Messages.markUnread : Messages.markRead}
            </Button>
          )}
          muiTableContainerProps={{
            sx: {
              flex: 1,
              borderRadius: 2,
              border: '1px solid',
              borderColor: 'divider',
            },
          }}
        />
      </Stack>
    </Page>
  );
};

export default AdvisorInsights;
