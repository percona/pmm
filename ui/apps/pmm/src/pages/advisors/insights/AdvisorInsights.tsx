import Button from '@mui/material/Button';
import BlockOutlinedIcon from '@mui/icons-material/BlockOutlined';
import CheckCircleOutlinedIcon from '@mui/icons-material/CheckCircleOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import DraftsOutlinedIcon from '@mui/icons-material/DraftsOutlined';
import FilterAltOffOutlinedIcon from '@mui/icons-material/FilterAltOffOutlined';
import FilterAltOutlinedIcon from '@mui/icons-material/FilterAltOutlined';
import LinkOutlinedIcon from '@mui/icons-material/LinkOutlined';
import MarkEmailReadOutlinedIcon from '@mui/icons-material/MarkEmailReadOutlined';
import MarkunreadOutlinedIcon from '@mui/icons-material/MarkunreadOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import RefreshOutlinedIcon from '@mui/icons-material/RefreshOutlined';
import ReplayOutlinedIcon from '@mui/icons-material/ReplayOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import Box from '@mui/material/Box';
import IconButton from '@mui/material/IconButton';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import { paperClasses } from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/percona-ui';
import { Page } from 'components/page';
import {
  useAdvisors,
  useChangeAdvisorChecks,
  useInsightsFilterValues,
  useInsights,
  useMarkInsightsRead,
  useStartAdvisorChecks,
} from 'hooks/api/useAdvisors';
import {
  type MRT_PaginationState,
  type MRT_Updater,
} from 'material-react-table';
import { closeSnackbar, enqueueSnackbar } from 'notistack';
import { FC, useEffect, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import {
  AdvisorCheckResultStatus,
  Insight,
  InsightsFilters,
  ListInsightsParams,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import { OrgRole } from 'types/user.types';
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

// URL query-string key for each filter field (deep-linkable filters)
const FILTER_PARAM: Record<keyof InsightFilters, string> = {
  serviceName: 'service',
  nodeName: 'node',
  category: 'category',
  severity: 'severity',
  status: 'status',
  isRead: 'read',
};

const DEFAULT_PAGE_SIZE = 100;

interface FilterOption {
  label: string;
  value: string;
}

interface RowActionMenuState {
  anchor: HTMLElement;
  insight: Insight;
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

  // the URL query string is the single source of truth for filters +
  // pagination, so a shared link reproduces the exact same view
  const batchId = searchParams.get('batchId') || undefined;
  const filters: InsightFilters = {
    serviceName: searchParams.get(FILTER_PARAM.serviceName) || '',
    nodeName: searchParams.get(FILTER_PARAM.nodeName) || '',
    category: searchParams.get(FILTER_PARAM.category) || '',
    severity: searchParams.get(FILTER_PARAM.severity) || '',
    status: searchParams.get(FILTER_PARAM.status) || '',
    isRead: searchParams.get(FILTER_PARAM.isRead) || '',
  };
  const pagination: MRT_PaginationState = {
    pageIndex: Math.max(0, (Number(searchParams.get('page')) || 1) - 1),
    pageSize: Number(searchParams.get('pageSize')) || DEFAULT_PAGE_SIZE,
  };

  // local mirror of the batch-id URL param, committed on blur/Enter so typing
  // does not refetch (and spam history) on every keystroke
  const [batchIdInput, setBatchIdInput] = useState(batchId ?? '');
  useEffect(() => {
    setBatchIdInput(batchId ?? '');
  }, [batchId]);

  // apply query-string changes; filter changes reset back to the first page
  const patchParams = (
    mutate: (params: URLSearchParams) => void,
    { resetPage = true }: { resetPage?: boolean } = {}
  ) => {
    const next = new URLSearchParams(searchParams);
    mutate(next);
    if (resetPage) {
      next.delete('page');
    }
    setSearchParams(next, { replace: true });
  };

  const updateFilter = (name: keyof InsightFilters, value: string) =>
    patchParams((p) => {
      if (value) {
        p.set(FILTER_PARAM[name], value);
      } else {
        p.delete(FILTER_PARAM[name]);
      }
    });

  const hasActiveFilters = Object.values(filters).some(Boolean) || !!batchId;

  const handleClearFilters = () => {
    // keep the chosen page size; drop every filter, the batch and the page
    const next = new URLSearchParams();
    const pageSize = searchParams.get('pageSize');
    if (pageSize) {
      next.set('pageSize', pageSize);
    }
    setSearchParams(next, { replace: true });
  };

  // filter portion of the query, shared with the bulk mark-as-read action
  const filterParams: InsightsFilters = {
    batchId,
    serviceName: filters.serviceName || undefined,
    nodeName: filters.nodeName || undefined,
    category: filters.category || undefined,
    severity: (filters.severity as Severity) || undefined,
    status: (filters.status as AdvisorCheckResultStatus) || undefined,
    isRead: filters.isRead === '' ? undefined : filters.isRead === 'true',
  };

  const params: ListInsightsParams = {
    pageIndex: pagination.pageIndex,
    pageSize: pagination.pageSize,
    ...filterParams,
  };

  const [actionMenu, setActionMenu] = useState<RowActionMenuState | null>(null);
  const openDetails = (insight: Insight) =>
    patchParams((p) => p.set('insight', insight.id), { resetPage: false });

  const { data, isLoading, isFetching, refetch } = useInsights(params);

  // the open details overlay is driven by the ?insight=<id> URL param (the
  // result's Check ID), so the overlay is deep-linkable via a shared URL
  const detailsInsight = useMemo(
    () =>
      (data?.results ?? []).find(
        (item) => item.id === (searchParams.get('insight') || '')
      ) ?? null,
    [data, searchParams]
  );
  const { data: advisors = [], refetch: refetchAdvisors } = useAdvisors();
  const { data: filterValues, refetch: refetchFilterValues } =
    useInsightsFilterValues();
  const { mutate: markRead, isPending: isMarking } = useMarkInsightsRead();
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

  // a check can be turned off for individual services while staying enabled
  const isCheckDisabledForService = (insight: Insight) =>
    !!checksByName
      .get(insight.checkName)
      ?.disabledServiceIds?.includes(insight.serviceId);

  const applyBatchIdFilter = (newBatchId: string) =>
    patchParams((p) => {
      if (newBatchId) {
        p.set('batchId', newBatchId);
      } else {
        p.delete('batchId');
      }
    });

  const handlePaginationChange = (
    updater: MRT_Updater<MRT_PaginationState>
  ) => {
    const nextPagination =
      typeof updater === 'function' ? updater(pagination) : updater;
    patchParams(
      (p) => {
        p.set('page', String(nextPagination.pageIndex + 1));
        p.set('pageSize', String(nextPagination.pageSize));
      },
      { resetPage: false }
    );
  };

  const commitBatchIdInput = () => {
    const trimmed = batchIdInput.trim();
    if (trimmed !== (batchId ?? '')) {
      applyBatchIdFilter(trimmed);
    }
  };

  // re-runs the check against this insight's service only; running it against
  // every service is what the Advisor checks page is for
  const handleRerun = (insight: Insight) => {
    const checkSummary =
      checksByName.get(insight.checkName)?.summary ?? insight.checkName;
    startChecks(
      { names: [insight.checkName], serviceIds: [insight.serviceId] },
      {
        onSuccess: (newBatchId) => {
          void navigator.clipboard.writeText(newBatchId);
          enqueueSnackbar(
            Messages.success.rerunStarted(checkSummary, insight.serviceName),
            {
              variant: 'success',
              action: (key) => (
                <Button
                  color="inherit"
                  size="small"
                  onClick={() => {
                    closeSnackbar(key);
                    applyBatchIdFilter(newBatchId);
                  }}
                  data-testid="view-run-results"
                >
                  {Messages.viewResults}
                </Button>
              ),
            }
          );
        },
      }
    );
  };

  // enables/disables the check for this insight's service only, leaving the
  // check running everywhere else
  const handleToggleCheckForService = (insight: Insight) => {
    const check = checksByName.get(insight.checkName);
    if (!check) {
      return;
    }
    const enable = isCheckDisabledForService(insight);
    changeChecks(
      [{ name: check.name, serviceIds: [insight.serviceId], enable }],
      {
        onSuccess: () =>
          enqueueSnackbar(
            enable
              ? Messages.success.checkEnabled(
                  check.summary,
                  insight.serviceName
                )
              : Messages.success.checkDisabled(
                  check.summary,
                  insight.serviceName
                ),
            { variant: 'success' }
          ),
      }
    );
  };

  const handleCopyAsText = async (insight: Insight) => {
    await navigator.clipboard.writeText(insightToText(insight));
    enqueueSnackbar(Messages.success.copied, { variant: 'success' });
  };

  // shareable deep link: keeps the current filters and page so the recipient
  // lands on the same view with this insight's details pane open
  const handleCopyUrl = async (insight: Insight) => {
    const params = new URLSearchParams(searchParams);
    params.set('insight', insight.id);
    const { origin, pathname } = window.location;
    await navigator.clipboard.writeText(`${origin}${pathname}?${params}`);
    enqueueSnackbar(Messages.success.urlCopied, { variant: 'success' });
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
        .map((category) => ({ label: category, value: category })),
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

  const handleToggleRead = (item: Insight) =>
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

  const columns = useMemo(() => getInsightsColumns(), []);

  const handleMarkFilteredRead = () =>
    markRead(
      { filters: filterParams, isRead: true },
      {
        onSuccess: () =>
          enqueueSnackbar(Messages.success.markedFilteredRead, {
            variant: 'success',
          }),
      }
    );

  return (
    <Page
      title={Messages.title}
      fullWidth
      wide
      fillViewport
      footer={null}
      roles={[OrgRole.Editor, OrgRole.Admin]}
    >
      <Stack
        gap={2}
        sx={{
          flex: 1,
          minHeight: 0,
          // let the table fill the remaining height and scroll internally
          // (mirrors RealtimeTableWrapper), so the page itself does not scroll
          [`& > .${paperClasses.root}`]: {
            flex: 1,
            display: 'flex',
            flexDirection: 'column',
            minHeight: 0,
            overflow: 'hidden',
          },
        }}
      >
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
          <TextField
            size="small"
            // deterministic id: React's useId default breaks jsdom selector matching
            id="batchId-filter-input"
            label={Messages.filters.batchId}
            value={batchIdInput}
            onChange={(e) => setBatchIdInput(e.target.value)}
            onBlur={commitBatchIdInput}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                commitBatchIdInput();
              }
            }}
            sx={{ minWidth: 260 }}
            data-testid="batch-id-filter"
          />
          {/* tight icon cluster so the toolbar stays on a single row */}
          <Stack direction="row" gap={0.5} sx={{ alignSelf: 'center' }}>
            <Tooltip title={Messages.filters.clear} arrow>
              <Box component="span">
                <IconButton
                  disabled={!hasActiveFilters}
                  onClick={handleClearFilters}
                  aria-label={Messages.filters.clear}
                  data-testid="clear-filters"
                >
                  <FilterAltOffOutlinedIcon />
                </IconButton>
              </Box>
            </Tooltip>
            <Tooltip title={Messages.filters.refreshTooltip} arrow>
              <IconButton
                onClick={handleRefresh}
                aria-label={Messages.filters.refresh}
                data-testid="refresh-insights"
              >
                <RefreshOutlinedIcon />
              </IconButton>
            </Tooltip>
            <Tooltip
              title={
                hasActiveFilters
                  ? Messages.markFilteredReadTooltip
                  : Messages.markFilteredReadNoFilters
              }
              arrow
            >
              <Box component="span">
                <IconButton
                  // require a filter so a stray click cannot mark everything read
                  disabled={!hasActiveFilters || isMarking || !data?.totalItems}
                  onClick={handleMarkFilteredRead}
                  aria-label={Messages.markFilteredRead}
                  data-testid="mark-filtered-read"
                >
                  <MarkEmailReadOutlinedIcon />
                </IconButton>
              </Box>
            </Tooltip>
          </Stack>
        </Stack>
        <Table
          tableName="advisor-insights-table"
          columns={columns}
          data={data?.results ?? []}
          rowCount={data?.totalItems ?? 0}
          noDataMessage={Messages.noData}
          noDataAlertProps={{ sx: { justifyContent: 'center' } }}
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
          onPaginationChange={handlePaginationChange}
          enableRowActions
          positionActionsColumn="last"
          // mirrors the Advisors list: show the "Actions" header (percona-ui
          // hides it by default) and lock the column width (grow: false)
          displayColumnDefOptions={{
            'mrt-row-actions': {
              header: Messages.columns.actions,
              // undefined lets MRT render the `header` string with its normal head-cell
              // layout (percona-ui defaults it to a visually-hidden span)
              Header: undefined,
              size: 110,
              grow: false,
              muiTableBodyCellProps: { align: 'center', sx: { px: 1 } },
              // MRT zeroes the bottom padding on display-column head cells;
              // match the 0.6rem data headers get so the label aligns with them
              muiTableHeadCellProps: {
                align: 'center',
                sx: {
                  px: 1,
                  pb: '0.6rem',
                  // align='center' only sets text-align. MRT's head-cell
                  // content row pushes its (here empty) actions box right with
                  // margin-left:auto, which eats the free space before
                  // justify-content applies — so stretch the label box to fill
                  // the row and centre the label inside it instead.
                  '& .Mui-TableHeadCell-Content-Labels': {
                    flex: 1,
                    justifyContent: 'center',
                  },
                },
              },
            },
          }}
          renderRowActions={({ row }) => (
            <Stack
              direction="row"
              gap={0.25}
              alignItems="center"
              justifyContent="center"
              // percona-ui's theme pins MuiIconButton to a fixed 40x40, so the
              // icons overflow the column however small their padding gets.
              // Unpin the box to collapse each button onto its icon.
              sx={{
                width: '100%',
                '& .MuiIconButton-root': {
                  width: 'auto',
                  height: 'auto',
                  p: 0.5,
                },
              }}
            >
              <Tooltip
                title={
                  row.original.isRead
                    ? Messages.markAsUnread
                    : Messages.markAsRead
                }
                arrow
              >
                <IconButton
                  // grey once read, blue while unread — same signal the
                  // Read/Unread chip used to carry
                  sx={{
                    color: row.original.isRead ? 'text.secondary' : 'info.main',
                  }}
                  aria-label={
                    row.original.isRead
                      ? Messages.markAsUnread
                      : Messages.markAsRead
                  }
                  onClick={(e) => {
                    // don't let the click bubble to the row (double-click opens details)
                    e.stopPropagation();
                    handleToggleRead(row.original);
                  }}
                  data-testid={`insight-${row.original.id}-read-state`}
                >
                  {row.original.isRead ? (
                    <DraftsOutlinedIcon />
                  ) : (
                    <MarkunreadOutlinedIcon />
                  )}
                </IconButton>
              </Tooltip>
              <Tooltip title={Messages.actions.more} arrow>
                <IconButton
                  aria-label={Messages.actions.more}
                  onClick={(e) =>
                    setActionMenu({
                      anchor: e.currentTarget,
                      insight: row.original,
                    })
                  }
                  data-testid={`insight-${row.original.id}-actions`}
                >
                  <MoreVertIcon />
                </IconButton>
              </Tooltip>
            </Stack>
          )}
          muiTableBodyRowProps={({ row }) => {
            const check = checksByName.get(row.original.checkName);
            // dim results whose check no longer runs for that row's service,
            // whether it is off globally or only there; unknown checks stay
            // normal to avoid flicker while the advisors list loads
            const checkDisabled = check
              ? !check.enabled || isCheckDisabledForService(row.original)
              : false;

            return {
              'data-testid': `insight-row-${row.original.id}`,
              'data-check-disabled': checkDisabled ? 'true' : undefined,
              onDoubleClick: () => openDetails(row.original),
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
                openDetails(actionMenu.insight);
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
            // the backend silently skips checks disabled for the target, so
            // don't offer the action when this service would be skipped
            disabled={
              isStarting ||
              !actionMenu ||
              !checksByName.get(actionMenu.insight.checkName)?.enabled ||
              isCheckDisabledForService(actionMenu.insight)
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
            // rows recorded before batch grouping have no batch ID
            disabled={!actionMenu?.insight.batchId}
            onClick={() => {
              if (actionMenu) {
                applyBatchIdFilter(actionMenu.insight.batchId);
              }
              setActionMenu(null);
            }}
            data-testid="action-filter-by-batch-id"
          >
            <ListItemIcon>
              <FilterAltOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.filterByBatchId}</ListItemText>
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
            onClick={() => {
              if (actionMenu) {
                handleCopyUrl(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-copy-url"
          >
            <ListItemIcon>
              <LinkOutlinedIcon fontSize="small" />
            </ListItemIcon>
            <ListItemText>{Messages.actions.copyUrl}</ListItemText>
          </MenuItem>
          <MenuItem
            // unknown checks (e.g. removed advisors) cannot be toggled, and a
            // globally disabled check is re-enabled from the Advisor checks page
            disabled={
              isChangingCheck ||
              !actionMenu ||
              !checksByName.get(actionMenu.insight.checkName)?.enabled
            }
            onClick={() => {
              if (actionMenu) {
                handleToggleCheckForService(actionMenu.insight);
              }
              setActionMenu(null);
            }}
            data-testid="action-disable-check"
          >
            <ListItemIcon>
              {actionMenu && isCheckDisabledForService(actionMenu.insight) ? (
                <CheckCircleOutlinedIcon fontSize="small" />
              ) : (
                <BlockOutlinedIcon fontSize="small" />
              )}
            </ListItemIcon>
            <ListItemText>
              {actionMenu && isCheckDisabledForService(actionMenu.insight)
                ? Messages.actions.enableCheck
                : Messages.actions.disableCheck}
            </ListItemText>
          </MenuItem>
        </Menu>
        <InsightDetailsPane
          insight={detailsInsight}
          onClose={() =>
            patchParams((p) => p.delete('insight'), { resetPage: false })
          }
        />
      </Stack>
    </Page>
  );
};

export default AdvisorInsights;
