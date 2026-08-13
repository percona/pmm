import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import FilterAltOffOutlinedIcon from '@mui/icons-material/FilterAltOffOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import RefreshOutlinedIcon from '@mui/icons-material/RefreshOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
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
import { Table } from '@percona/peak-ui';
import { Page } from 'components/page';
import { useRuns } from 'hooks/api/useAdvisors';
import {
  type MRT_PaginationState,
  type MRT_Updater,
} from 'material-react-table';
import { enqueueSnackbar } from 'notistack';
import { FC, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { AdvisorCheckTriggeredBy, AdvisorRun } from 'types/advisors.types';
import { OrgRole } from 'types/user.types';
import { getRunsColumns } from './AdvisorRuns.constants';
import { TRIGGERED_BY_FILTER_OPTIONS } from './AdvisorRuns.filters';
import { Messages } from './AdvisorRuns.messages';
import { isRunning } from './AdvisorRuns.utils';

const DEFAULT_PAGE_SIZE = 50;
const RUNNING_POLL_INTERVAL_MS = 60_000;

interface ActionMenuState {
  anchorEl: HTMLElement;
  run: AdvisorRun;
}

const AdvisorRuns: FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();
  const [actionMenu, setActionMenu] = useState<ActionMenuState | null>(null);

  // the query string is the single source of truth, so links reproduce the view
  const triggeredBy = searchParams.get('triggeredBy') || '';
  const pagination: MRT_PaginationState = {
    pageIndex: Math.max(0, (Number(searchParams.get('page')) || 1) - 1),
    pageSize: Number(searchParams.get('pageSize')) || DEFAULT_PAGE_SIZE,
  };

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

  const handleClearFilters = () => {
    const next = new URLSearchParams();
    const pageSize = searchParams.get('pageSize');
    if (pageSize) {
      next.set('pageSize', pageSize);
    }
    setSearchParams(next, { replace: true });
  };

  const { data, isLoading, isFetching, refetch } = useRuns(
    {
      pageIndex: pagination.pageIndex,
      pageSize: pagination.pageSize,
      triggeredBy: (triggeredBy as AdvisorCheckTriggeredBy) || undefined,
    },
    {
      // a run only shows a duration once it finishes, so poll until none is
      // in flight; the interval belongs to the query, so concurrent runs still
      // cost one request per minute
      refetchInterval: (query) =>
        query.state.data?.results.some(isRunning)
          ? RUNNING_POLL_INTERVAL_MS
          : false,
    }
  );

  const columns = useMemo(() => getRunsColumns(), []);

  const openInsights = (run: AdvisorRun) =>
    navigate(`/advisors/insights?runId=${encodeURIComponent(run.id)}`);

  const copyRunId = (run: AdvisorRun) => {
    void navigator.clipboard.writeText(run.id);
    enqueueSnackbar(Messages.success.runIdCopied, { variant: 'success' });
  };

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
          // the table fills the height and scrolls internally, not the page
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

        <Stack direction="row" gap={2} alignItems="center" flexWrap="wrap">
          <TextField
            select
            size="small"
            label={Messages.filters.triggeredBy}
            value={triggeredBy}
            onChange={(e) =>
              patchParams((p) => {
                if (e.target.value) {
                  p.set('triggeredBy', e.target.value);
                } else {
                  p.delete('triggeredBy');
                }
              })
            }
            sx={{ minWidth: 180 }}
            data-testid="triggeredBy-filter"
          >
            {TRIGGERED_BY_FILTER_OPTIONS.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </TextField>

          <Tooltip title={Messages.filters.refreshTooltip} arrow>
            <IconButton
              onClick={() => void refetch()}
              aria-label={Messages.filters.refresh}
              data-testid="refresh-runs"
            >
              <RefreshOutlinedIcon />
            </IconButton>
          </Tooltip>

          <IconButton
            onClick={handleClearFilters}
            disabled={!triggeredBy}
            aria-label={Messages.filters.clear}
            data-testid="clear-run-filters"
          >
            <FilterAltOffOutlinedIcon />
          </IconButton>
        </Stack>

        <Table
          tableName="advisor-runs-table"
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
          displayColumnDefOptions={{
            'mrt-row-actions': {
              header: Messages.columns.actions,
              size: 70,
              grow: false,
            },
          }}
          // The API takes no sort param, so sorting would only reorder one page
          enableSorting={false}
          muiTableBodyRowProps={({ row }) => ({
            onDoubleClick: () => openInsights(row.original),
            sx: { cursor: 'pointer' },
          })}
          renderRowActions={({ row }) => (
            <IconButton
              size="small"
              aria-label={Messages.actions.more}
              onClick={(e) =>
                setActionMenu({ anchorEl: e.currentTarget, run: row.original })
              }
              data-testid={`run-${row.original.id}-actions`}
            >
              <MoreVertIcon fontSize="small" />
            </IconButton>
          )}
        />
      </Stack>

      <Menu
        open={!!actionMenu}
        anchorEl={actionMenu?.anchorEl}
        onClose={() => setActionMenu(null)}
      >
        <MenuItem
          onClick={() => {
            if (actionMenu) {
              openInsights(actionMenu.run);
            }
            setActionMenu(null);
          }}
          data-testid="action-view-run-insights"
        >
          <ListItemIcon>
            <VisibilityOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.viewInsights}</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            if (actionMenu) {
              copyRunId(actionMenu.run);
            }
            setActionMenu(null);
          }}
          data-testid="action-copy-run-id"
        >
          <ListItemIcon>
            <ContentCopyOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.copyRunId}</ListItemText>
        </MenuItem>
      </Menu>
    </Page>
  );
};

export default AdvisorRuns;
