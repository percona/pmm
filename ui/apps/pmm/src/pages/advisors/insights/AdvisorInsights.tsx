import { FC, useMemo, useRef, useState } from 'react';
import Button from '@mui/material/Button';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/percona-ui';
import { type MRT_PaginationState } from 'material-react-table';
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
import { capitalize } from 'utils/text.utils';
import { Messages } from './AdvisorInsights.messages';
import { getInsightsColumns } from './AdvisorInsights.constants';
import {
  READ_FILTER_OPTIONS,
  SEVERITY_FILTER_OPTIONS,
  STATUS_FILTER_OPTIONS,
} from './AdvisorInsights.filters';

const SERVICE_FILTER_DEBOUNCE_MS = 400;

interface InsightFilters {
  serviceName: string;
  category: string;
  severity: string;
  status: string;
  isRead: string;
}

const NO_FILTERS: InsightFilters = {
  serviceName: '',
  category: '',
  severity: '',
  status: '',
  isRead: '',
};

interface FilterOption {
  label: string;
  value: string;
}

interface FilterSelectProps {
  id: string;
  label: string;
  options: FilterOption[];
  value: string;
  onChange: (value: string) => void;
}

const FilterSelect: FC<FilterSelectProps> = ({
  id,
  label,
  options,
  value,
  onChange,
}) => (
  <TextField
    select
    size="small"
    // deterministic id: React's useId default breaks jsdom selector matching
    id={`${id}-filter-select`}
    label={label}
    value={value || Messages.filters.all}
    onChange={(e) =>
      onChange(e.target.value === Messages.filters.all ? '' : e.target.value)
    }
    sx={{ minWidth: 140 }}
    data-testid={`${id}-filter`}
  >
    <MenuItem value={Messages.filters.all}>{Messages.filters.all}</MenuItem>
    {options.map((option) => (
      <MenuItem key={option.value} value={option.value}>
        {option.label}
      </MenuItem>
    ))}
  </TextField>
);

const AdvisorInsights: FC = () => {
  const [searchParams] = useSearchParams();
  // deep link to the results of a concrete run, e.g. from the "View results" snackbar
  const runId = searchParams.get('runId') || undefined;
  const [pagination, setPagination] = useState<MRT_PaginationState>({
    pageIndex: 0,
    pageSize: 100,
  });
  const [filters, setFilters] = useState<InsightFilters>(NO_FILTERS);
  const [serviceInput, setServiceInput] = useState('');
  const serviceTimer = useRef<ReturnType<typeof setTimeout>>();

  const updateFilter = (name: keyof InsightFilters, value: string) => {
    setFilters((current) => ({ ...current, [name]: value }));
    // filters change the result set, so start from the first page
    setPagination((current) => ({ ...current, pageIndex: 0 }));
  };

  const handleServiceInputChange = (value: string) => {
    setServiceInput(value);
    clearTimeout(serviceTimer.current);
    serviceTimer.current = setTimeout(
      () => updateFilter('serviceName', value),
      SERVICE_FILTER_DEBOUNCE_MS
    );
  };

  const params = useMemo<ListCheckResultsHistoryParams>(
    () => ({
      pageIndex: pagination.pageIndex,
      pageSize: pagination.pageSize,
      runId,
      serviceName: filters.serviceName || undefined,
      category: filters.category || undefined,
      severity: (filters.severity as Severity) || undefined,
      status: (filters.status as AdvisorCheckResultStatus) || undefined,
      isRead: filters.isRead === '' ? undefined : filters.isRead === 'true',
    }),
    [pagination, filters, runId]
  );

  const { data, isLoading, isFetching } = useCheckResultsHistory(params);
  const { data: advisors = [] } = useAdvisors();
  const { mutate: markRead, isPending: isMarking } = useMarkCheckResultsRead();

  const categoryOptions = useMemo<FilterOption[]>(
    () =>
      [...new Set(advisors.map((advisor) => advisor.category))]
        .sort()
        .map((category) => ({ label: capitalize(category), value: category })),
    [advisors]
  );

  const columns = useMemo(() => getInsightsColumns(), []);

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
        <Stack direction="row" flexWrap="wrap" gap={2}>
          <TextField
            size="small"
            id="service-filter-input"
            label={Messages.filters.service}
            placeholder={Messages.filters.servicePlaceholder}
            value={serviceInput}
            onChange={(e) => handleServiceInputChange(e.target.value)}
            sx={{ minWidth: 220 }}
            data-testid="service-filter"
          />
          <FilterSelect
            id="category"
            label={Messages.filters.category}
            options={categoryOptions}
            value={filters.category}
            onChange={(value) => updateFilter('category', value)}
          />
          <FilterSelect
            id="severity"
            label={Messages.filters.severity}
            options={SEVERITY_FILTER_OPTIONS}
            value={filters.severity}
            onChange={(value) => updateFilter('severity', value)}
          />
          <FilterSelect
            id="status"
            label={Messages.filters.status}
            options={STATUS_FILTER_OPTIONS}
            value={filters.status}
            onChange={(value) => updateFilter('status', value)}
          />
          <FilterSelect
            id="isRead"
            label={Messages.filters.read}
            options={READ_FILTER_OPTIONS}
            value={filters.isRead}
            onChange={(value) => updateFilter('isRead', value)}
          />
        </Stack>
        <Table
          tableName="advisor-insights-table"
          columns={columns}
          data={data?.results ?? []}
          rowCount={data?.totalItems ?? 0}
          noDataMessage={Messages.noData}
          manualPagination
          enablePagination
          enableGlobalFilter={false}
          enableColumnFilters={false}
          enableColumnActions={false}
          enableHiding={false}
          enableStickyHeader
          state={{
            pagination,
            isLoading,
            showProgressBars: isFetching && !isLoading,
          }}
          onPaginationChange={setPagination}
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
