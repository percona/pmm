import { QueryClient } from '@tanstack/react-query';
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

// The field is editable until the HA status arrives, so anything asserted while the query is
// still in flight holds whatever the answer turns out to be. Render, then wait for the status to
// land in the cache, so every assertion below is made against the settled form.
const renderWithSettledHA = async (status: HAStatus) => {
  getHAStatusMock.mockResolvedValue({ status });

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <TestWrapper>
      {wrapWithQueryProvider(
        <AdvancedSettingsForm settings={settings} />,
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
