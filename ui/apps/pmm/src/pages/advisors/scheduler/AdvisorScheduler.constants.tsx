import WarningAmberOutlinedIcon from '@mui/icons-material/WarningAmberOutlined';
import CircularProgress from '@mui/material/CircularProgress';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { ADVISOR_TECHNOLOGY } from 'lib/constants';
import { type MRT_ColumnDef } from 'material-react-table';
import { type FC } from 'react';
import {
  type AdvisorCheck,
  type AdvisorRun,
  AdvisorRunStatus,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  type AdvisorTargetService,
} from 'types/advisors.types';
import { Messages } from './AdvisorScheduler.messages';
import {
  describeDays,
  formatUtc,
  localHint,
  missingTargets,
  nextRuns,
  relativeTime,
  scopeChecks,
  scopeServices,
  severitySummary,
} from './AdvisorScheduler.utils';
import { StatusDot, Swatch, TwoLine } from './AdvisorScheduler.components';

const MailOffFlag: FC = () => (
  <Tooltip title={Messages.email.notificationsOff} arrow>
    <WarningAmberOutlinedIcon
      fontSize="inherit"
      color="warning"
      aria-label={Messages.email.notificationsOff}
      data-testid="schedule-email-off-flag"
    />
  </Tooltip>
);

export const RUN_STATUS_COLOR: Partial<Record<AdvisorRunStatus, string>> = {
  [AdvisorRunStatus.completed]: 'success.main',
  [AdvisorRunStatus.interrupted]: 'warning.main',
  [AdvisorRunStatus.aborted]: 'error.main',
};

interface LastRunProps {
  run?: AdvisorRun | null;
  now: Date;
  onOpenRun: (runId: string) => void;
}

export const LastRun: FC<LastRunProps> = ({ run, now, onOpenRun }) => {
  if (!run) {
    return <TwoLine primary={Messages.last.never} muted />;
  }
  const started = new Date(run.startedAt);
  if (
    run.status === AdvisorRunStatus.queued ||
    run.status === AdvisorRunStatus.running
  ) {
    return (
      <TwoLine
        primary={
          <>
            <CircularProgress size={12} />
            {run.status === AdvisorRunStatus.queued
              ? Messages.last.queued
              : Messages.last.running(
                  run.checksCount ?? 0,
                  run.plannedChecksCount ?? 0
                )}
          </>
        }
        secondary={Messages.last.started(relativeTime(started, now))}
        testId="schedule-last-run"
      />
    );
  }
  const label =
    run.status === AdvisorRunStatus.interrupted
      ? Messages.last.interrupted
      : run.status === AdvisorRunStatus.aborted
        ? Messages.last.aborted
        : Messages.last.findings(run.findingsCount ?? 0);
  return (
    <TwoLine
      primary={
        <>
          <StatusDot color={RUN_STATUS_COLOR[run.status] ?? 'text.disabled'} />
          <Tooltip title={Messages.last.openRun} arrow>
            <Link
              component="button"
              variant="body2"
              onClick={(e) => {
                e.stopPropagation();
                onOpenRun(run.id);
              }}
            >
              {label}
            </Link>
          </Tooltip>
        </>
      }
      secondary={formatUtc(started)}
      testId="schedule-last-run"
    />
  );
};

export interface ScheduleColumnsContext {
  now: Date;
  advisorsOn: boolean;
  notificationsOn: boolean;
  colors: Map<string, string>;
  // undefined until loaded
  checks?: AdvisorCheck[];
  services?: AdvisorTargetService[];
  settingsRecipients?: string[];
  onToggle: (schedule: AdvisorSchedule) => void;
  onOpenRun: (runId: string) => void;
}

const emailCell = (
  schedule: AdvisorSchedule,
  { notificationsOn, settingsRecipients }: ScheduleColumnsContext
) => {
  if (schedule.emailMode === AdvisorScheduleEmailMode.off) {
    return <TwoLine primary={Messages.email.none} muted />;
  }
  const flag = notificationsOn ? null : <MailOffFlag />;
  const severities = severitySummary(schedule.emailSeverities);
  if (schedule.emailMode === AdvisorScheduleEmailMode.custom) {
    const addresses = schedule.emailAddresses ?? [];
    return (
      <TwoLine
        primary={
          <>
            {addresses[0]}
            {addresses.length > 1 &&
              ` ${Messages.email.more(addresses.length - 1)}`}
            {flag}
          </>
        }
        secondary={[Messages.email.ownRecipients, severities]
          .filter(Boolean)
          .join(' · ')}
      />
    );
  }
  return (
    <TwoLine
      primary={
        <>
          {Messages.email.settingsRecipients}
          {flag}
        </>
      }
      secondary={[
        settingsRecipients &&
          Messages.email.addresses(settingsRecipients.length),
        severities,
      ]
        .filter(Boolean)
        .join(' · ')}
    />
  );
};

const nextRunCell = (
  schedule: AdvisorSchedule,
  { advisorsOn, now }: ScheduleColumnsContext
) => {
  if (!advisorsOn) {
    return (
      <TwoLine
        primary={Messages.next.notScheduled}
        secondary={Messages.next.advisorsOff}
        muted
      />
    );
  }
  if (!schedule.enabled) {
    return <TwoLine primary={Messages.next.paused} muted />;
  }
  const next = nextRuns(
    schedule.daysOfWeek ?? [],
    schedule.timeOfDay,
    now,
    1
  )[0];
  if (!next) {
    return <TwoLine primary="—" muted />;
  }
  return (
    <TwoLine
      primary={formatUtc(next)}
      secondary={relativeTime(next, now)}
      testId="schedule-next-run"
    />
  );
};

export const getScheduleColumns = (
  ctx: ScheduleColumnsContext
): MRT_ColumnDef<AdvisorSchedule>[] => [
  {
    id: 'name',
    header: Messages.columns.schedule,
    accessorKey: 'name',
    size: 220,
    grow: true,
    Cell: ({ row }) => {
      const schedule = row.original;
      const missing = missingTargets(schedule, ctx.checks, ctx.services);
      return (
        <Stack gap={0.5}>
          <Stack direction="row" alignItems="center" gap={1.25}>
            <Swatch
              color={ctx.colors.get(schedule.scheduleId) ?? ''}
              paused={!schedule.enabled}
            />
            <Typography variant="body2" fontWeight={500}>
              {schedule.name}
            </Typography>
          </Stack>
          {(missing.services || missing.checks) && (
            <Tooltip
              title={
                missing.services
                  ? Messages.targetsNoServiceTooltip
                  : Messages.targetsNoCheckTooltip
              }
              arrow
            >
              <Typography
                variant="caption"
                color="warning.main"
                sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}
                data-testid={`schedule-${schedule.scheduleId}-targets-nothing`}
              >
                <WarningAmberOutlinedIcon fontSize="inherit" />
                {missing.services
                  ? Messages.targetsNoService
                  : Messages.targetsNoCheck}
              </Typography>
            </Tooltip>
          )}
        </Stack>
      );
    },
  },
  {
    id: 'when',
    header: Messages.columns.when,
    size: 190,
    grow: false,
    Cell: ({ row }) => {
      const schedule = row.original;
      const next = nextRuns(
        schedule.daysOfWeek ?? [],
        schedule.timeOfDay,
        ctx.now,
        1
      )[0];
      return (
        <TwoLine
          primary={`${describeDays(schedule.daysOfWeek ?? [])}, ${schedule.timeOfDay} UTC`}
          secondary={localHint(next)}
        />
      );
    },
  },
  {
    id: 'checks',
    header: Messages.columns.checks,
    size: 160,
    grow: false,
    Cell: ({ row }) => {
      const names = row.original.checkNames ?? [];
      if (!names.length) {
        return (
          <TwoLine
            primary={Messages.allChecks}
            secondary={ctx.checks && Messages.today(ctx.checks.length)}
          />
        );
      }
      const technologies = [
        ...new Set(
          scopeChecks(ctx.checks ?? [], names).map(
            (c) => ADVISOR_TECHNOLOGY[c.technology]
          )
        ),
      ];
      return (
        <TwoLine
          primary={Messages.checksCount(names.length)}
          secondary={technologies.join(', ')}
        />
      );
    },
  },
  {
    id: 'services',
    header: Messages.columns.services,
    size: 180,
    grow: false,
    Cell: ({ row }) => {
      const ids = row.original.serviceIds ?? [];
      if (!ids.length) {
        return (
          <TwoLine
            primary={Messages.allServices}
            secondary={ctx.services && Messages.today(ctx.services.length)}
          />
        );
      }
      const names = scopeServices(ctx.services ?? [], ids).map(
        (s) => s.serviceName
      );
      return (
        <TwoLine
          primary={Messages.servicesCount(ids.length)}
          secondary={`${names.slice(0, 2).join(', ')}${names.length > 2 ? '…' : ''}`}
        />
      );
    },
  },
  {
    id: 'email',
    header: Messages.columns.email,
    size: 200,
    grow: false,
    Cell: ({ row }) => emailCell(row.original, ctx),
  },
  {
    id: 'nextRun',
    header: Messages.columns.nextRun,
    size: 190,
    grow: false,
    Cell: ({ row }) => nextRunCell(row.original, ctx),
  },
  {
    id: 'lastRun',
    header: Messages.columns.lastRun,
    size: 190,
    grow: false,
    Cell: ({ row }) => (
      <LastRun
        run={row.original.lastRun}
        now={ctx.now}
        onOpenRun={ctx.onOpenRun}
      />
    ),
  },
  {
    id: 'enabled',
    header: Messages.columns.enabled,
    size: 100,
    grow: false,
    Cell: ({ row }) => (
      <Tooltip
        title={
          row.original.enabled ? Messages.toggle.pause : Messages.toggle.enable
        }
        arrow
      >
        <Switch
          size="small"
          checked={!!row.original.enabled}
          onDoubleClick={(e) => e.stopPropagation()}
          onChange={() => ctx.onToggle(row.original)}
          slotProps={{
            input: { 'aria-label': Messages.toggle.label(row.original.name) },
          }}
          data-testid={`schedule-${row.original.scheduleId}-enabled`}
        />
      </Tooltip>
    ),
  },
];
