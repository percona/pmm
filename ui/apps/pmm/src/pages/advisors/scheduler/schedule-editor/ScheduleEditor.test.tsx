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
  type AdvisorSchedule,
  AdvisorScheduleEmailMode,
  AdvisorTechnology,
} from 'types/advisors.types';
import { Severity } from 'types/severity.types';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithSnackbarProvider,
  wrapWithUserProvider,
} from 'utils/testUtils';
import { ScheduleEditor } from './ScheduleEditor';

vi.mock('api/advisors');

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
    enabled: false,
    category: 'Security',
  },
] as AdvisorCheck[];

const SERVICES = [
  {
    serviceId: 'mysql-1',
    serviceName: 'mysql-prod-01',
    technology: AdvisorTechnology.mysql,
    environment: 'prod',
  },
  {
    serviceId: 'mysql-2',
    serviceName: 'mysql-stg-01',
    technology: AdvisorTechnology.mysql,
    environment: 'staging',
  },
];

const NOTIFICATION = {
  enabled: true,
  emailAddresses: ['dba@example.com'],
  severities: [Severity.critical, Severity.error],
};

const EXISTING: AdvisorSchedule = {
  scheduleId: 'prod',
  name: 'Prod security',
  enabled: true,
  daysOfWeek: [1, 2, 3, 4, 5],
  timeOfDay: '00:30',
  serviceIds: ['mysql-1'],
  emailMode: AdvisorScheduleEmailMode.custom,
  emailAddresses: ['secops@example.com'],
  createdAt: '2026-10-02T00:00:00Z',
  updatedAt: '2026-10-02T00:00:00Z',
};

const onClose = vi.fn();

const renderEditor = (
  schedule?: AdvisorSchedule,
  schedules: AdvisorSchedule[] = [EXISTING]
) =>
  render(
    wrapWithQueryProvider(
      wrapWithSnackbarProvider(
        wrapWithUserProvider(
          wrapWithRouter(
            <ScheduleEditor
              open
              schedule={schedule}
              schedules={schedules}
              checks={CHECKS}
              services={SERVICES}
              notification={NOTIFICATION}
              onClose={onClose}
            />
          )
        )
      )
    )
  );

const save = () => fireEvent.click(screen.getByTestId('schedule-editor-save'));

const typeName = (name: string) =>
  fireEvent.change(screen.getByTestId('schedule-name'), {
    target: { value: name },
  });

describe('ScheduleEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-24T12:00:00Z'));
    vi.mocked(advisorsApi.createAdvisorSchedule).mockImplementation(
      async (input) => ({
        ...EXISTING,
        ...input,
        scheduleId: 'new',
        nextRunAt: '2026-10-26T06:00:00Z',
      })
    );
    vi.mocked(advisorsApi.updateAdvisorSchedule).mockImplementation(
      async (scheduleId, input) => ({ ...EXISTING, ...input, scheduleId })
    );
    vi.mocked(advisorsApi.sendTestAdvisorNotification).mockResolvedValue();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('shows errors only after a submit, and names them in the footer', async () => {
    renderEditor();

    expect(
      screen.getByText('Shown in Run history and in the email subject.')
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole('radio', { name: /^Selected checks/ }));
    save();

    expect(
      await screen.findByText('Give the schedule a name.')
    ).toBeInTheDocument();
    expect(
      screen.getByText('Pick at least one check, or choose All checks.')
    ).toBeInTheDocument();
    expect(screen.getByTestId('schedule-editor-message')).toHaveTextContent(
      'Fix the highlighted fields: name, checks.'
    );
    expect(advisorsApi.createAdvisorSchedule).not.toHaveBeenCalled();

    typeName('Nightly');
    fireEvent.click(screen.getByRole('radio', { name: /^All checks/ }));
    expect(screen.getByTestId('schedule-editor-message')).toHaveTextContent('');
  });

  it('rejects a name another schedule has, ignoring case', async () => {
    renderEditor();

    typeName('prod SECURITY');
    save();

    expect(
      await screen.findByText(
        'A schedule named "prod SECURITY" already exists.'
      )
    ).toBeInTheDocument();
  });

  it('creates a schedule from the form', async () => {
    renderEditor();

    typeName(' Weekday audit ');
    fireEvent.click(screen.getByTestId('schedule-days-preset-1'));
    fireEvent.change(screen.getByTestId('schedule-time'), {
      target: { value: '23:15' },
    });
    expect(screen.getByTestId('schedule-preview')).toHaveTextContent(
      'Runs every weekday at 23:15 UTC'
    );
    fireEvent.click(screen.getByRole('radio', { name: /^These recipients/ }));
    fireEvent.change(screen.getByTestId('schedule-email-addresses'), {
      target: { value: 'secops@example.com, SecOps@example.com' },
    });
    fireEvent.click(screen.getByRole('radio', { name: /^Only these/ }));
    // starts from the Settings severities
    expect(screen.getByTestId('schedule-severity-critical')).toBeChecked();
    fireEvent.click(screen.getByTestId('schedule-severity-error'));
    save();

    await waitFor(() =>
      expect(advisorsApi.createAdvisorSchedule).toHaveBeenCalledWith({
        name: 'Weekday audit',
        enabled: true,
        daysOfWeek: [1, 2, 3, 4, 5],
        timeOfDay: '23:15',
        checkNames: [],
        serviceIds: [],
        emailMode: AdvisorScheduleEmailMode.custom,
        emailAddresses: ['secops@example.com'],
        emailSeverities: [Severity.critical],
      })
    );
    expect(onClose).toHaveBeenCalled();
    expect(
      await screen.findByText(
        'Created "Weekday audit". Next run: Mon, 26 Oct, 06:00 UTC.'
      )
    ).toBeInTheDocument();
  });

  it('edits a schedule by replacing it', async () => {
    renderEditor(EXISTING);

    expect(screen.getByText('Edit schedule')).toBeInTheDocument();
    expect(screen.getByTestId('schedule-name')).toHaveValue('Prod security');
    expect(
      screen.getByTestId('schedule-services-picker-count')
    ).toHaveTextContent('1 of 2 selected');
    fireEvent.click(
      screen.getByTestId('schedule-services-picker-item-mysql-2')
    );
    save();

    await waitFor(() =>
      expect(advisorsApi.updateAdvisorSchedule).toHaveBeenCalledWith(
        'prod',
        expect.objectContaining({
          name: 'Prod security',
          serviceIds: ['mysql-1', 'mysql-2'],
          emailAddresses: ['secops@example.com'],
        })
      )
    );
  });

  it('narrows the service picker by environment and selects what is shown', async () => {
    renderEditor();

    fireEvent.click(screen.getByRole('radio', { name: /^Selected services/ }));
    const picker = screen.getByTestId('schedule-services-picker');
    fireEvent.change(
      within(picker).getByTestId('schedule-services-picker-search'),
      {
        target: { value: 'stg' },
      }
    );
    fireEvent.click(
      within(picker).getByTestId('schedule-services-picker-select-shown')
    );

    expect(
      within(picker).getByTestId('schedule-services-picker-count')
    ).toHaveTextContent('1 of 2 selected');
    expect(screen.getByTestId('schedule-coverage-total')).toHaveTextContent(
      '1'
    );
  });

  it('marks checks disabled in the Catalog and leaves them out of the coverage', () => {
    renderEditor();

    expect(
      screen.getByText(/2 checks today, 1 of them disabled/)
    ).toBeInTheDocument();
    // one enabled MySQL check on two MySQL services
    expect(screen.getByTestId('schedule-coverage-total')).toHaveTextContent(
      '2'
    );
    fireEvent.click(screen.getByRole('radio', { name: /^Selected checks/ }));
    expect(
      within(screen.getByTestId('schedule-checks-picker')).getByText(
        'Disabled in Catalog'
      )
    ).toBeInTheDocument();
  });

  it('sends a test email to the Settings recipients for the Settings severities', async () => {
    renderEditor();

    fireEvent.click(screen.getByTestId('schedule-test-email'));

    await waitFor(() =>
      expect(advisorsApi.sendTestAdvisorNotification).toHaveBeenCalled()
    );
    expect(
      vi.mocked(advisorsApi.sendTestAdvisorNotification).mock.calls[0][0]
    ).toEqual({
      emailAddresses: ['dba@example.com'],
      severities: [Severity.critical, Severity.error],
    });
    expect(
      await screen.findByText(
        'Sent a test report to dba@example.com with sample Critical and Error findings. No checks were run.'
      )
    ).toBeInTheDocument();
  });

  it('cannot send a test email to invalid recipients, and hides severities with no email', () => {
    renderEditor();

    fireEvent.click(screen.getByRole('radio', { name: /^These recipients/ }));
    fireEvent.change(screen.getByTestId('schedule-email-addresses'), {
      target: { value: 'not-an-address' },
    });
    expect(screen.getByTestId('schedule-test-email')).toBeDisabled();

    fireEvent.click(screen.getByRole('radio', { name: /^Don't email/ }));
    expect(screen.queryByTestId('schedule-severities')).not.toBeInTheDocument();
  });
});
