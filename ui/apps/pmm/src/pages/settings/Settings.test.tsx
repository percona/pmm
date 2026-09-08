import { render, screen, waitFor } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { Settings } from './Settings';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import * as settingsApi from 'api/settings';
import * as versionApi from 'api/version';
import { SETTINGS_MOCK } from 'api/__mocks__/settings';
import { VERSION_MOCK } from 'api/__mocks__/version';
import { DistributionMethod } from 'types/version.types';

vi.mock('api/settings');
vi.mock('api/version');
vi.mock('./components/metrics-resolution/MetricsResolutionForm', () => ({
  MetricsResolutionForm: () => null,
}));
vi.mock('./components/advanced/AdvancedSettingsForm', () => ({
  AdvancedSettingsForm: () => null,
}));
vi.mock('./components/ssh-key/SshKeyForm', () => ({
  SshKeyForm: () => null,
}));

const getSettingsMock = vi.mocked(settingsApi.getSettings);
const getVersionMock = vi.mocked(versionApi.getVersion);

const renderWithRoute = (initialPath: string) =>
  render(
    <TestWrapper routerProps={{ initialEntries: [initialPath] }}>
      {wrapWithQueryProvider(
        <Routes>
          <Route path="/settings/:tab?" element={<Settings />} />
        </Routes>
      )}
    </TestWrapper>
  );

describe('Settings', () => {
  beforeEach(() => {
    getSettingsMock.mockResolvedValue(SETTINGS_MOCK);
    getVersionMock.mockResolvedValue(VERSION_MOCK);
  });

  it('shows loading state when settings are not yet loaded', () => {
    getSettingsMock.mockImplementation(() => new Promise(() => {}));

    render(<TestWrapper>{wrapWithQueryProvider(<Settings />)}</TestWrapper>);

    expect(screen.getByTestId('settings-loading')).toBeInTheDocument();
  });

  describe('tab navigation by URL', () => {
    it('ssh tab is not shown when distribution type is not AMI', async () => {
      renderWithRoute('/settings/metrics-resolution');

      await screen.findByTestId('settings-tab-metrics');

      expect(screen.queryByTestId('settings-tab-ssh')).not.toBeInTheDocument();
    });

    it('activates metrics tab for /settings/metrics', async () => {
      renderWithRoute('/settings/metrics-resolution');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-metrics')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
    });

    it('activates advanced tab for /settings/advanced', async () => {
      renderWithRoute('/settings/advanced-settings');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-advanced')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
    });

    it('activates ssh tab for /settings/ssh when distributed as AMI', async () => {
      getVersionMock.mockResolvedValueOnce({
        ...VERSION_MOCK,
        distributionMethod: DistributionMethod.ami,
      });

      renderWithRoute('/settings/ssh-key');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-ssh')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
    });

    it('redirects from ssh tab for /settings/ssh to default', async () => {
      renderWithRoute('/settings/ssh-key');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-metrics')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
    });

    it('defaults to metrics tab when no tab is in the URL', async () => {
      renderWithRoute('/settings');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-metrics')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
    });

    it('shows ssh tab when distribution type is AMI', async () => {
      getVersionMock.mockResolvedValueOnce({
        ...VERSION_MOCK,
        distributionMethod: DistributionMethod.ami,
      });

      renderWithRoute('/settings/metrics-resolution');
      await waitFor(() =>
        expect(screen.queryByTestId('settings-tab-ssh')).toBeInTheDocument()
      );
    });
  });
});
