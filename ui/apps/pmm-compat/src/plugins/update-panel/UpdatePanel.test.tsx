import React from 'react';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { OrgRole } from '@grafana/data';
import { config } from '@grafana/runtime';
import { getReadonlySettings } from 'plugins/shared/api/settings';
import { checkForUpdates } from 'plugins/shared/api/updates';
import { UpdatePanel } from './UpdatePanel';

jest.mock('plugins/shared/api/settings');
jest.mock('plugins/shared/api/updates');
jest.mock('@grafana/ui', () => ({
  ...jest.requireActual('@grafana/ui'),
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

const checkForUpdatesMock = jest.mocked(checkForUpdates);
const getReadonlySettingsMock = jest.mocked(getReadonlySettings);

const renderPanel = () =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <UpdatePanel />
    </QueryClientProvider>
  );

describe('UpdatePanel::', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    config.bootData.user.orgRole = OrgRole.Admin;
    getReadonlySettingsMock.mockResolvedValue({
      updatesEnabled: true,
      advisorEnabled: true,
    });
    checkForUpdatesMock.mockResolvedValue({
      installed: { version: '3.4.0', timestamp: '2025-01-01T00:00:00Z' },
      latest: { version: '3.5.0', timestamp: '2025-02-01T00:00:00Z' },
      updateAvailable: true,
      lastChecked: '2025-02-02T10:00:00Z',
    });
  });

  it('should offer an upgrade to an admin when an update is available', async () => {
    renderPanel();

    expect(await screen.findByText('Upgrade to 3.5.0')).toBeInTheDocument();
    expect(screen.getByTestId('update-installed-version')).toHaveTextContent(
      '3.4.0'
    );
    expect(screen.getByTestId('update-last-check')).toHaveTextContent(
      'February 02, 10:00 UTC'
    );
  });

  it('should fall back to the installed version when the check fails', async () => {
    checkForUpdatesMock
      .mockRejectedValueOnce(new Error('failed'))
      .mockResolvedValueOnce({
        installed: { version: '3.4.0' },
        updateAvailable: false,
      });

    renderPanel();

    expect(
      await screen.findByTestId('update-installed-version')
    ).toHaveTextContent('3.4.0');
    expect(checkForUpdatesMock).toHaveBeenLastCalledWith({
      force: false,
      only_installed_version: true,
    });
  });

  it('should not check for updates for a non-admin user', async () => {
    config.bootData.user.orgRole = OrgRole.Viewer;

    renderPanel();

    expect(
      await screen.findByText('Insufficient access permissions')
    ).toBeInTheDocument();
    expect(checkForUpdatesMock).not.toHaveBeenCalled();
  });

  it('should not check for updates when they are disabled', async () => {
    getReadonlySettingsMock.mockResolvedValue({
      updatesEnabled: false,
      advisorEnabled: true,
    });

    renderPanel();

    expect(await screen.findByTestId('updates-disabled')).toBeInTheDocument();
    expect(checkForUpdatesMock).not.toHaveBeenCalled();
  });
});
