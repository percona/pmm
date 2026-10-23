import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import * as advisorsApi from 'api/advisors';
import * as settingsApi from 'api/settings';
import type { Settings } from 'types/settings.types';
import { Severity } from 'types/severity.types';
import { AdvisorsForm } from './AdvisorsForm';

vi.mock('api/advisors');
vi.mock('api/settings');

const sendTestMock = vi.mocked(advisorsApi.sendTestAdvisorNotification);

const settings = {
  advisorEnabled: true,
  advisorRunIntervals: {
    rareInterval: '259200s',
    standardInterval: '86400s',
    frequentInterval: '14400s',
  },
  advisorHistoryRetention: '2592000s',
  advisorNotificationsEnabled: true,
  advisorNotificationSeverities: [Severity.critical, Severity.warning],
  advisorNotificationEmailAddresses: ['dba@example.com'],
} as Settings;

const severityCheckbox = (label: string) =>
  screen.getByTestId(`advisorSeverities-${label}-checkbox-input`);

const renderForm = () =>
  render(
    <TestWrapper>
      {wrapWithSnackbarProvider(
        wrapWithQueryProvider(<AdvisorsForm settings={settings} />)
      )}
    </TestWrapper>
  );

describe('AdvisorsForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('emails a test report with the recipients and severities currently in the form', async () => {
    sendTestMock.mockResolvedValue(undefined);
    renderForm();

    fireEvent.change(
      screen.getByTestId('advisorNotificationEmails-text-input'),
      { target: { value: 'dba@example.com, oncall@example.com' } }
    );
    fireEvent.click(severityCheckbox('error'));

    await waitFor(() =>
      expect(screen.getByTestId('advisor-test-email-button')).toBeEnabled()
    );
    fireEvent.click(screen.getByTestId('advisor-test-email-button'));

    // the mutation function is also handed a TanStack context argument
    await waitFor(() => expect(sendTestMock).toHaveBeenCalled());
    expect(sendTestMock.mock.calls[0][0]).toEqual({
      emailAddresses: ['dba@example.com', 'oncall@example.com'],
      severities: [Severity.critical, Severity.error, Severity.warning],
    });
  });

  it('does not offer the test until a severity is selected', async () => {
    renderForm();

    fireEvent.click(severityCheckbox('critical'));
    fireEvent.click(severityCheckbox('warning'));

    await waitFor(() =>
      expect(screen.getByTestId('advisor-test-email-button')).toBeDisabled()
    );
    expect(sendTestMock).not.toHaveBeenCalled();
  });

  it('does not offer the test until the recipients are valid', async () => {
    renderForm();

    fireEvent.change(
      screen.getByTestId('advisorNotificationEmails-text-input'),
      { target: { value: '' } }
    );

    await waitFor(() =>
      expect(screen.getByTestId('advisor-test-email-button')).toBeDisabled()
    );
    expect(sendTestMock).not.toHaveBeenCalled();
  });

  it('checks the saved severities', () => {
    renderForm();

    expect(severityCheckbox('critical')).toBeChecked();
    expect(severityCheckbox('error')).not.toBeChecked();
    expect(severityCheckbox('warning')).toBeChecked();
    expect(severityCheckbox('info')).not.toBeChecked();
  });

  it('saves the selected severities, most severe first', async () => {
    renderForm();

    fireEvent.click(severityCheckbox('info'));
    fireEvent.click(severityCheckbox('critical'));
    fireEvent.click(severityCheckbox('error'));

    await waitFor(() =>
      expect(screen.getByTestId('advisors-button')).toBeEnabled()
    );
    fireEvent.click(screen.getByTestId('advisors-button'));

    await waitFor(() => expect(settingsApi.updateSettings).toHaveBeenCalled());
    expect(
      vi.mocked(settingsApi.updateSettings).mock.calls[0][0]
        .advisorNotificationSeverities
    ).toEqual([Severity.error, Severity.warning, Severity.info]);
  });

  it('requires at least one severity while notifications are on', async () => {
    renderForm();

    fireEvent.click(severityCheckbox('critical'));
    fireEvent.click(severityCheckbox('warning'));

    expect(
      await screen.findByText(
        'Select at least one severity, or turn Advisor notifications off'
      )
    ).toBeInTheDocument();
    expect(screen.getByTestId('advisors-button')).toBeDisabled();
  });
});
