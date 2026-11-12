import CloseIcon from '@mui/icons-material/Close';
import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Checkbox from '@mui/material/Checkbox';
import Chip from '@mui/material/Chip';
import Drawer from '@mui/material/Drawer';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import FormHelperText from '@mui/material/FormHelperText';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Stack from '@mui/material/Stack';
import { alpha } from '@mui/material/styles';
import Switch from '@mui/material/Switch';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableFooter from '@mui/material/TableFooter';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { useUser } from 'contexts/user';
import {
  useSaveAdvisorSchedule,
  useSendTestAdvisorNotification,
} from 'hooks/api/useAdvisors';
import { ADVISOR_TECHNOLOGY, SEVERITY } from 'lib/constants';
import { enqueueSnackbar } from 'notistack';
import { ADVISOR_SEVERITY_OPTIONS } from 'pages/settings/components/advisors/Advisors.constants';
import {
  type FC,
  type FormEvent,
  type ReactNode,
  useMemo,
  useState,
} from 'react';
import { Link as RouterLink } from 'react-router-dom';
import {
  type AdvisorCheck,
  type AdvisorNotificationSettings,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  type AdvisorTargetService,
} from 'types/advisors.types';
import { MONO_FONT, TechnologyLabel } from '../AdvisorScheduler.components';
import {
  daysPhrase,
  EVERY_DAY,
  formatLocalDay,
  formatLocalTime,
  formatUtc,
  isUtcViewer,
  joinAnd,
  localDayShift,
  MAX_SCHEDULE_NAME_LENGTH,
  nextRuns,
  observesDaylightSaving,
  offsetLabel,
  scopeChecks,
  scopeServices,
  technologyCoverage,
  timeZoneLabel,
  WEEK_ORDER,
  WEEKDAYS,
  WEEKDAYS_LONG,
  WEEKDAYS_SHORT,
  WEEKEND,
} from '../AdvisorScheduler.utils';
import { Messages } from './ScheduleEditor.messages';
import {
  emptyScheduleForm,
  FORM_FIELDS,
  orderSeverities,
  parseEmails,
  type ScheduleForm,
  type ScheduleFormField,
  testEmailTarget,
  toScheduleForm,
  toScheduleInputFromForm,
  validateScheduleForm,
} from './ScheduleEditor.utils';
import { type PickerItem, ScopePicker } from './ScopePicker';

// element each field's error moves the focus to on submit
const FOCUS_TARGET: Record<ScheduleFormField, string> = {
  name: 'schedule-name',
  days: 'schedule-day-1',
  time: 'schedule-time',
  checks: 'schedule-checks-selected',
  services: 'schedule-services-selected',
  email: 'schedule-email-addresses',
  severities: 'schedule-severities-custom',
};

const DAY_PRESETS = [
  { label: Messages.when.everyDay, days: EVERY_DAY },
  { label: Messages.when.weekdays, days: WEEKDAYS },
  { label: Messages.when.weekends, days: WEEKEND },
];

const Block: FC<{ title?: string; children: ReactNode }> = ({
  title,
  children,
}) => (
  <Stack
    component="section"
    gap={1.25}
    sx={{ pb: 2.5, borderBottom: 1, borderColor: 'divider' }}
  >
    {title && (
      <Typography variant="subtitle1" component="h3" fontWeight={600}>
        {title}
      </Typography>
    )}
    {children}
  </Stack>
);

// a radio option with a title and a description under it
const Option: FC<{
  value: string;
  title: string;
  hint?: ReactNode;
  id?: string;
}> = ({ value, title, hint, id }) => (
  <FormControlLabel
    value={value}
    sx={{ alignItems: 'flex-start', mr: 0, '& .MuiRadio-root': { pt: 0.25 } }}
    control={<Radio size="small" id={id} />}
    label={
      <Stack>
        <Typography variant="body2" fontWeight={500}>
          {title}
        </Typography>
        {hint && (
          <Typography variant="body2" color="text.secondary" component="div">
            {hint}
          </Typography>
        )}
      </Stack>
    }
  />
);

interface ScheduleEditorProps {
  open: boolean;
  // the schedule to edit; a new one is created when unset
  schedule?: AdvisorSchedule | null;
  schedules: AdvisorSchedule[];
  checks?: AdvisorCheck[];
  services?: AdvisorTargetService[];
  notification?: AdvisorNotificationSettings;
  onClose: () => void;
}

export const ScheduleEditor: FC<ScheduleEditorProps> = ({
  open,
  schedule,
  schedules,
  checks,
  services,
  notification,
  onClose,
}) => {
  const { user } = useUser();
  const [form, setForm] = useState<ScheduleForm>(emptyScheduleForm);
  const [submitted, setSubmitted] = useState(false);
  // the schedule the form was last loaded for; reloading only when it changes
  // keeps edits when the page refetches the list
  const [loadedFor, setLoadedFor] = useState<string | null>(null);
  const loadKey = open ? (schedule?.scheduleId ?? '') : null;
  if (loadKey !== loadedFor) {
    setLoadedFor(loadKey);
    if (loadKey !== null) {
      setForm(
        schedule
          ? toScheduleForm(schedule, checks, services)
          : emptyScheduleForm()
      );
      setSubmitted(false);
    }
  }

  const { mutate: save, isPending: isSaving } = useSaveAdvisorSchedule();
  const { mutate: sendTest, isPending: isSendingTest } =
    useSendTestAdvisorNotification();

  const update = (changes: Partial<ScheduleForm>) =>
    setForm((current) => ({ ...current, ...changes }));

  const takenNames = schedules
    .filter((s) => s.scheduleId !== schedule?.scheduleId)
    .map((s) => s.name);
  const errors = validateScheduleForm(form, takenNames);
  const failedFields = FORM_FIELDS.filter((field) => errors[field]);
  const shownError = (field: ScheduleFormField) =>
    submitted ? errors[field] : undefined;

  const allChecks = useMemo(() => checks ?? [], [checks]);
  const allServices = useMemo(() => services ?? [], [services]);

  const checkItems = useMemo<PickerItem[]>(
    () =>
      allChecks.map((check) => ({
        id: check.name,
        label: check.summary || check.name,
        technology: check.technology,
        group: check.category,
        meta: [check.category],
        search: `${check.summary} ${check.name}`.toLowerCase(),
        warning: check.enabled
          ? undefined
          : {
              label: Messages.checks.disabledInCatalog,
              tooltip: Messages.checks.disabledTooltip,
            },
      })),
    [allChecks]
  );
  const serviceItems = useMemo<PickerItem[]>(
    () =>
      allServices.map((service) => ({
        id: service.serviceId,
        label: service.serviceName,
        technology: service.technology,
        group: service.environment ?? '',
        meta: [
          service.environment ?? '',
          service.cluster ?? '',
          service.nodeName ?? '',
        ],
        search:
          `${service.serviceName} ${service.cluster ?? ''} ${service.nodeName ?? ''}`.toLowerCase(),
      })),
    [allServices]
  );

  const coverageChecks = scopeChecks(
    allChecks,
    form.checksMode === 'selected' ? form.checks : undefined
  ).filter((c) => c.enabled);
  const coverageServices = scopeServices(
    allServices,
    form.servicesMode === 'selected' ? form.services : undefined
  );
  const coverage = technologyCoverage(coverageChecks, coverageServices);
  const total = coverage.reduce((sum, r) => sum + r.executions, 0);
  const idle = coverage.filter((r) => r.checks && !r.services);
  const unchecked = coverage.filter((r) => r.services && !r.checks);

  const testTarget = testEmailTarget(form, notification);
  const settingsSeverities = joinAnd(
    (notification?.severities ?? []).map((s) => SEVERITY[s])
  );

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (failedFields.length) {
      const target = document.getElementById(FOCUS_TARGET[failedFields[0]]);
      target?.focus();
      target?.scrollIntoView?.({ block: 'center', behavior: 'smooth' });
      return;
    }
    const input = toScheduleInputFromForm(form);
    save(
      { scheduleId: schedule?.scheduleId, input },
      {
        onSuccess: (saved) => {
          const next = saved.nextRunAt ? new Date(saved.nextRunAt) : null;
          enqueueSnackbar(
            [
              schedule
                ? Messages.success.saved(saved.name)
                : Messages.success.created(saved.name),
              input.enabled
                ? next && Messages.success.next(formatUtc(next))
                : Messages.success.paused,
            ]
              .filter(Boolean)
              .join(' '),
            { variant: 'success' }
          );
          onClose();
        },
      }
    );
  };

  const onSendTest = () => {
    if (!testTarget) {
      return;
    }
    sendTest(
      {
        emailAddresses: testTarget.to,
        severities: testTarget.severities,
      },
      {
        onSuccess: () =>
          enqueueSnackbar(
            Messages.email.testSent(
              testTarget.to.join(', '),
              joinAnd(testTarget.severities.map((s) => SEVERITY[s]))
            ),
            { variant: 'success' }
          ),
      }
    );
  };

  const renderPreview = () => {
    const runs = nextRuns(form.days, form.time, new Date(), 3);
    if (!runs.length) {
      return (
        <Typography variant="body2">{Messages.when.previewEmpty}</Typography>
      );
    }
    const first = runs[0];
    const local = !isUtcViewer(first);
    const zone = timeZoneLabel();
    const shift = localDayShift(first);
    const notes = [
      local && shift ? Messages.when.shift(shift > 0) : '',
      local && observesDaylightSaving(first.getUTCFullYear())
        ? Messages.when.dst(zone)
        : '',
    ].filter(Boolean);
    return (
      <>
        <Typography variant="body1" fontWeight={500}>
          {Messages.when.runs(daysPhrase(form.days), form.time)}
          {local &&
            Messages.when.local(
              formatLocalTime(first),
              zone,
              offsetLabel(first)
            )}
          .
        </Typography>
        <Box
          component="ol"
          aria-label={Messages.when.nextRuns}
          sx={{
            m: 0,
            p: 0,
            listStyle: 'none',
            display: 'flex',
            flexDirection: 'column',
            gap: 0.5,
            fontSize: 13,
            color: 'text.secondary',
            fontVariantNumeric: 'tabular-nums',
          }}
          data-testid="schedule-preview-runs"
        >
          {runs.map((at, i) => (
            <li key={at.getTime()}>
              {i === 0 && (
                <Box
                  component="span"
                  sx={{
                    mr: 1,
                    px: 0.75,
                    py: '1px',
                    borderRadius: '10px',
                    bgcolor: 'primary.main',
                    color: 'primary.contrastText',
                    fontSize: 11,
                    fontWeight: 600,
                  }}
                  data-testid="schedule-preview-next"
                >
                  {Messages.when.next}
                </Box>
              )}
              <Box
                component="span"
                sx={{
                  fontFamily: MONO_FONT,
                  fontSize: 12.5,
                  color: 'text.primary',
                }}
              >
                {formatUtc(at)}
              </Box>
              {local &&
                Messages.when.yourTime(
                  `${formatLocalDay(at)}, ${formatLocalTime(at)}`
                )}
            </li>
          ))}
        </Box>
        {notes.length > 0 && (
          <Typography variant="caption" color="text.secondary">
            {notes.join(' ')}
          </Typography>
        )}
      </>
    );
  };

  const renderCoverageNotes = () => {
    if (!total && coverageChecks.length && coverageServices.length) {
      const techs = [
        ...new Set(
          coverageServices.map((s) => ADVISOR_TECHNOLOGY[s.technology])
        ),
      ].join(', ');
      return (
        <Alert severity="warning" data-testid="schedule-coverage-nothing">
          {Messages.coverage.nothing(techs || Messages.coverage.yourServices)}
        </Alert>
      );
    }
    const all = form.servicesMode === 'all';
    return (
      <>
        {idle.length > 0 && (
          <Alert severity="info" data-testid="schedule-coverage-idle">
            {Messages.coverage.idle(
              idle
                .map((r) =>
                  Messages.coverage.checkCount(
                    r.checks,
                    ADVISOR_TECHNOLOGY[r.technology]
                  )
                )
                .join(' and '),
              idle.map((r) => ADVISOR_TECHNOLOGY[r.technology]).join(' or '),
              all
            )}
          </Alert>
        )}
        {unchecked.length > 0 && (
          <Alert severity="info" data-testid="schedule-coverage-unchecked">
            {Messages.coverage.unchecked(
              unchecked
                .map((r) =>
                  Messages.coverage.serviceCount(
                    r.services,
                    ADVISOR_TECHNOLOGY[r.technology]
                  )
                )
                .join(' and '),
              unchecked
                .map((r) => ADVISOR_TECHNOLOGY[r.technology])
                .join(' or '),
              unchecked.length === 1 && unchecked[0].services === 1
            )}
          </Alert>
        )}
      </>
    );
  };

  const emailCount = parseEmails(form.emailText).length;
  const disabledChecks = allChecks.filter((c) => !c.enabled).length;

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          // @ts-expect-error data-testid is passed through to the DOM
          'data-testid': 'schedule-editor',
          sx: {
            width: { xs: '100%', sm: 720 },
            display: 'flex',
            flexDirection: 'column',
          },
        },
      }}
    >
      <Stack
        direction="row"
        alignItems="center"
        justifyContent="space-between"
        sx={{
          px: 2.5,
          pt: 2,
          pb: 1.5,
          borderBottom: 1,
          borderColor: 'divider',
        }}
      >
        <Typography variant="h6" component="h2">
          {schedule ? Messages.editTitle : Messages.createTitle}
        </Typography>
        <IconButton
          size="small"
          aria-label={Messages.close}
          onClick={onClose}
          data-testid="schedule-editor-close"
        >
          <CloseIcon fontSize="small" />
        </IconButton>
      </Stack>

      <Stack
        component="form"
        id="schedule-editor-form"
        noValidate
        onSubmit={onSubmit}
        gap={2.5}
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          px: 2.5,
          pt: 2.5,
          pb: 3,
        }}
      >
        <Block>
          <Stack
            direction="row"
            gap={2}
            flexWrap="wrap"
            alignItems="flex-start"
          >
            <TextField
              id="schedule-name"
              label={Messages.name.label}
              value={form.name}
              onChange={(e) => update({ name: e.target.value })}
              placeholder={Messages.name.placeholder}
              error={!!shownError('name')}
              helperText={shownError('name') ?? Messages.name.help}
              autoComplete="off"
              slotProps={{
                htmlInput: {
                  maxLength: MAX_SCHEDULE_NAME_LENGTH,
                  'data-testid': 'schedule-name',
                },
              }}
              sx={{ flex: '1 1 280px' }}
            />
            <FormControlLabel
              sx={{ alignItems: 'flex-start', mt: 1 }}
              control={
                <Switch
                  checked={form.enabled}
                  onChange={(e) => update({ enabled: e.target.checked })}
                  data-testid="schedule-enabled"
                />
              }
              label={
                <Stack>
                  <Typography variant="body2">
                    {Messages.enabled.label}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {Messages.enabled.help}
                  </Typography>
                </Stack>
              }
            />
          </Stack>
        </Block>

        <Block title={Messages.when.title}>
          <Stack gap={1} role="group" aria-labelledby="schedule-days-label">
            <Typography
              id="schedule-days-label"
              variant="body2"
              color="text.secondary"
            >
              {Messages.when.days}
            </Typography>
            <Stack direction="row" flexWrap="wrap" gap={0.75}>
              {WEEK_ORDER.map((day) => {
                const on = form.days.includes(day);
                return (
                  <Button
                    key={day}
                    id={`schedule-day-${day}`}
                    variant={on ? 'contained' : 'outlined'}
                    color={on ? 'primary' : 'inherit'}
                    aria-pressed={on}
                    aria-label={WEEKDAYS_LONG[day]}
                    onClick={() =>
                      update({
                        days: on
                          ? form.days.filter((d) => d !== day)
                          : [...form.days, day],
                      })
                    }
                    sx={{
                      minWidth: 52,
                      height: 36,
                      px: 1.5,
                      borderRadius: '18px',
                      boxShadow: 'none',
                      borderColor: on ? undefined : 'divider',
                    }}
                    data-testid={`schedule-day-${day}`}
                  >
                    {WEEKDAYS_SHORT[day]}
                  </Button>
                );
              })}
            </Stack>
            <Stack
              direction="row"
              gap={0.75}
              alignItems="center"
              sx={{ fontSize: 13 }}
            >
              <Typography variant="body2" color="text.secondary">
                {Messages.when.setTo}
              </Typography>
              {DAY_PRESETS.map(({ label, days }, i) => (
                <Stack direction="row" gap={0.75} key={label}>
                  {i > 0 && <span>·</span>}
                  <Link
                    component="button"
                    type="button"
                    variant="body2"
                    onClick={() => update({ days })}
                    data-testid={`schedule-days-preset-${i}`}
                  >
                    {label}
                  </Link>
                </Stack>
              ))}
            </Stack>
            {shownError('days') && (
              <FormHelperText error>{shownError('days')}</FormHelperText>
            )}
          </Stack>
          <TextField
            id="schedule-time"
            type="time"
            label={Messages.when.time}
            value={form.time}
            onChange={(e) => update({ time: e.target.value })}
            error={!!shownError('time')}
            helperText={shownError('time') ?? Messages.when.timeHelp}
            slotProps={{
              inputLabel: { shrink: true },
              htmlInput: { step: 60, 'data-testid': 'schedule-time' },
            }}
            sx={{ width: 200 }}
          />
          <Stack
            gap={0.75}
            aria-live="polite"
            sx={{
              px: 1.75,
              py: 1.5,
              borderRadius: 2,
              bgcolor: (theme) => alpha(theme.palette.primary.main, 0.1),
            }}
            data-testid="schedule-preview"
          >
            {renderPreview()}
          </Stack>
        </Block>

        <Block title={Messages.checks.title}>
          <RadioGroup
            value={form.checksMode}
            onChange={(e) =>
              update({
                checksMode: e.target.value as ScheduleForm['checksMode'],
              })
            }
            aria-label={Messages.checks.title}
          >
            <Option
              value="all"
              title={Messages.checks.all}
              hint={
                checks &&
                Messages.checks.allHint(allChecks.length, disabledChecks)
              }
            />
            <Option
              value="selected"
              id="schedule-checks-selected"
              title={Messages.checks.selected}
              hint={Messages.checks.selectedHint}
            />
          </RadioGroup>
          {form.checksMode === 'selected' && (
            <>
              <ScopePicker
                items={checkItems}
                selected={form.checks}
                onChange={(selected) => update({ checks: selected })}
                searchLabel={Messages.checks.search}
                groupLabel={Messages.checks.group}
                allGroupsLabel={Messages.checks.allGroups}
                noMatch={Messages.checks.noMatch}
                canShowSelectedOnly
                testId="schedule-checks-picker"
              />
              {shownError('checks') && (
                <FormHelperText error>{shownError('checks')}</FormHelperText>
              )}
            </>
          )}
        </Block>

        <Block title={Messages.services.title}>
          <RadioGroup
            value={form.servicesMode}
            onChange={(e) =>
              update({
                servicesMode: e.target.value as ScheduleForm['servicesMode'],
              })
            }
            aria-label={Messages.services.title}
          >
            <Option
              value="all"
              title={Messages.services.all}
              hint={services && Messages.services.allHint(allServices.length)}
            />
            <Option
              value="selected"
              id="schedule-services-selected"
              title={Messages.services.selected}
              hint={Messages.services.selectedHint}
            />
          </RadioGroup>
          {form.servicesMode === 'selected' && (
            <>
              <ScopePicker
                items={serviceItems}
                selected={form.services}
                onChange={(selected) => update({ services: selected })}
                searchLabel={Messages.services.search}
                groupLabel={Messages.services.group}
                allGroupsLabel={Messages.services.allGroups}
                noMatch={Messages.services.noMatch}
                testId="schedule-services-picker"
              />
              {shownError('services') && (
                <FormHelperText error>{shownError('services')}</FormHelperText>
              )}
            </>
          )}
        </Block>

        <Block title={Messages.coverage.title}>
          <Typography variant="body2" color="text.secondary">
            {Messages.coverage.hint}
          </Typography>
          <Table size="small" data-testid="schedule-coverage">
            <TableHead>
              <TableRow>
                <TableCell>{Messages.coverage.technology}</TableCell>
                <TableCell align="right">{Messages.coverage.checks}</TableCell>
                <TableCell align="right">
                  {Messages.coverage.services}
                </TableCell>
                <TableCell align="right">
                  {Messages.coverage.executions}
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {coverage.map((row) => (
                <TableRow key={row.technology}>
                  <TableCell>
                    <TechnologyLabel technology={row.technology} />
                  </TableCell>
                  <TableCell
                    align="right"
                    sx={row.services ? undefined : { color: 'text.disabled' }}
                  >
                    {row.checks}
                  </TableCell>
                  <TableCell
                    align="right"
                    sx={row.checks ? undefined : { color: 'text.disabled' }}
                  >
                    {row.services}
                  </TableCell>
                  <TableCell align="right">{row.executions}</TableCell>
                </TableRow>
              ))}
            </TableBody>
            <TableFooter>
              <TableRow
                sx={{ '& td': { fontWeight: 500, color: 'text.primary' } }}
              >
                <TableCell>{Messages.coverage.eachRun}</TableCell>
                <TableCell align="right">
                  {coverage
                    .filter((r) => r.services)
                    .reduce((sum, r) => sum + r.checks, 0)}
                </TableCell>
                <TableCell align="right">
                  {coverage
                    .filter((r) => r.checks)
                    .reduce((sum, r) => sum + r.services, 0)}
                </TableCell>
                <TableCell align="right" data-testid="schedule-coverage-total">
                  {total}
                </TableCell>
              </TableRow>
            </TableFooter>
          </Table>
          {checks && services && renderCoverageNotes()}
        </Block>

        <Block title={Messages.email.title}>
          {notification && !notification.enabled && (
            <Alert
              severity="warning"
              data-testid="schedule-email-notifications-off"
            >
              {Messages.email.notificationsOff}
            </Alert>
          )}
          <RadioGroup
            value={form.emailMode}
            onChange={(e) => {
              update({ emailMode: e.target.value as AdvisorScheduleEmailMode });
            }}
            aria-label={Messages.email.title}
          >
            <Option
              value={AdvisorScheduleEmailMode.settings}
              title={Messages.email.settings}
              hint={
                <Stack gap={0.75} sx={{ mt: 0.25 }}>
                  {Messages.email.settingsHint}
                  <Stack
                    direction="row"
                    flexWrap="wrap"
                    gap={0.75}
                    alignItems="center"
                  >
                    {notification?.emailAddresses?.length
                      ? notification.emailAddresses.map((address) => (
                          <Chip
                            key={address}
                            size="small"
                            variant="outlined"
                            label={address}
                          />
                        ))
                      : notification && (
                          <span>{Messages.email.noSettingsRecipients}</span>
                        )}
                    {user?.isPMMAdmin && (
                      <Link
                        component={RouterLink}
                        to="/settings/advisors"
                        variant="body2"
                      >
                        {Messages.email.changeInSettings}
                      </Link>
                    )}
                  </Stack>
                </Stack>
              }
            />
            <Option
              value={AdvisorScheduleEmailMode.custom}
              title={Messages.email.custom}
              hint={Messages.email.customHint}
            />
            {form.emailMode === AdvisorScheduleEmailMode.custom && (
              <TextField
                id="schedule-email-addresses"
                label={Messages.email.addresses}
                value={form.emailText}
                onChange={(e) => update({ emailText: e.target.value })}
                placeholder={Messages.email.addressesPlaceholder}
                multiline
                minRows={2}
                error={!!shownError('email')}
                helperText={
                  shownError('email') ??
                  (emailCount
                    ? Messages.email.addressesCount(emailCount)
                    : Messages.email.addressesHelp)
                }
                slotProps={{
                  htmlInput: { 'data-testid': 'schedule-email-addresses' },
                }}
                sx={{ my: 1, ml: 4 }}
              />
            )}
            <Option
              value={AdvisorScheduleEmailMode.off}
              title={Messages.email.off}
              hint={Messages.email.offHint}
            />
          </RadioGroup>

          {form.emailMode !== AdvisorScheduleEmailMode.off && (
            <Stack gap={1} sx={{ pt: 1 }} data-testid="schedule-severities">
              <Typography
                variant="body2"
                fontWeight={600}
                id="schedule-severities-label"
              >
                {Messages.email.severities}
              </Typography>
              <RadioGroup
                value={form.severitiesMode}
                onChange={(e) => {
                  const severitiesMode = e.target
                    .value as ScheduleForm['severitiesMode'];
                  update({
                    severitiesMode,
                    // start from the Settings severities rather than an empty pick
                    severities:
                      severitiesMode === 'custom' && !form.severities.length
                        ? (notification?.severities ?? [])
                        : form.severities,
                  });
                }}
                aria-labelledby="schedule-severities-label"
              >
                <Option
                  value="settings"
                  title={Messages.email.sameAsSettings}
                  hint={Messages.email.sameAsSettingsHint(settingsSeverities)}
                />
                <Option
                  value="custom"
                  id="schedule-severities-custom"
                  title={Messages.email.onlyThese}
                  hint={Messages.email.onlyTheseHint}
                />
              </RadioGroup>
              {form.severitiesMode === 'custom' && (
                <Box sx={{ ml: 4 }}>
                  <FormGroup row aria-label={Messages.email.severitiesGroup}>
                    {ADVISOR_SEVERITY_OPTIONS.map((severity) => (
                      <FormControlLabel
                        key={severity}
                        label={SEVERITY[severity]}
                        control={
                          <Checkbox
                            size="small"
                            checked={form.severities.includes(severity)}
                            onChange={(e) =>
                              update({
                                severities: orderSeverities(
                                  e.target.checked
                                    ? [...form.severities, severity]
                                    : form.severities.filter(
                                        (s) => s !== severity
                                      )
                                ),
                              })
                            }
                            slotProps={{
                              input: {
                                // @ts-expect-error data-testid is passed through to the DOM
                                'data-testid': `schedule-severity-${SEVERITY[severity].toLowerCase()}`,
                              },
                            }}
                          />
                        }
                      />
                    ))}
                  </FormGroup>
                  {shownError('severities') && (
                    <FormHelperText error>
                      {shownError('severities')}
                    </FormHelperText>
                  )}
                </Box>
              )}
              <Stack
                direction="row"
                gap={1.5}
                alignItems="center"
                flexWrap="wrap"
                sx={{ pt: 1 }}
              >
                <span title={testTarget ? '' : Messages.email.testDisabled}>
                  <Button
                    variant="outlined"
                    disabled={!testTarget || isSendingTest}
                    onClick={onSendTest}
                    data-testid="schedule-test-email"
                  >
                    {isSendingTest
                      ? Messages.email.testPending
                      : Messages.email.test}
                  </Button>
                </span>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{ flex: '1 1 260px' }}
                >
                  {Messages.email.testHint}
                </Typography>
              </Stack>
            </Stack>
          )}
        </Block>
      </Stack>

      <Stack
        direction="row"
        alignItems="center"
        gap={1}
        sx={{
          px: 2.5,
          py: 1.5,
          borderTop: 1,
          borderColor: 'divider',
          bgcolor: 'background.default',
        }}
      >
        <Typography
          variant="body2"
          color="error"
          role="alert"
          sx={{ flex: 1 }}
          data-testid="schedule-editor-message"
        >
          {submitted && failedFields.length
            ? Messages.validation.fix([...failedFields])
            : ''}
        </Typography>
        <Button
          color="inherit"
          onClick={onClose}
          data-testid="schedule-editor-cancel"
        >
          {Messages.cancel}
        </Button>
        <Button
          type="submit"
          form="schedule-editor-form"
          variant="contained"
          disabled={isSaving}
          data-testid="schedule-editor-save"
        >
          {schedule ? Messages.save : Messages.create}
        </Button>
      </Stack>
    </Drawer>
  );
};
