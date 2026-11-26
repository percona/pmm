import CloseIcon from '@mui/icons-material/Close';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Accordion from '@mui/material/Accordion';
import AccordionDetails from '@mui/material/AccordionDetails';
import AccordionSummary from '@mui/material/AccordionSummary';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Chip from '@mui/material/Chip';
import Drawer from '@mui/material/Drawer';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableFooter from '@mui/material/TableFooter';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Typography from '@mui/material/Typography';
import { useRuns } from 'hooks/api/useAdvisors';
import { SEVERITY } from 'lib/constants';
import { type FC, type ReactNode } from 'react';
import {
  type AdvisorCheck,
  type AdvisorNotificationSettings,
  AdvisorRunStatus,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  type AdvisorTargetService,
} from 'types/advisors.types';
import {
  MONO_FONT,
  StatusDot,
  Swatch,
  TechnologyLabel,
} from './AdvisorScheduler.components';
import { RUN_STATUS_COLOR } from './AdvisorScheduler.constants';
import { Messages } from './AdvisorScheduler.messages';
import {
  coveredChecks,
  describeDays,
  formatLocalWeekdayTime,
  formatUtcDay,
  formatUtc,
  formatUtcTime,
  isUtcViewer,
  joinAnd,
  localHint,
  nextRuns,
  relativeTime,
  SCHEDULE_TECHNOLOGIES,
  scopeChecks,
  scopeServices,
  serviceCoverage,
} from './AdvisorScheduler.utils';

const RUN_STATUS_LABEL: Partial<Record<AdvisorRunStatus, string>> = {
  [AdvisorRunStatus.queued]: Messages.runStatus.queued,
  [AdvisorRunStatus.running]: Messages.runStatus.running,
  [AdvisorRunStatus.completed]: Messages.runStatus.completed,
  [AdvisorRunStatus.interrupted]: Messages.runStatus.interrupted,
  [AdvisorRunStatus.aborted]: Messages.runStatus.aborted,
};

const Section: FC<{ title: string; children: ReactNode; testId?: string }> = ({
  title,
  children,
  testId,
}) => (
  <Box component="section" data-testid={testId}>
    <Typography
      variant="overline"
      component="h3"
      color="text.secondary"
      fontWeight={600}
      sx={{ display: 'block', mb: 1, lineHeight: 1.5 }}
    >
      {title}
    </Typography>
    {children}
  </Box>
);

const Note: FC<{ children: ReactNode }> = ({ children }) => (
  <Typography variant="body2" color="text.secondary" sx={{ mt: 0.75 }}>
    {children}
  </Typography>
);

const severityNames = (
  severities?: AdvisorNotificationSettings['severities']
) => joinAnd((severities ?? []).map((s) => SEVERITY[s]));

interface ScheduleDetailsPaneProps {
  schedule: AdvisorSchedule | null;
  color: string;
  now: Date;
  advisorsOn: boolean;
  runActive: boolean;
  checks?: AdvisorCheck[];
  services?: AdvisorTargetService[];
  notification?: AdvisorNotificationSettings;
  onClose: () => void;
  onRun: (schedule: AdvisorSchedule) => void;
  onEdit: (schedule: AdvisorSchedule) => void;
  onOpenRun: (runId: string) => void;
}

export const ScheduleDetailsPane: FC<ScheduleDetailsPaneProps> = ({
  schedule,
  color,
  now,
  advisorsOn,
  runActive,
  checks = [],
  services = [],
  notification,
  onClose,
  onRun,
  onEdit,
  onOpenRun,
}) => {
  const { data: recentRuns } = useRuns(
    { scheduleId: schedule?.scheduleId, pageIndex: 0, pageSize: 3 },
    { enabled: !!schedule }
  );

  const renderBody = (s: AdvisorSchedule) => {
    const days = s.daysOfWeek ?? [];
    const scopedChecks = scopeChecks(checks, s.checkNames);
    const scopedServices = scopeServices(services, s.serviceIds);
    const rows = serviceCoverage(scopedChecks, scopedServices);
    const idle = rows.filter((r) => !r.checks);
    const upcoming =
      advisorsOn && s.enabled ? nextRuns(days, s.timeOfDay, now, 5) : [];
    const hint = localHint(
      upcoming[0] ?? nextRuns(days, s.timeOfDay, now, 1)[0]
    );
    const missingServices = (s.serviceIds ?? []).length - scopedServices.length;

    const emailText =
      s.emailMode === AdvisorScheduleEmailMode.off
        ? Messages.email.none
        : s.emailMode === AdvisorScheduleEmailMode.custom
          ? (s.emailAddresses ?? []).join(', ')
          : notification?.emailAddresses?.length
            ? Messages.pane.settingsRecipients(
                notification.emailAddresses.join(', ')
              )
            : notification
              ? Messages.pane.noSettingsRecipients
              : Messages.email.settingsRecipients;
    const severityText =
      s.emailMode === AdvisorScheduleEmailMode.off
        ? ''
        : s.emailSeverities?.length
          ? Messages.pane.ownSeverities(severityNames(s.emailSeverities))
          : notification?.severities?.length
            ? Messages.pane.settingsSeverities(
                severityNames(notification.severities)
              )
            : '';

    return (
      <Stack gap={2.75} sx={{ overflowY: 'auto', px: 2.5, pt: 2, pb: 3.5 }}>
        <Stack direction="row" gap={1} flexWrap="wrap">
          <Button
            variant="contained"
            disabled={runActive || !advisorsOn}
            onClick={() => onRun(s)}
            data-testid="schedule-pane-run"
          >
            {Messages.actions.runNow}
          </Button>
          <Button
            variant="outlined"
            onClick={() => onEdit(s)}
            data-testid="schedule-pane-edit"
          >
            {Messages.actions.edit}
          </Button>
        </Stack>

        <Box
          component="dl"
          sx={{
            display: 'grid',
            gridTemplateColumns: '110px minmax(0, 1fr)',
            columnGap: 1.75,
            rowGap: 0.75,
            m: 0,
            '& dt': { color: 'text.secondary' },
            '& dd': { m: 0 },
          }}
          data-testid="schedule-pane-summary"
        >
          <dt>{Messages.pane.when}</dt>
          <dd>
            {Messages.pane.whenValue(describeDays(days), s.timeOfDay)}
            {hint && (
              <Typography component="span" color="text.secondary">
                {` (${hint})`}
              </Typography>
            )}
          </dd>
          <dt>{Messages.pane.checks}</dt>
          <dd>
            {s.checkNames?.length
              ? Messages.pane.selected(s.checkNames.length)
              : Messages.pane.allChecks(checks.length)}
          </dd>
          <dt>{Messages.pane.services}</dt>
          <dd>
            {s.serviceIds?.length
              ? [
                  Messages.pane.selected(s.serviceIds.length),
                  missingServices > 0 && Messages.pane.missing(missingServices),
                ]
                  .filter(Boolean)
                  .join(', ')
              : Messages.pane.allServices(services.length)}
          </dd>
          <dt>{Messages.pane.email}</dt>
          <dd>
            {emailText}
            {severityText && (
              <Typography
                variant="body2"
                color="text.secondary"
                sx={{ display: 'block' }}
              >
                {severityText}
              </Typography>
            )}
          </dd>
        </Box>

        <Section
          title={Messages.pane.nextRuns}
          testId="schedule-pane-next-runs"
        >
          {upcoming.length ? (
            <Table size="small">
              <TableBody>
                {upcoming.map((at) => (
                  <TableRow key={at.getTime()}>
                    <TableCell sx={{ fontFamily: MONO_FONT }}>
                      {formatUtcDay(at)}
                    </TableCell>
                    <TableCell sx={{ fontFamily: MONO_FONT }}>
                      {`${formatUtcTime(at)} UTC`}
                    </TableCell>
                    <TableCell sx={{ color: 'text.secondary' }}>
                      {isUtcViewer(at)
                        ? ''
                        : Messages.pane.yourTime(formatLocalWeekdayTime(at))}
                    </TableCell>
                    <TableCell sx={{ color: 'text.secondary' }}>
                      {relativeTime(at, now)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <Note>
              {advisorsOn
                ? Messages.pane.pausedNote
                : Messages.pane.nothingWhileOff}
            </Note>
          )}
        </Section>

        <Section title={Messages.pane.coverage} testId="schedule-pane-coverage">
          <Box sx={{ overflowX: 'auto' }}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>{Messages.pane.service}</TableCell>
                  <TableCell>{Messages.pane.technology}</TableCell>
                  <TableCell>{Messages.pane.environment}</TableCell>
                  <TableCell>{Messages.pane.cluster}</TableCell>
                  <TableCell align="right">
                    {Messages.pane.checksColumn}
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {rows.map(({ service, checks: count }) => (
                  <TableRow key={service.serviceId}>
                    <TableCell>{service.serviceName}</TableCell>
                    <TableCell>
                      <TechnologyLabel technology={service.technology} />
                    </TableCell>
                    <TableCell>{service.environment}</TableCell>
                    <TableCell>{service.cluster}</TableCell>
                    <TableCell align="right">{count}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
              <TableFooter>
                <TableRow>
                  <TableCell
                    colSpan={4}
                    sx={{ fontWeight: 500, color: 'text.primary' }}
                  >
                    {Messages.pane.coverageTotal(
                      coveredChecks(scopedChecks, scopedServices),
                      rows.length - idle.length
                    )}
                  </TableCell>
                  <TableCell
                    align="right"
                    sx={{ fontWeight: 500, color: 'text.primary' }}
                  >
                    {rows.reduce((sum, r) => sum + r.checks, 0)}
                  </TableCell>
                </TableRow>
              </TableFooter>
            </Table>
          </Box>
          <Note>
            {Messages.pane.coverageNote}
            {idle.length > 0 &&
              ` ${Messages.pane.noCheckApplies(idle.map((r) => r.service.serviceName).join(', '))}`}
          </Note>
        </Section>

        <Section title={Messages.pane.checks} testId="schedule-pane-checks">
          {SCHEDULE_TECHNOLOGIES.map((technology) => {
            const list = scopedChecks.filter(
              (c) => c.technology === technology
            );
            if (!list.length) {
              return null;
            }
            const matched = scopedServices.some(
              (v) => v.technology === technology
            );
            return (
              <Accordion
                key={technology}
                disableGutters
                elevation={0}
                square
                sx={{
                  borderTop: 1,
                  borderColor: 'divider',
                  '&::before': { display: 'none' },
                }}
              >
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Stack direction="row" gap={1} alignItems="center">
                    <TechnologyLabel technology={technology} />
                    <Typography variant="body2" color="text.secondary">
                      {matched
                        ? Messages.pane.checksByTechnology(list.length)
                        : `${Messages.pane.checksByTechnology(list.length)}, ${Messages.pane.noMatchingService}`}
                    </Typography>
                  </Stack>
                </AccordionSummary>
                <AccordionDetails>
                  <Box
                    component="ul"
                    sx={{
                      m: 0,
                      pl: 3,
                      columns: '2 220px',
                      columnGap: 2.5,
                      fontSize: 13,
                      color: 'text.secondary',
                      '& li': { breakInside: 'avoid', py: '1px' },
                    }}
                  >
                    {list.map((check) => (
                      <li key={check.name}>{check.summary || check.name}</li>
                    ))}
                  </Box>
                </AccordionDetails>
              </Accordion>
            );
          })}
        </Section>

        <Section
          title={Messages.pane.recentRuns}
          testId="schedule-pane-recent-runs"
        >
          {recentRuns?.results.length ? (
            <Table size="small">
              <TableBody>
                {recentRuns.results.map((run) => (
                  <TableRow key={run.id}>
                    <TableCell sx={{ fontFamily: MONO_FONT }}>
                      {formatUtc(new Date(run.startedAt))}
                    </TableCell>
                    <TableCell>
                      <Stack direction="row" alignItems="center" gap={0.75}>
                        <StatusDot
                          color={RUN_STATUS_COLOR[run.status] ?? 'primary.main'}
                        />
                        {RUN_STATUS_LABEL[run.status]}
                      </Stack>
                    </TableCell>
                    <TableCell align="right">
                      {Messages.last.findings(run.findingsCount ?? 0)}
                    </TableCell>
                    <TableCell>
                      <Link
                        component="button"
                        variant="body2"
                        onClick={() => onOpenRun(run.id)}
                      >
                        {Messages.pane.open}
                      </Link>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <Note>{Messages.pane.notRunYet}</Note>
          )}
        </Section>
      </Stack>
    );
  };

  return (
    <Drawer
      anchor="right"
      open={!!schedule}
      onClose={onClose}
      slotProps={{
        paper: {
          // @ts-expect-error data-testid is passed through to the DOM
          'data-testid': 'schedule-details-pane',
          sx: {
            width: { xs: '100%', sm: 620 },
            display: 'flex',
            flexDirection: 'column',
          },
        },
      }}
    >
      {schedule && (
        <>
          <Stack
            direction="row"
            alignItems="flex-start"
            gap={1.25}
            sx={{
              px: 2.5,
              pt: 2.25,
              pb: 1.5,
              borderBottom: 1,
              borderColor: 'divider',
            }}
          >
            <Stack
              direction="row"
              alignItems="center"
              gap={1.25}
              sx={{ flex: 1 }}
            >
              <Swatch color={color} paused={!schedule.enabled} />
              <Typography variant="h6" component="h2">
                {schedule.name}
              </Typography>
              {!schedule.enabled && (
                <Chip
                  size="small"
                  variant="outlined"
                  label={Messages.pane.paused}
                  sx={{ borderStyle: 'dashed' }}
                />
              )}
            </Stack>
            <IconButton
              size="small"
              aria-label={Messages.pane.close}
              onClick={onClose}
              data-testid="schedule-pane-close"
            >
              <CloseIcon fontSize="small" />
            </IconButton>
          </Stack>
          {renderBody(schedule)}
        </>
      )}
    </Drawer>
  );
};
