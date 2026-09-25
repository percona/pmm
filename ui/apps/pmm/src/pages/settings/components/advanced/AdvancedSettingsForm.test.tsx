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
import { toFormValues, toPayload } from './AdvancedSettingsForm.utils';

vi.mock('api/settings');
vi.mock('api/ha', () => ({
  getHAStatus: vi.fn(),
  getHANodes: vi.fn(),
}));

const getHAStatusMock = vi.mocked(haApi.getHAStatus);
const updateSettingsMock = vi.mocked(settingsApi.updateSettings);

const settings = { dataRetention: '2592000s' } as SettingsType;

const renderForm = (formSettings: SettingsType = settings) => {
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

  return queryClient;
};

// The field is editable until the HA status arrives, so anything asserted while the query is
// still in flight holds whatever the answer turns out to be. Render, then wait for the status to
// land in the cache, so every assertion below is made against the settled form.
const renderWithSettledHA = async (
  status: HAStatus,
  formSettings: SettingsType = settings
) => {
  getHAStatusMock.mockResolvedValue({ status });

  const queryClient = renderForm(formSettings);

  await waitFor(() =>
    expect(queryClient.getQueryData(['ha:status'])).toEqual({ status })
  );
};

const retentionInput = () => screen.getByTestId('retention-number-input');

// peak-ui's TextInput passes only sx through formHelperTextProps, so a helper-text test ID never
// reaches the DOM; check the input's own state and the message text instead.
const expectNoRetentionError = () => {
  expect(retentionInput()).toHaveAttribute('aria-invalid', 'false');
  expect(
    screen.queryByText(/Value should be in the range/)
  ).not.toBeInTheDocument();
};

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

  const outOfRangeRetention = {
    ...SETTINGS_MOCK,
    dataRetention: `${4000 * 24 * 60 * 60}s`,
  };

  // An unchanged retention is echoed back only at the risk of being stale, so it is left out
  // whatever the HA status is; the server treats a missing retention as "no change".
  it('leaves an unchanged retention out when high availability is disabled', async () => {
    await renderWithSettledHA('Disabled', SETTINGS_MOCK);

    await submitAnotherSetting();

    await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
    expect(updateSettingsMock.mock.calls[0][0]).not.toHaveProperty(
      'dataRetention'
    );
  });

  it('sends the retention once the user changes it', async () => {
    await renderWithSettledHA('Disabled', SETTINGS_MOCK);

    fireEvent.change(retentionInput(), { target: { value: '45' } });
    const submit = screen.getByTestId('advanced-button');
    await waitFor(() => expect(submit).toBeEnabled());
    fireEvent.click(submit);

    await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
    expect(updateSettingsMock.mock.calls[0][0]).toHaveProperty(
      'dataRetention',
      `${45 * 24 * 60 * 60}s`
    );
  });

  it('still rejects a typed retention outside the range', async () => {
    await renderWithSettledHA('Disabled', SETTINGS_MOCK);

    fireEvent.change(retentionInput(), { target: { value: '4000' } });

    await waitFor(() =>
      expect(retentionInput()).toHaveAttribute('aria-invalid', 'true')
    );
    expect(
      screen.getByText(/Value should be in the range/)
    ).toBeInTheDocument();
    expect(screen.getByTestId('advanced-button')).toBeDisabled();
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
    await renderWithSettledHA('Enabled', outOfRangeRetention);

    expect(retentionInput()).toHaveValue(4000);

    await submitAnotherSetting();

    await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
    expect(updateSettingsMock.mock.calls[0][0]).not.toHaveProperty(
      'dataRetention'
    );
    // Only now has every field been validated, so this is when a range error would show.
    expectNoRetentionError();
  });

  // Until /ha/status answers, or if it fails, the form cannot know the field is locked, so
  // neither the payload nor validation may depend on that answer.
  it.each([
    [
      'still loading',
      () => getHAStatusMock.mockReturnValue(new Promise(() => {})),
    ],
    [
      'failed',
      () => getHAStatusMock.mockRejectedValue(new Error('unavailable')),
    ],
  ])(
    'does not send or block on a loaded retention while the HA status is %s',
    async (_, mockStatus) => {
      mockStatus();
      renderForm(outOfRangeRetention);

      await submitAnotherSetting();

      await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
      expect(updateSettingsMock.mock.calls[0][0]).not.toHaveProperty(
        'dataRetention'
      );
      expectNoRetentionError();
    }
  );

  // The field is editable until the HA status answers, so a user can type into it first. Once the
  // lock arrives the field is disabled and cannot be changed back, so the typed value must go,
  // and so must any error it raised, which would otherwise keep Save disabled.
  it.each([
    ['a valid', '45', false],
    ['an out-of-range', '5000', true],
  ])(
    'drops %s retention typed before the HA lock arrives',
    async (_, typed, invalid) => {
      let resolveStatus: (value: { status: HAStatus }) => void = () => {};
      getHAStatusMock.mockReturnValue(
        new Promise((resolve) => {
          resolveStatus = resolve;
        })
      );
      renderForm(SETTINGS_MOCK);

      fireEvent.change(retentionInput(), { target: { value: typed } });
      expect(retentionInput()).toHaveValue(Number(typed));
      await waitFor(() =>
        expect(retentionInput()).toHaveAttribute(
          'aria-invalid',
          String(invalid)
        )
      );

      resolveStatus({ status: 'Enabled' });
      await waitFor(() => expect(retentionInput()).toBeDisabled());
      expect(retentionInput()).toHaveValue(30);
      expectNoRetentionError();

      await submitAnotherSetting();

      await waitFor(() => expect(updateSettingsMock).toHaveBeenCalled());
      expect(updateSettingsMock.mock.calls[0][0]).not.toHaveProperty(
        'dataRetention'
      );
    }
  );
});

describe('toPayload', () => {
  const values = { ...toFormValues(SETTINGS_MOCK), retention: '45' };

  it('sends a changed retention', () => {
    expect(toPayload(values, '30')).toHaveProperty(
      'dataRetention',
      `${45 * 24 * 60 * 60}s`
    );
  });

  // The form resets a locked field, but the payload must not rely on that having happened.
  it('leaves a changed retention out when the field is locked', () => {
    expect(toPayload(values, '30', true)).not.toHaveProperty('dataRetention');
  });
});
