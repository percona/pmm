import AddOutlinedIcon from '@mui/icons-material/AddOutlined';
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined';
import DeleteOutlineOutlinedIcon from '@mui/icons-material/DeleteOutlineOutlined';
import EditOutlinedIcon from '@mui/icons-material/EditOutlined';
import HistoryOutlinedIcon from '@mui/icons-material/HistoryOutlined';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import PlayArrowOutlinedIcon from '@mui/icons-material/PlayArrowOutlined';
import RefreshOutlinedIcon from '@mui/icons-material/RefreshOutlined';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';
import Dialog from '@mui/material/Dialog';
import DialogActions from '@mui/material/DialogActions';
import DialogContent from '@mui/material/DialogContent';
import DialogContentText from '@mui/material/DialogContentText';
import DialogTitle from '@mui/material/DialogTitle';
import Divider from '@mui/material/Divider';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { Table } from '@percona/peak-ui';
import { Page } from 'components/page';
import { useSettings } from 'contexts/settings';
import {
  useAdvisorChecks,
  useAdvisorNotificationSettings,
  useAdvisorSchedules,
  useAdvisorServices,
  useDeleteAdvisorSchedule,
  useRunAdvisorSchedule,
  useRuns,
  useSaveAdvisorSchedule,
} from 'hooks/api/useAdvisors';
import { enqueueSnackbar } from 'notistack';
import { type FC, useCallback, useEffect, useMemo, useState } from 'react';
import {
  Link as RouterLink,
  useNavigate,
  useSearchParams,
} from 'react-router-dom';
import { AdvisorRunStatus, type AdvisorSchedule } from 'types/advisors.types';
import { OrgRole } from 'types/user.types';
import {
  ACTIVE_RUN_POLL_INTERVAL_MS,
  isRunning,
} from '../runs/AdvisorRuns.utils';
import { getScheduleColumns } from './AdvisorScheduler.constants';
import { Messages } from './AdvisorScheduler.messages';
import {
  copyName,
  coveredChecks,
  DEFAULT_SCHEDULE_INPUT,
  formatUtc,
  nextRuns,
  offsetLabel,
  relativeTime,
  scheduleColors,
  scopeChecks,
  scopeServices,
  serviceCoverage,
  timeZoneLabel,
  toScheduleInput,
} from './AdvisorScheduler.utils';
import { ScheduleDetailsPane } from './ScheduleDetailsPane';
import { ScheduleEditor } from './schedule-editor';
import { ScheduleWeek } from './ScheduleWeek';

// how often the page refreshes when nothing runs, so times and last runs stay current
const IDLE_REFRESH_MS = 60_000;

const nextRunText = (
  schedule: Pick<AdvisorSchedule, 'daysOfWeek' | 'timeOfDay'>
) => {
  const next = nextRuns(
    schedule.daysOfWeek ?? [],
    schedule.timeOfDay,
    new Date(),
    1
  )[0];
  return next ? formatUtc(next) : '';
};

interface EditorState {
  schedule?: AdvisorSchedule;
}

interface MenuState {
  anchorEl: HTMLElement;
  schedule: AdvisorSchedule;
}

const AdvisorScheduler: FC = () => {
  const navigate = useNavigate();
  const theme = useTheme();
  const { settings } = useSettings();
  const [searchParams, setSearchParams] = useSearchParams();
  const [now, setNow] = useState(() => new Date());
  const [editor, setEditor] = useState<EditorState | null>(null);
  const [menu, setMenu] = useState<MenuState | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<AdvisorSchedule | null>(
    null
  );

  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), IDLE_REFRESH_MS);
    return () => clearInterval(id);
  }, []);

  // the most recent run tells whether one is in progress, scheduled or not
  const { data: latestRuns } = useRuns(
    { pageIndex: 0, pageSize: 1 },
    {
      refetchInterval: (query) =>
        query.state.data?.results.some(isRunning)
          ? ACTIVE_RUN_POLL_INTERVAL_MS
          : IDLE_REFRESH_MS,
    }
  );
  const activeRun = latestRuns?.results.find(isRunning);
  const {
    data: schedules = [],
    isLoading,
    isFetching,
    refetch,
  } = useAdvisorSchedules({
    refetchInterval: activeRun ? ACTIVE_RUN_POLL_INTERVAL_MS : IDLE_REFRESH_MS,
  });
  const { data: checks } = useAdvisorChecks();
  const { data: services } = useAdvisorServices();
  const { data: notification } = useAdvisorNotificationSettings();

  const { mutate: save } = useSaveAdvisorSchedule();
  const { mutate: remove, isPending: isDeleting } = useDeleteAdvisorSchedule();
  const { mutate: runSchedule, isPending: isStarting } =
    useRunAdvisorSchedule();

  const advisorsOn = !!settings?.advisorEnabled;
  const notificationsOn = notification ? !!notification.enabled : true;
  const colors = useMemo(
    () => scheduleColors(schedules, theme.palette.mode),
    [schedules, theme.palette.mode]
  );

  const detailsId = searchParams.get('details') || '';
  const details = schedules.find((s) => s.scheduleId === detailsId) ?? null;

  const setDetails = (scheduleId?: string) => {
    const next = new URLSearchParams(searchParams);
    if (scheduleId) {
      next.set('details', scheduleId);
    } else {
      next.delete('details');
    }
    setSearchParams(next, { replace: true });
  };

  const openRun = useCallback(
    (runId: string) =>
      navigate(`/advisors/runs?runId=${encodeURIComponent(runId)}`),
    [navigate]
  );

  const coverageOf = useCallback(
    (schedule: AdvisorSchedule) => {
      const scoped = scopeServices(services ?? [], schedule.serviceIds);
      const checksInScope = scopeChecks(checks ?? [], schedule.checkNames);
      return {
        checks: coveredChecks(checksInScope, scoped),
        services: serviceCoverage(checksInScope, scoped).filter((r) => r.checks)
          .length,
      };
    },
    [checks, services]
  );

  const toggle = useCallback(
    (schedule: AdvisorSchedule) => {
      const enabled = !schedule.enabled;
      save(
        {
          scheduleId: schedule.scheduleId,
          input: toScheduleInput(schedule, { enabled }),
        },
        {
          onSuccess: (saved) =>
            enqueueSnackbar(
              enabled
                ? Messages.success.enabled(saved.name, nextRunText(saved))
                : Messages.success.paused(saved.name),
              { variant: 'success' }
            ),
        }
      );
    },
    [save]
  );

  const runNow = (schedule: AdvisorSchedule) =>
    runSchedule(schedule.scheduleId, {
      onSuccess: () =>
        enqueueSnackbar(Messages.success.started(schedule.name), {
          variant: 'success',
        }),
    });

  // a copy starts paused, so nothing runs twice until it is changed
  const duplicate = (schedule: AdvisorSchedule) =>
    save(
      {
        input: toScheduleInput(schedule, {
          name: copyName(
            schedule.name,
            schedules.map((s) => s.name)
          ),
          enabled: false,
        }),
      },
      {
        onSuccess: (saved) =>
          enqueueSnackbar(Messages.success.duplicated(saved.name), {
            variant: 'success',
          }),
      }
    );

  const restoreDefault = () =>
    save(
      { input: DEFAULT_SCHEDULE_INPUT },
      {
        onSuccess: (saved) =>
          enqueueSnackbar(
            Messages.success.created(saved.name, nextRunText(saved)),
            { variant: 'success' }
          ),
      }
    );

  const confirmDelete = () => {
    if (!deleteTarget) {
      return;
    }
    const target = deleteTarget;
    remove(target.scheduleId, {
      onSuccess: () => {
        enqueueSnackbar(Messages.success.deleted(target.name), {
          variant: 'success',
        });
        if (detailsId === target.scheduleId) {
          setDetails();
        }
        setDeleteTarget(null);
      },
    });
  };

  const columns = useMemo(
    () =>
      getScheduleColumns({
        now,
        advisorsOn,
        notificationsOn,
        colors,
        checks,
        services,
        settingsRecipients: notification
          ? (notification.emailAddresses ?? [])
          : undefined,
        onToggle: toggle,
        onOpenRun: openRun,
      }),
    [
      now,
      advisorsOn,
      notificationsOn,
      colors,
      checks,
      services,
      notification,
      toggle,
      openRun,
    ]
  );

  const runBlocked = !!activeRun || !advisorsOn || isStarting;

  const renderBanners = () => (
    <>
      {settings && !advisorsOn && (
        <Alert severity="warning" data-testid="schedules-advisors-off">
          {Messages.banners.advisorsOff}{' '}
          <Link component={RouterLink} to="/settings/advisors" color="inherit">
            {Messages.settingsLink}
          </Link>
          .
        </Alert>
      )}
      {activeRun && (
        <Alert
          severity="info"
          icon={<CircularProgress size={16} />}
          data-testid="schedules-run-active"
        >
          <b>{activeRun.scheduleName || Messages.banners.aRun}</b>{' '}
          {activeRun.status === AdvisorRunStatus.queued
            ? Messages.banners.queued
            : Messages.banners.running(
                activeRun.checksCount ?? 0,
                activeRun.plannedChecksCount ?? 0,
                relativeTime(new Date(activeRun.startedAt), now)
              )}{' '}
          <Link
            component="button"
            color="inherit"
            onClick={() => openRun(activeRun.id)}
            sx={{ verticalAlign: 'baseline' }}
          >
            {Messages.banners.openRun}
          </Link>
        </Alert>
      )}
      {advisorsOn && !notificationsOn && schedules.length > 0 && (
        <Alert severity="warning" data-testid="schedules-notifications-off">
          {Messages.banners.notificationsOff}{' '}
          <Link component={RouterLink} to="/settings/advisors" color="inherit">
            {Messages.settingsLink}
          </Link>
          .
        </Alert>
      )}
    </>
  );

  return (
    <Page
      title={Messages.title}
      maxWidth="full"
      footer={null}
      roles={[OrgRole.Editor, OrgRole.Admin]}
    >
      <Stack gap={2.5}>
        <Typography variant="body2">{Messages.description}</Typography>

        <Stack
          direction="row"
          flexWrap="wrap"
          alignItems="center"
          columnGap={1.5}
          rowGap={1}
        >
          <Button
            variant="contained"
            startIcon={<AddOutlinedIcon />}
            onClick={() => setEditor({})}
            data-testid="schedule-new"
          >
            {Messages.newSchedule}
          </Button>
          <Tooltip title={Messages.reload} arrow>
            <IconButton
              aria-label={Messages.reload}
              onClick={() => void refetch()}
              data-testid="schedules-refresh"
            >
              <RefreshOutlinedIcon />
            </IconButton>
          </Tooltip>
          <Box sx={{ flex: 1 }} />
          <Typography
            variant="body2"
            color="text.secondary"
            data-testid="schedules-time-zone"
          >
            {Messages.allTimesAre} <b>UTC</b>.{' '}
            {Messages.yourTimeZone(timeZoneLabel(), offsetLabel(now))}
          </Typography>
        </Stack>

        {renderBanners()}

        {!isLoading && (
          <ScheduleWeek
            schedules={schedules}
            colors={colors}
            now={now}
            advisorsOn={advisorsOn}
            coverageOf={coverageOf}
            onOpen={(s) => setDetails(s.scheduleId)}
          />
        )}

        {!isLoading && !schedules.length ? (
          <Stack
            gap={1.25}
            alignItems="flex-start"
            sx={{
              border: 1,
              borderStyle: 'dashed',
              borderColor: 'divider',
              borderRadius: 2,
              px: 3,
              py: 4.5,
            }}
            data-testid="schedules-empty"
          >
            <Typography variant="h6" component="h2">
              {Messages.empty.title}
            </Typography>
            <Typography
              variant="body2"
              color="text.secondary"
              sx={{ maxWidth: '62ch' }}
            >
              {Messages.empty.body}
            </Typography>
            <Stack direction="row" gap={1.25} flexWrap="wrap" sx={{ mt: 0.75 }}>
              <Button
                variant="contained"
                onClick={restoreDefault}
                data-testid="schedules-restore-default"
              >
                {Messages.empty.restore}
              </Button>
              <Button variant="outlined" onClick={() => setEditor({})}>
                {Messages.newSchedule}
              </Button>
            </Stack>
          </Stack>
        ) : (
          <Table
            tableName="advisor-schedules-table"
            columns={columns}
            data={schedules}
            enableTopToolbar={false}
            enableGlobalFilter={false}
            enableColumnFilters={false}
            enableColumnActions={false}
            enableHiding={false}
            enableSorting={false}
            state={{ isLoading, showProgressBars: isFetching && !isLoading }}
            enableRowActions
            positionActionsColumn="last"
            displayColumnDefOptions={{
              'mrt-row-actions': {
                header: Messages.columns.actions,
                size: 70,
                grow: false,
              },
            }}
            renderRowActions={({ row }) => (
              <IconButton
                size="small"
                aria-label={Messages.actions.more(row.original.name)}
                onClick={(e) => {
                  e.stopPropagation();
                  setMenu({
                    anchorEl: e.currentTarget,
                    schedule: row.original,
                  });
                }}
                data-testid={`schedule-${row.original.scheduleId}-actions`}
              >
                <MoreVertIcon fontSize="small" />
              </IconButton>
            )}
            muiTableBodyRowProps={({ row }) => ({
              'data-testid': `schedule-row-${row.original.scheduleId}`,
              onDoubleClick: () => setDetails(row.original.scheduleId),
              selected: row.original.scheduleId === detailsId,
              sx: { cursor: 'pointer' },
            })}
            muiTableContainerProps={{
              sx: {
                borderRadius: 2,
                border: '1px solid',
                borderColor: 'divider',
              },
            }}
          />
        )}
      </Stack>

      <Menu
        open={!!menu}
        anchorEl={menu?.anchorEl}
        onClose={() => setMenu(null)}
      >
        {[
          {
            label: Messages.actions.runNow,
            icon: <PlayArrowOutlinedIcon fontSize="small" />,
            disabled: runBlocked,
            act: runNow,
            testId: 'schedule-action-run',
          },
          {
            label: Messages.actions.edit,
            icon: <EditOutlinedIcon fontSize="small" />,
            act: (s: AdvisorSchedule) => setEditor({ schedule: s }),
            testId: 'schedule-action-edit',
          },
          {
            label: Messages.actions.duplicate,
            icon: <ContentCopyOutlinedIcon fontSize="small" />,
            act: duplicate,
            testId: 'schedule-action-duplicate',
          },
          {
            label: Messages.actions.viewRuns,
            icon: <HistoryOutlinedIcon fontSize="small" />,
            act: (s: AdvisorSchedule) =>
              navigate(
                `/advisors/runs?scheduleId=${encodeURIComponent(s.scheduleId)}`
              ),
            testId: 'schedule-action-runs',
          },
        ].map(({ label, icon, disabled, act, testId }) => (
          <MenuItem
            key={testId}
            disabled={disabled}
            onClick={() => {
              if (menu) {
                act(menu.schedule);
              }
              setMenu(null);
            }}
            data-testid={testId}
          >
            <ListItemIcon>{icon}</ListItemIcon>
            <ListItemText>{label}</ListItemText>
          </MenuItem>
        ))}
        <Divider />
        <MenuItem
          onClick={() => {
            if (menu) {
              setDeleteTarget(menu.schedule);
            }
            setMenu(null);
          }}
          sx={{ color: 'error.main' }}
          data-testid="schedule-action-delete"
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <DeleteOutlineOutlinedIcon fontSize="small" />
          </ListItemIcon>
          <ListItemText>{Messages.actions.delete}</ListItemText>
        </MenuItem>
      </Menu>

      <ScheduleDetailsPane
        schedule={details}
        color={details ? (colors.get(details.scheduleId) ?? '') : ''}
        now={now}
        advisorsOn={advisorsOn}
        runActive={runBlocked}
        checks={checks}
        services={services}
        notification={notification}
        onClose={() => setDetails()}
        onRun={runNow}
        onEdit={(s) => setEditor({ schedule: s })}
        onOpenRun={openRun}
      />

      <ScheduleEditor
        open={!!editor}
        schedule={editor?.schedule}
        schedules={schedules}
        checks={checks}
        services={services}
        notification={notification}
        onClose={() => setEditor(null)}
      />

      <Dialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        data-testid="schedule-delete-dialog"
      >
        <DialogTitle>
          {deleteTarget ? Messages.deleteDialog.title(deleteTarget.name) : ''}
        </DialogTitle>
        <DialogContent>
          <DialogContentText>
            {schedules.length === 1
              ? Messages.deleteDialog.last
              : Messages.deleteDialog.notLast}{' '}
            {Messages.deleteDialog.history}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button color="inherit" onClick={() => setDeleteTarget(null)}>
            {Messages.deleteDialog.cancel}
          </Button>
          <Button
            color="error"
            variant="contained"
            disabled={isDeleting}
            onClick={confirmDelete}
            data-testid="schedule-delete-confirm"
          >
            {Messages.deleteDialog.confirm}
          </Button>
        </DialogActions>
      </Dialog>
    </Page>
  );
};

export default AdvisorScheduler;
