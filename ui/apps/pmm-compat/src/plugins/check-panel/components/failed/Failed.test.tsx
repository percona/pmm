import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { OrgRole, type PanelProps } from '@grafana/data';
import { config } from '@grafana/runtime';
import { getFailedServices } from 'plugins/shared/api/advisors';
import { getReadonlySettings } from 'plugins/shared/api/settings';
import type { FailedCheckSummary } from 'plugins/shared/types/advisors.types';
import { Failed } from './Failed';

jest.mock('plugins/shared/api/advisors');
jest.mock('plugins/shared/api/settings');

const getFailedServicesMock = jest.mocked(getFailedServices);
const getReadonlySettingsMock = jest.mocked(getReadonlySettings);

const zeroCounts = {
  emergency: 0,
  critical: 0,
  alert: 0,
  error: 0,
  warning: 0,
  notice: 0,
  info: 0,
  debug: 0,
};

const failedServices: FailedCheckSummary[] = [
  {
    serviceName: '',
    serviceId: '',
    counts: { ...zeroCounts, critical: 1, warning: 2 },
  },
  { serviceName: '', serviceId: '', counts: { ...zeroCounts, critical: 2 } },
];

const panelProps = {
  eventBus: {
    getStream: jest.fn(() => ({
      subscribe: () => ({ unsubscribe: jest.fn() }),
    })),
  },
} as unknown as PanelProps;

const setUser = (orgRole: OrgRole, isGrafanaAdmin = false) => {
  config.bootData.user.orgRole = orgRole;
  config.bootData.user.isGrafanaAdmin = isGrafanaAdmin;
};

const renderFailed = () =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <Failed {...panelProps} />
    </QueryClientProvider>
  );

describe('Failed::', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setUser(OrgRole.Admin, true);
    getReadonlySettingsMock.mockResolvedValue({
      advisorEnabled: true,
      updatesEnabled: true,
    });
  });

  it.each([
    ['admin', OrgRole.Admin],
    ['editor', OrgRole.Editor],
  ])(
    'should render a sum of failed checks by severity for %s user',
    async (_, role) => {
      setUser(role);
      getFailedServicesMock.mockResolvedValue(failedServices);

      renderFailed();

      await waitFor(() =>
        expect(screen.getByTestId('db-check-panel-critical')).toHaveTextContent(
          '3'
        )
      );
      expect(screen.getByTestId('db-check-panel-error')).toHaveTextContent('0');
      expect(screen.getByTestId('db-check-panel-warning')).toHaveTextContent(
        '2'
      );
      expect(screen.getByTestId('db-check-panel-notice')).toHaveTextContent(
        '0'
      );
    }
  );

  it.each([
    ['admin', OrgRole.Admin],
    ['editor', OrgRole.Editor],
  ])(
    'should render 0 when there are no failed checks for %s user',
    async (_, role) => {
      setUser(role);
      getFailedServicesMock.mockResolvedValue([
        { serviceName: '', serviceId: '', counts: zeroCounts },
      ]);

      renderFailed();

      expect(
        await screen.findByTestId('db-check-panel-zero-checks')
      ).toBeInTheDocument();
    }
  );

  it('should render unauthorized message and not fetch checks for viewer user', async () => {
    setUser(OrgRole.Viewer);

    renderFailed();

    expect(await screen.findByTestId('unauthorized')).toBeInTheDocument();
    expect(getFailedServicesMock).not.toHaveBeenCalled();
  });

  it('should link to settings when advisors are disabled', async () => {
    getReadonlySettingsMock.mockResolvedValue({
      advisorEnabled: false,
      updatesEnabled: true,
    });
    getFailedServicesMock.mockResolvedValue(failedServices);

    renderFailed();

    expect(
      await screen.findByTestId('db-check-panel-settings-link')
    ).toBeInTheDocument();
  });
});
