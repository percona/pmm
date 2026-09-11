import { render, screen, waitFor } from '@testing-library/react';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import * as haApi from 'api/ha';
import type { HAStatus } from 'types/ha.types';
import type { Settings as SettingsType } from 'types/settings.types';
import { AdvancedSettingsForm } from './AdvancedSettingsForm';

vi.mock('api/settings');
vi.mock('api/ha', () => ({
  getHAStatus: vi.fn(),
  getHANodes: vi.fn(),
}));

const getHAStatusMock = vi.mocked(haApi.getHAStatus);

const settings = { dataRetention: '2592000s' } as SettingsType;

const renderWithHA = (status: HAStatus) => {
  getHAStatusMock.mockResolvedValue({ status });

  return render(
    <TestWrapper>
      {wrapWithQueryProvider(<AdvancedSettingsForm settings={settings} />)}
    </TestWrapper>
  );
};

const retentionInput = () => screen.getByTestId('retention-number-input');

describe('AdvancedSettingsForm data retention in HA', () => {
  it('is editable when high availability is disabled', async () => {
    renderWithHA('Disabled');

    await waitFor(() => expect(getHAStatusMock).toHaveBeenCalled());
    expect(retentionInput()).toBeEnabled();
    expect(screen.queryByText(/dataRetentionDays/)).not.toBeInTheDocument();
  });

  it('is disabled and points at the chart when high availability is enabled', async () => {
    renderWithHA('Enabled');

    await waitFor(() => expect(retentionInput()).toBeDisabled());
    expect(screen.getByText(/dataRetentionDays/)).toBeInTheDocument();
  });
});
