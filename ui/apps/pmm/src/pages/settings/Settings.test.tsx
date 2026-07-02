import { render, screen, waitFor } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
import { Settings } from './Settings';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import * as settingsApi from 'api/settings';
import type { Settings as SettingsType } from 'types/settings.types';

vi.mock('api/settings');
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
const mockSettings = {} as SettingsType;

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
    getSettingsMock.mockImplementation(() => new Promise(() => {}));
  });

  it('shows loading state when settings are not yet loaded', () => {
    render(<TestWrapper>{wrapWithQueryProvider(<Settings />)}</TestWrapper>);

    expect(screen.getByTestId('settings-loading')).toBeInTheDocument();
  });

  describe('tab navigation by URL', () => {
    beforeEach(() => {
      getSettingsMock.mockResolvedValue(mockSettings);
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

    it('activates ssh tab for /settings/ssh', async () => {
      renderWithRoute('/settings/ssh-key');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-ssh')).toHaveAttribute(
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
  });
});
