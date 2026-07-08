import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import BlockOutlinedIcon from '@mui/icons-material/BlockOutlined';
import CheckCircleOutlinedIcon from '@mui/icons-material/CheckCircleOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import FilterAltOffOutlinedIcon from '@mui/icons-material/FilterAltOffOutlined';
import FilterAltOutlinedIcon from '@mui/icons-material/FilterAltOutlined';
import MarkEmailReadOutlinedIcon from '@mui/icons-material/MarkEmailReadOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import RefreshOutlinedIcon from '@mui/icons-material/RefreshOutlined';
import ReplayOutlinedIcon from '@mui/icons-material/ReplayOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import IconButton from '@mui/material/IconButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/percona-ui';
import { Page } from 'components/page';
import {
  useAdvisors,
  useChangeAdvisorChecks,
  useCheckResultsFilterValues,
  useCheckResultsHistory,
  useMarkCheckResultsRead,
  useStartAdvisorChecks,
} from 'hooks/api/useAdvisors';
import { type MRT_PaginationState } from 'material-react-table';
import { closeSnackbar, enqueueSnackbar } from 'notistack';
import { FC, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  AdvisorCheckResultStatus,
  CheckResultHistoryItem,
  ListCheckResultsHistoryParams,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { OrgRole } from 'types/user.types';
import { capitalize } from 'utils/text.utils';
import { getInsightsColumns } from './AdvisorInsights.constants';
import { insightToText } from './AdvisorInsights.utils';
import { InsightDetailsPane } from './details-pane';
import {
  READ_FILTER_OPTIONS,
  SEVERITY_FILTER_OPTIONS,
  STATUS_FILTER_OPTIONS,
} from './AdvisorInsights.filters';
import { Messages } from './AdvisorInsights.messages';

interface InsightFilters {
  serviceName: string;
  nodeName: string;
  category: string;
  severity: string;
  status: string;
  isRead: string;
}

const NO_FILTERS: InsightFilters = {
  serviceName: '',
  nodeName: '',
  category: '',
  severity: '',
  status: '',
  isRead: '',
};

interface FilterOption {
  label: string;
  value: string;
}

interface RowActionMenuState {
  anchor: HTMLElement;
  insight: CheckResultHistoryItem;
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
  const [searchParams, setSearchParams] = useSearchParams();
  // deep link to the results of a concrete run, e.g. from the "View results" snackbar
  const runId = searchParams.get('runId') || undefined;
  const [pagination, setPagination] = useState<MRT_PaginationState>({
    pageIndex: 0,
    pageSize: 100,
  });
  const [filters, setFilters] = useState<InsightFilters>(NO_FILTERS);

  const updateFilter = (name: keyof InsightFilters, value: string) => {
    setFilters((current) => ({ ...current, [name]: value }));
    // filters change the result set, so start from the first page
    setPagination((current) => ({ ...current, pageIndex: 0 }));
  };

  const hasActiveFilters = Object.values(filters).some(Boolean) || !!runId;

  const handleClearFilters = () => {
    setFilters(NO_FILTERS);
    if (runId) {
      setSearchParams({});
    }
    setPagination((current) => ({ ...current, pageIndex: 0 }));
  };

  const params = useMemo<ListCheckResultsHistoryParams>(
    () => ({
      pageIndex: pagination.pageIndex,
      pageSize: pagination.pageSize,
      runId,
      serviceName: filters.serviceName || undefined,
      nodeName: filters.nodeName || undefined,
      category: filters.category || undefined,
      severity: (filters.severity as Severity) || undefined,
      status: (filters.status as AdvisorCheckResultStatus) || undefined,
      isRead: filters.isRead === '' ? undefined : filters.isRead === 'true',
    }),
    [pagination, filters, runId]
  );

  const [actionMenu, setActionMenu] = useState<RowActionMenuState | null>(null);
  const [detailsInsight, setDetailsInsight] =
    useState<CheckResultHistoryItem | null>(null);

  const { data, isLoading, isFetching, refetch } =
    useCheckResultsHistory(params);
  const { data: advisors = [], refetch: refetchAdvisors } = useAdvisors();
  const { data: filterValues, refetch: refetchFilterValues } =
    useCheckResultsFilterValues();
  const { mutate: markRead, isPending: isMarking } = useMarkCheckResultsRead();
  const { mutate: startChecks, isPending: isStarting } =
    useStartAdvisorChecks();
  const { mutate: changeChecks, isPending: isChangingCheck } =
    useChangeAdvisorChecks();

  const checksByName = useMemo(
    () =>
      new Map(
        advisors.flatMap((advisor) => advisor.checks).map((c) => [c.name, c])
      ),
    [advisors]
  );

  const applyRunIdFilter = (newRunId: string) => {
    setSearchParams(newRunId ? { runId: newRunId } : {});
    setPagination((current) => ({ ...current, pageIndex: 0 }));
  };

  const handleRerun = (insight: CheckResultHistoryItem) => {
    const checkSummary =
      checksByName.get(insight.checkName)?.summary ?? insight.checkName;
    startChecks([insight.checkName], {
      onSuccess: (newRunId) =>
        enqueueSnackbar(Messages.success.rerunStarted(checkSummary), {
          variant: 'success',
          action: (key) => (
            <Button
              color="inherit"
              size="small"
              onClick={() => {
                closeSnackbar(key);
                applyRunIdFilter(newRunId);
              }}
              data-testid="view-run-results"
            >
              {Messages.viewResults}
            </Button>
          ),
        }),
    });
  };

  const handleToggleCheckEnabled = (insight: CheckResultHistoryItem) => {
    const check = checksByName.get(insight.checkName);
    if (!check) {
      return;
    }
    changeChecks([{ name: check.name, enable: !check.enabled }], {
      onSuccess: () =>
        enqueueSnackbar(
          check.enabled
            ? Messages.success.checkDisabled(check.summary)
            : Messages.success.checkEnabled(check.summary),
          { variant: 'success' }
        ),
    });
  };

  const handleCopyAsText = async (insight: CheckResultHistoryItem) => {
    await navigator.clipboard.writeText(insightToText(insight));
    enqueueSnackbar(Messages.success.copied, { variant: 'success' });
  };

  // reloads the results and the data behind the filter dropdowns
  const handleRefresh = () => {
    refetch();
    refetchAdvisors();
    refetchFilterValues();
  };

  const categoryOptions = useMemo<FilterOption[]>(
    () =>
      [...new Set(advisors.map((advisor) => advisor.category))]
        .sort()
        .map((category) => ({ label: capitalize(category), value: category })),
    [advisors]
  );

  const serviceOptions = useMemo<FilterOption[]>(
    () =>
      (filterValues?.serviceNames ?? []).map((name) => ({
        label: name,
        value: name,
      })),
    [filterValues]
  );

  const nodeOptions = useMemo<FilterOption[]>(
    () =>
      (filterValues?.nodeNames ?? []).map((name) => ({
        label: name,
        value: name,
      })),
    [filterValues]
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
          <FilterSelect
            id="serviceName"
            label={Messages.filters.service}
            options={serviceOptions}
            value={filters.serviceName}
            onChange={(value) => updateFilter('serviceName', value)}
          />
          <FilterSelect
            id="nodeName"
            label={Messages.filters.node}
            options={nodeOptions}
            value={filters.nodeName}
            onChange={(value) => updateFilter('nodeName', value)}
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
          {runId && (
            <Chip
              label={Messages.filters.run(runId.slice(0, 8))}
              onDelete={() => applyRunIdFilter('')}
              sx={{ alignSelf: 'center' }}
              data-testid="run-id-filter-chip"
            />
          )}
          <Button
            startIcon={<FilterAltOffOutlinedIcon />}
            disabled={!hasActiveFilters}
            onClick={handleClearFilters}
            data-testid="clear-filters"
          >
            {Messages.filters.clear}
          </Button>
          <Tooltip title={Messages.filters.refreshTooltip} arrow>
            <Button
              startIcon={<RefreshOutlinedIcon />}
              onClick={handleRefresh}
              data-testid="refresh-insights"
            >
              {Messages.filters.refresh}
            </Button>
          </Tooltip>
        </Stack>
        <Table
          tableName="advisor-insights-table"
          columns={columns}
          data={data?.results ?? []}
          rowCount={data?.totalItems ?? 0}
          noDataMessage={Messages.noData}
          manualPagination
          enablePagination
          enableTopToolbar={false}
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
            <IconButton
              size="small"
              onClick={(e) =>
                setActionMenu({
                  anchor: e.currentTarget,
                  insight: row.original,
                })
              }
              data-testid={`insight-${row.original.id}-actions`}
            >
              <MoreVertIcon fontSize="small" />
            </IconButton>
          )}
          muiTableBodyRowProps={({ row }) => {
            const check = checksByName.get(row.original.checkName);
            // dim results of disabled checks; unknown checks stay normal
            // to avoid flicker while the advisors list loads
            const checkDisabled = check ? !check.enabled : false;

            return {
              'data-testid': `insight-row-${row.original.id}`,
              'data-check-disabled': checkDisabled ? 'true' : undefined,
              sx: checkDisabled ? { opacity: 0.5 } : undefined,
            };
          }}
          muiTableContainerProps={{
            sx: {
              flex: 1,
              borderRadius: 2,
              border: '1px solid',
              borderColor: 'divider',
            },
          }}
        />
        <Menu
          anchorEl={actionMenu?.anchor}
          open={!!actionMenu}
          onClose={() => setActionMenu(null)}
        >
          <MenuItem
            onClick={() => {
              if (actionMenu) {
                setDetailsInsight(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-view-details"
          >
            <ListItemIcon>
              <VisibilityOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.viewDetails}</ListItemText>
          </MenuItem>
          <MenuItem
            disabled={isMarking}
            onClick={() => {
              if (actionMenu) {
                handleToggleRead(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-toggle-read"
          >
            <ListItemIcon>
              <MarkEmailReadOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.toggleRead}</ListItemText>
          </MenuItem>
          <MenuItem
            // the backend silently skips disabled checks, so don't offer the action
            disabled={
              isStarting ||
              !actionMenu ||
              !checksByName.get(actionMenu.insight.checkName)?.enabled
            }
            onClick={() => {
              if (actionMenu) {
                handleRerun(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-rerun-now"
          >
            <ListItemIcon>
              <ReplayOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.rerunNow}</ListItemText>
          </MenuItem>
          <MenuItem
            // rows recorded before run grouping have no run ID
            disabled={!actionMenu?.insight.runId}
            onClick={() => {
              if (actionMenu) {
                applyRunIdFilter(actionMenu.insight.runId);
              }
              setActionMenu(null);
            }}
            data-testid="action-filter-by-run-id"
          >
            <ListItemIcon>
              <FilterAltOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.filterByRunId}</ListItemText>
          </MenuItem>
          <MenuItem
            onClick={() => {
              if (actionMenu) {
                handleCopyAsText(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-copy-as-text"
          >
            <ListItemIcon>
              <ContentCopyOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.copyAsText}</ListItemText>
          </MenuItem>
          <MenuItem
            // unknown checks (e.g. removed advisors) cannot be toggled
            disabled={
              isChangingCheck ||
              !actionMenu ||
              !checksByName.has(actionMenu.insight.checkName)
            }
            onClick={() => {
              if (actionMenu) {
                handleToggleCheckEnabled(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-disable-check"
          >
            <ListItemIcon>
              {actionMenu &&
              checksByName.get(actionMenu.insight.checkName)?.enabled ===
                false ? (
                <CheckCircleOutlinedIcon fontSize="small" />
              ) : (
                <BlockOutlinedIcon fontSize="small" />
              )}
            </ListItemIcon>
            <ListItemText>
              {actionMenu &&
              checksByName.get(actionMenu.insight.checkName)?.enabled === false
                ? Messages.actions.enableCheck
                : Messages.actions.disableCheck}
            </ListItemText>
          </MenuItem>
        </Menu>
        <InsightDetailsPane
          insight={detailsInsight}
          checkEnabled={
            detailsInsight
              ? checksByName.get(detailsInsight.checkName)?.enabled
              : undefined
          }
          onClose={() => setDetailsInsight(null)}
        />
      </Stack>
    </Page>
  );
};

export default AdvisorInsights;
