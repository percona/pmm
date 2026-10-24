import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import * as advisorsApi from 'api/advisors';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  type AdvisorCheck,
  AdvisorCheckTriggeredBy,
  type AdvisorRun,
  AdvisorRunStatus,
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  AdvisorTechnology,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSettings,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import AdvisorScheduler from './AdvisorScheduler';
import { DEFAULT_SCHEDULE_INPUT } from './AdvisorScheduler.utils';

vi.mock('api/advisors');

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => ({
  ...(await vi.importActual('react-router-dom')),
  useNavigate: () => mockNavigate,
}));

const RUN: AdvisorRun = {
  id: 'run-1',
  triggeredBy: AdvisorCheckTriggeredBy.scheduler,
  status: AdvisorRunStatus.completed,
  startedAt: '2026-10-24T06:00:05Z',
  finishedAt: '2026-10-24T06:02:00Z',
  plannedChecksCount: 2,
  plannedServicesCount: 2,
  checksCount: 2,
  servicesCount: 2,
  findingsCount: 22,
  errorsCount: 0,
  severityCounts: [],
  checkNames: [],
  serviceIds: [],
  scheduleId: 'daily',
  scheduleName: 'All checks daily',
  scheduledFor: '2026-10-24T06:00:00Z',
};

const DAILY: AdvisorSchedule = {
  scheduleId: 'daily',
  name: 'All checks daily',
  enabled: true,
  daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
  timeOfDay: '06:00',
  emailMode: AdvisorScheduleEmailMode.settings,
  lastRun: RUN,
  createdAt: '2026-10-01T00:00:00Z',
  updatedAt: '2026-10-01T00:00:00Z',
};

const PROD: AdvisorSchedule = {
  scheduleId: 'prod',
  name: 'Prod security',
  enabled: true,
  daysOfWeek: [1, 2, 3, 4, 5],
  timeOfDay: '00:30',
  serviceIds: ['removed-service'],
  emailMode: AdvisorScheduleEmailMode.custom,
  emailAddresses: ['secops@example.com', 'dba@example.com'],
  emailSeverities: [Severity.critical, Severity.error],
  createdAt: '2026-10-02T00:00:00Z',
  updatedAt: '2026-10-02T00:00:00Z',
};

// false booleans are omitted from the JSON
const PAUSED: AdvisorSchedule = {
  scheduleId: 'paused',
  name: 'Version audit',
  daysOfWeek: [1],
  timeOfDay: '12:00',
  emailMode: AdvisorScheduleEmailMode.off,
  createdAt: '2026-10-03T00:00:00Z',
  updatedAt: '2026-10-03T00:00:00Z',
};

const CHECKS = [
  {
    name: 'mysql_version',
    summary: 'MySQL version',
    technology: AdvisorTechnology.mysql,
    enabled: true,
    category: 'Versions',
  },
  {
    name: 'mongodb_auth',
    summary: 'MongoDB authentication',
    technology: AdvisorTechnology.mongodb,
    enabled: true,
    category: 'Security',
  },
] as AdvisorCheck[];

const SERVICES = [
  {
    serviceId: 'mysql-1',
    serviceName: 'mysql-prod-01',
    technology: AdvisorTechnology.mysql,
    environment: 'prod',
    cluster: 'orders',
  },
  {
    serviceId: 'pg-1',
    serviceName: 'pg-prod-01',
    technology: AdvisorTechnology.postgresql,
  },
];

const renderPage = (
  initialEntry = '/advisors/scheduler',
  advisorEnabled = true
) =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(
          wrapWithSettings(
            wrapWithRouter(<AdvisorScheduler />, {
              initialEntries: [initialEntry],
            }),
            { settings: { advisorEnabled } }
          )
        )
      )
    )
  );

const openMenu = async (scheduleId: string) => {
  fireEvent.click(await screen.findByTestId(`schedule-${scheduleId}-actions`));
};

describe('AdvisorScheduler', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Saturday, 24 Oct 2026; only Date is faked so the queries still settle
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-24T12:00:00Z'));
    vi.mocked(advisorsApi.listAdvisorSchedules).mockResolvedValue([
      DAILY,
      PROD,
      PAUSED,
    ]);
    vi.mocked(advisorsApi.listAdvisorChecks).mockResolvedValue(CHECKS);
    vi.mocked(advisorsApi.listAdvisorServices).mockResolvedValue(SERVICES);
    vi.mocked(advisorsApi.getAdvisorNotificationSettings).mockResolvedValue({
      enabled: true,
      emailAddresses: ['dba@example.com', 'oncall@example.com'],
      severities: [Severity.critical, Severity.error],
    });
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 0,
      totalPages: 0,
      results: [],
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('lists each schedule with its timing, scope, email and runs', async () => {
    renderPage();

    const daily = await screen.findByTestId('schedule-row-daily');
    expect(daily).toHaveTextContent('Every day, 06:00 UTC');
    expect(daily).toHaveTextContent('All checks');
    expect(daily).toHaveTextContent('2 today');
    expect(daily).toHaveTextContent('Settings recipients');
    await waitFor(() => expect(daily).toHaveTextContent('2 addresses'));
    expect(within(daily).getByTestId('schedule-next-run')).toHaveTextContent(
      'Sun, 25 Oct, 06:00 UTCin 18 h'
    );
    expect(within(daily).getByTestId('schedule-last-run')).toHaveTextContent(
      '22 findings'
    );

    const prod = screen.getByTestId('schedule-row-prod');
    expect(prod).toHaveTextContent('Mon–Fri, 00:30 UTC');
    expect(prod).toHaveTextContent('secops@example.com +1');
    expect(prod).toHaveTextContent('Own recipients · Critical, Error');

    const paused = screen.getByTestId('schedule-row-paused');
    expect(paused).toHaveTextContent('No email');
    expect(paused).toHaveTextContent('Paused');
    expect(paused).toHaveTextContent('Never run');
  });

  it('flags a schedule whose services were all removed', async () => {
    renderPage();

    expect(
      await screen.findByTestId('schedule-prod-targets-nothing')
    ).toHaveTextContent('Targets no existing service');
    expect(
      screen.queryByTestId('schedule-daily-targets-nothing')
    ).not.toBeInTheDocument();
  });

  it('pauses a schedule by replacing it with enabled off', async () => {
    vi.mocked(advisorsApi.updateAdvisorSchedule).mockResolvedValue({
      ...DAILY,
      enabled: false,
    });
    renderPage();

    fireEvent.click(
      within(await screen.findByTestId('schedule-daily-enabled')).getByRole(
        'switch'
      )
    );

    await waitFor(() =>
      expect(advisorsApi.updateAdvisorSchedule).toHaveBeenCalledWith('daily', {
        name: 'All checks daily',
        enabled: false,
        daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
        timeOfDay: '06:00',
        checkNames: [],
        serviceIds: [],
        emailMode: AdvisorScheduleEmailMode.settings,
        emailAddresses: [],
        emailSeverities: [],
      })
    );
    expect(
      await screen.findByText(
        `Paused "All checks daily". It won't run until you enable it.`
      )
    ).toBeInTheDocument();
    // the switch does not open the details pane
    expect(
      screen.queryByTestId('schedule-details-pane')
    ).not.toBeInTheDocument();
  });

  it('duplicates a schedule as a paused copy', async () => {
    vi.mocked(advisorsApi.createAdvisorSchedule).mockImplementation(
      async (input) => ({ ...PROD, ...input, scheduleId: 'copy' })
    );
    renderPage();

    await openMenu('prod');
    fireEvent.click(await screen.findByTestId('schedule-action-duplicate'));

    await waitFor(() =>
      expect(advisorsApi.createAdvisorSchedule).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'Prod security (copy)',
          enabled: false,
          serviceIds: ['removed-service'],
          emailAddresses: ['secops@example.com', 'dba@example.com'],
        })
      )
    );
    expect(
      await screen.findByText(
        'Duplicated as "Prod security (copy)". It stays paused until you enable it, so nothing runs twice.'
      )
    ).toBeInTheDocument();
  });

  it('deletes a schedule after confirming', async () => {
    vi.mocked(advisorsApi.deleteAdvisorSchedule).mockResolvedValue();
    renderPage();

    await openMenu('daily');
    fireEvent.click(await screen.findByTestId('schedule-action-delete'));

    const dialog = await screen.findByTestId('schedule-delete-dialog');
    expect(dialog).toHaveTextContent('Delete "All checks daily"?');
    expect(dialog).toHaveTextContent(
      'Its future runs are cancelled. Past runs stay in Run history under this name.'
    );
    fireEvent.click(within(dialog).getByTestId('schedule-delete-confirm'));

    await waitFor(() =>
      expect(advisorsApi.deleteAdvisorSchedule).toHaveBeenCalledWith(
        'daily',
        expect.anything()
      )
    );
    expect(
      await screen.findByText('Deleted "All checks daily".')
    ).toBeInTheDocument();
  });

  it('warns before deleting the last schedule', async () => {
    vi.mocked(advisorsApi.listAdvisorSchedules).mockResolvedValue([DAILY]);
    renderPage();

    await openMenu('daily');
    fireEvent.click(await screen.findByTestId('schedule-action-delete'));

    expect(
      await screen.findByTestId('schedule-delete-dialog')
    ).toHaveTextContent(
      'This is the last schedule. Without it, Advisor checks run only when someone starts them.'
    );
  });

  it('runs a schedule now', async () => {
    vi.mocked(advisorsApi.runAdvisorSchedule).mockResolvedValue('run-2');
    renderPage();

    await openMenu('daily');
    fireEvent.click(await screen.findByTestId('schedule-action-run'));

    await waitFor(() =>
      expect(advisorsApi.runAdvisorSchedule).toHaveBeenCalledWith(
        'daily',
        expect.anything()
      )
    );
    expect(
      await screen.findByText(
        'Started "All checks daily". Follow it in Run history.'
      )
    ).toBeInTheDocument();
  });

  it('shows the run in progress and blocks Run now until it ends', async () => {
    vi.mocked(advisorsApi.listRuns).mockResolvedValue({
      totalItems: 1,
      totalPages: 1,
      results: [
        {
          ...RUN,
          status: AdvisorRunStatus.running,
          finishedAt: null,
          checksCount: 1,
          startedAt: '2026-10-24T11:58:00Z',
        },
      ],
    });
    renderPage();

    expect(await screen.findByTestId('schedules-run-active')).toHaveTextContent(
      'All checks daily is running: 1 of 2 checks done, started 2 min ago.'
    );
    await openMenu('daily');
    expect(await screen.findByTestId('schedule-action-run')).toHaveAttribute(
      'aria-disabled',
      'true'
    );
  });

  it('offers the default schedule when there is none', async () => {
    vi.mocked(advisorsApi.listAdvisorSchedules).mockResolvedValue([]);
    vi.mocked(advisorsApi.createAdvisorSchedule).mockResolvedValue(DAILY);
    renderPage();

    expect(await screen.findByTestId('schedules-empty')).toHaveTextContent(
      'No schedules'
    );
    expect(screen.getByTestId('schedule-week-empty')).toHaveTextContent(
      'Nothing is scheduled.'
    );
    fireEvent.click(screen.getByTestId('schedules-restore-default'));

    await waitFor(() =>
      expect(advisorsApi.createAdvisorSchedule).toHaveBeenCalledWith(
        DEFAULT_SCHEDULE_INPUT
      )
    );
    expect(
      await screen.findByText(
        'Created "All checks daily". Next run: Sun, 25 Oct, 06:00 UTC.'
      )
    ).toBeInTheDocument();
  });

  it('says nothing runs while Advisors are off', async () => {
    renderPage('/advisors/scheduler', false);

    expect(
      await screen.findByTestId('schedules-advisors-off')
    ).toBeInTheDocument();
    const daily = await screen.findByTestId('schedule-row-daily');
    expect(daily).toHaveTextContent('Not scheduled');
    expect(daily).toHaveTextContent('Advisors are off');
    expect(screen.getByTestId('schedule-week-empty')).toHaveTextContent(
      'Nothing runs while Advisors are turned off.'
    );
  });

  it('flags email schedules while notifications are off', async () => {
    vi.mocked(advisorsApi.getAdvisorNotificationSettings).mockResolvedValue({});
    renderPage();

    expect(
      await screen.findByTestId('schedules-notifications-off')
    ).toBeInTheDocument();
    const daily = await screen.findByTestId('schedule-row-daily');
    expect(
      within(daily).getByTestId('schedule-email-off-flag')
    ).toBeInTheDocument();
    expect(
      within(screen.getByTestId('schedule-row-paused')).queryByTestId(
        'schedule-email-off-flag'
      )
    ).not.toBeInTheDocument();
  });

  it('opens the details from a row, with what each run covers', async () => {
    renderPage();

    fireEvent.dblClick(await screen.findByTestId('schedule-row-daily'));

    const pane = await screen.findByTestId('schedule-details-pane');
    expect(within(pane).getByTestId('schedule-pane-summary')).toHaveTextContent(
      'Settings recipients: dba@example.com, oncall@example.com'
    );
    expect(within(pane).getByTestId('schedule-pane-summary')).toHaveTextContent(
      'Critical and Error findings, as in Settings'
    );
    const coverage = within(pane).getByTestId('schedule-pane-coverage');
    expect(coverage).toHaveTextContent('mysql-prod-01');
    expect(coverage).toHaveTextContent('1 check on 1 service');
    expect(coverage).toHaveTextContent(
      'No selected check applies to pg-prod-01.'
    );
    expect(
      within(pane).getByTestId('schedule-pane-next-runs')
    ).toHaveTextContent('Sun, 25 Oct');
  });

  it('opens the details from a link', async () => {
    renderPage('/advisors/scheduler?details=paused');

    const pane = await screen.findByTestId('schedule-details-pane');
    expect(pane).toHaveTextContent('Version audit');
    expect(
      within(pane).getByTestId('schedule-pane-next-runs')
    ).toHaveTextContent('Paused. Enable the schedule to plan its runs.');
    expect(
      within(pane).getByTestId('schedule-pane-recent-runs')
    ).toHaveTextContent('This schedule has not run yet.');
  });

  it('opens Run history filtered to the schedule', async () => {
    renderPage();

    await openMenu('prod');
    fireEvent.click(await screen.findByTestId('schedule-action-runs'));

    expect(mockNavigate).toHaveBeenCalledWith('/advisors/runs?scheduleId=prod');
  });
});
