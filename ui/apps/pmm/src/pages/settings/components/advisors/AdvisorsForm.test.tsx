import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import * as advisorsApi from 'api/advisors';
import type { Settings } from 'types/settings.types';
import { Severity } from 'types/severity.types';
import { AdvisorsForm } from './AdvisorsForm';

vi.mock('api/advisors');
vi.mock('api/settings');

const sendTestMock = vi.mocked(advisorsApi.sendTestAdvisorNotification);

const settings = {
  advisorEnabled: true,
  advisorRunIntervals: {
    rareInterval: '280800s',
    standardInterval: '86400s',
    frequentInterval: '14400s',
  },
  advisorHistoryRetention: '2592000s',
  advisorNotificationsEnabled: true,
  advisorNotificationSeverityThreshold: Severity.warning,
  advisorNotificationEmailAddresses: ['dba@example.com'],
} as Settings;

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

  it('emails a test report to the addresses currently in the field', async () => {
    sendTestMock.mockResolvedValue(undefined);
    renderForm();

    fireEvent.change(
      screen.getByTestId('advisorNotificationEmails-text-input'),
      { target: { value: 'dba@example.com, oncall@example.com' } }
    );

    await waitFor(() =>
      expect(screen.getByTestId('advisor-test-email-button')).toBeEnabled()
    );
    fireEvent.click(screen.getByTestId('advisor-test-email-button'));

    // the mutation function is also handed a TanStack context argument
    await waitFor(() => expect(sendTestMock).toHaveBeenCalled());
    expect(sendTestMock.mock.calls[0][0]).toEqual({
      emailAddresses: ['dba@example.com', 'oncall@example.com'],
    });
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
});
