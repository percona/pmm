import { ComponentProps } from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { useSettingsList } from '@sep/api';
import { Settings } from './Settings';
import { TestWrapper } from 'utils/testWrapper';
import {
  measurePageSurface,
  measureSurface,
  wrapWithQueryProvider,
} from 'utils/testUtils';
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
vi.mock('./components/advisors/AdvisorsForm', () => ({
  AdvisorsForm: () => null,
}));
vi.mock('./components/ssh-key/SshKeyForm', () => ({
  SshKeyForm: () => null,
}));
vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  useSettingsList: vi.fn(),
}));
// The stub stands in for the tab's SEP read (the real one reaches
// `useSettingsList` through `useServiceNowConnection`), so the disabled case
// can assert the route fires no SEP request rather than only that the tab is
// absent from the DOM.
vi.mock('./components/servicenow', () => ({
  ServiceNowConnectionTab: () => {
    useSettingsList();
    return <div data-testid="servicenow-tab" />;
  },
}));

const getSettingsMock = vi.mocked(settingsApi.getSettings);
const getVersionMock = vi.mocked(versionApi.getVersion);
const useSettingsListMock = vi.mocked(useSettingsList);

const LocationProbe = () => (
  <span data-testid="location-probe">{useLocation().pathname}</span>
);

const renderWithRoute = (
  initialPath: string,
  wrapperProps?: Partial<ComponentProps<typeof TestWrapper>>
) =>
  render(
    <TestWrapper
      routerProps={{ initialEntries: [initialPath] }}
      {...wrapperProps}
    >
      {wrapWithQueryProvider(
        <>
          <LocationProbe />
          <Routes>
            <Route path="/settings/:tab?" element={<Settings />} />
          </Routes>
        </>
      )}
    </TestWrapper>
  );

describe('Settings', () => {
  beforeEach(() => {
    useSettingsListMock.mockClear();
    getSettingsMock.mockResolvedValue(SETTINGS_MOCK);
    getVersionMock.mockResolvedValue(VERSION_MOCK);
  });

  it('shows loading state when settings are not yet loaded', () => {
    getSettingsMock.mockImplementation(() => new Promise(() => {}));

    render(<TestWrapper>{wrapWithQueryProvider(<Settings />)}</TestWrapper>);

    expect(screen.getByTestId('settings-loading')).toBeInTheDocument();
  });

  it('shows the loading state on the paper surface the loaded page uses', () => {
    const stage = measurePageSurface('default');
    const paper = measurePageSurface('paper');

    // Guards the assertion below: it only means anything while the two
    // surfaces actually differ.
    expect(paper).not.toBe(stage);

    getSettingsMock.mockImplementation(() => new Promise(() => {}));

    const loading = measureSurface(() => {
      const result = render(
        <TestWrapper>{wrapWithQueryProvider(<Settings />)}</TestWrapper>
      );
      expect(screen.getByTestId('settings-loading')).toBeInTheDocument();

      return result;
    });

    expect(loading).toBe(paper);
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

    it('activates advisors tab for /settings/advisors', async () => {
      renderWithRoute('/settings/advisors');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-advisors')).toHaveAttribute(
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

    it('activates the ServiceNow tab for /settings/servicenow-connection when SEP is enabled', async () => {
      getSettingsMock.mockResolvedValue({ ...SETTINGS_MOCK, sepEnabled: true });

      renderWithRoute('/settings/servicenow-connection');
      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-servicenow')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
      expect(screen.getByTestId('servicenow-tab')).toBeInTheDocument();
      expect(useSettingsListMock).toHaveBeenCalled();
    });

    it('shows the ServiceNow tab when SEP is enabled', async () => {
      getSettingsMock.mockResolvedValue({ ...SETTINGS_MOCK, sepEnabled: true });

      renderWithRoute('/settings/metrics-resolution');

      await waitFor(() =>
        expect(
          screen.getByTestId('settings-tab-servicenow')
        ).toBeInTheDocument()
      );
    });

    it('does not show the ServiceNow tab when SEP is disabled', async () => {
      renderWithRoute('/settings/metrics-resolution');

      await screen.findByTestId('settings-tab-metrics');

      expect(
        screen.queryByTestId('settings-tab-servicenow')
      ).not.toBeInTheDocument();
    });

    it('redirects from /settings/servicenow-connection to default when SEP is disabled', async () => {
      renderWithRoute('/settings/servicenow-connection');

      await waitFor(() =>
        expect(screen.getByTestId('settings-tab-metrics')).toHaveAttribute(
          'aria-selected',
          'true'
        )
      );
      expect(
        screen.queryByTestId('settings-tab-servicenow')
      ).not.toBeInTheDocument();
      expect(screen.queryByTestId('servicenow-tab')).not.toBeInTheDocument();
      expect(useSettingsListMock).not.toHaveBeenCalled();
      expect(screen.getByTestId('location-probe')).toHaveTextContent(
        '/settings'
      );
    });

    it('keeps the ServiceNow URL while the user is still resolving', async () => {
      renderWithRoute('/settings/servicenow-connection', {
        userContext: { isLoading: false, user: undefined },
      });

      await waitFor(() =>
        expect(screen.getByTestId('location-probe')).toHaveTextContent(
          '/settings/servicenow-connection'
        )
      );
      expect(useSettingsListMock).not.toHaveBeenCalled();
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
