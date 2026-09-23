import { QueryClient } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import * as haApi from 'api/ha';
import * as settingsApi from 'api/settings';
import { SETTINGS_MOCK } from 'api/__mocks__/settings';
import type { HAStatus } from 'types/ha.types';
import type { Settings as SettingsType } from 'types/settings.types';
import { AdvancedSettingsForm } from './AdvancedSettingsForm';

vi.mock('api/settings');
vi.mock('api/ha', () => ({
  getHAStatus: vi.fn(),
  getHANodes: vi.fn(),
}));

const getHAStatusMock = vi.mocked(haApi.getHAStatus);
const updateSettingsMock = vi.mocked(settingsApi.updateSettings);

const settings = { dataRetention: '2592000s' } as SettingsType;

// The field is editable until the HA status arrives, so anything asserted while the query is
// still in flight holds whatever the answer turns out to be. Render, then wait for the status to
// land in the cache, so every assertion below is made against the settled form.
const renderWithSettledHA = async (
  status: HAStatus,
  formSettings: SettingsType = settings
) => {
  getHAStatusMock.mockResolvedValue({ status });

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <TestWrapper>
      {wrapWithQueryProvider(
        <AdvancedSettingsForm settings={formSettings} />,
        queryClient
      )}
    </TestWrapper>
  );

  await waitFor(() =>
    expect(queryClient.getQueryData(['ha:status'])).toEqual({ status })
  );
};

const retentionInput = () => screen.getByTestId('retention-number-input');

describe('AdvancedSettingsForm data retention in HA', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('is editable when high availability is disabled', async () => {
    await renderWithSettledHA('Disabled');

    expect(retentionInput()).toBeEnabled();
    expect(screen.queryByText(/dataRetentionDays/)).not.toBeInTheDocument();
  });

  it('is disabled and points at the chart when high availability is enabled', async () => {
    await renderWithSettledHA('Enabled');

    expect(retentionInput()).toBeDisabled();
    expect(screen.getByText(/dataRetentionDays/)).toBeInTheDocument();
  });
});

describe('AdvancedSettingsForm submit with data retention in HA', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // Dirties a field other than retention and submits, the way a user saving any other setting
  // on the page would.
  const submitAnotherSetting = async () => {
    fireEvent.change(screen.getByTestId('publicAddress-text-input'), {
      target: { value: '1.2.3.4' },
    });
    const submit = screen.getByTestId('advanced-button');
    await waitFor(() => expect(submit).toBeEnabled());
    fireEvent.click(submit);
  };

  it('sends the retention when high availability is disabled', async () => {
    await renderWithSettledHA('Disabled', SETTINGS_MOCK);

    await submitAnotherSetting();

    await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
    expect(updateSettingsMock.mock.calls[0][0]).toHaveProperty(
      'dataRetention',
      '2592000s'
    );
  });

  // The server refuses any retention that differs from the stored one in HA, and the user
  // cannot correct a disabled field, so a stale form value must never be sent.
  it('leaves the retention out when high availability is enabled', async () => {
    await renderWithSettledHA('Enabled', SETTINGS_MOCK);

    await submitAnotherSetting();

    await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
    expect(updateSettingsMock.mock.calls[0][0]).not.toHaveProperty(
      'dataRetention'
    );
  });

  // The pmm-ha chart accepts up to 36500 days, beyond the range the form enforces.
  it('does not let a locked retention outside the form range block saving', async () => {
    await renderWithSettledHA('Enabled', {
      ...SETTINGS_MOCK,
      dataRetention: `${4000 * 24 * 60 * 60}s`,
    });

    expect(retentionInput()).toHaveValue(4000);
    expect(
      screen.queryByTestId('retention-field-error-message')
    ).not.toBeInTheDocument();

    await submitAnotherSetting();

    await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
    expect(updateSettingsMock.mock.calls[0][0]).not.toHaveProperty(
      'dataRetention'
    );
  });
});
