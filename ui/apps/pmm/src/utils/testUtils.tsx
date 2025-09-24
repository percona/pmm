import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { UpdatesContext, UpdatesContextProps } from 'contexts/updates';
import { UserContext, UserContextProps } from 'contexts/user';
import { ReactElement } from 'react';
import { UpdateStatus } from 'types/updates.types';
import { TEST_USER_ADMIN } from './testStubs';
import { MemoryRouter, MemoryRouterProps } from 'react-router-dom';
import { SettingsContext } from 'contexts/settings';
import { FrontendSettings, Settings } from 'types/settings.types';
import { GrafanaContext, GrafanaContextProps } from 'contexts/grafana';

export const wrapWithUpdatesProvider = (
  children: ReactElement,
  value: Partial<UpdatesContextProps> = {}
) => (
  <UpdatesContext.Provider
    value={{
      inProgress: false,
      isLoading: false,
      status: UpdateStatus.UpToDate,
      recheck: () => {},
      setStatus: () => {},
      versionInfo: {
        installed: {
          version: '3.0.0',
          fullVersion: '3.0.0',
          timestamp: '2024-07-23T00:00:00Z',
        },
        latest: {
          version: '3.0.0',
          tag: '',
          timestamp: null,
          releaseNotesText: '',
          releaseNotesUrl: '',
        },
        updateAvailable: false,
        latestNewsUrl: 'https://per.co.na/pmm/3.0.0',
        lastCheck: '2024-07-30T10:34:05.886739003Z',
      },
      areClientsUpToDate: true,
      ...value,
    }}
  >
    {children}
  </UpdatesContext.Provider>
);

export const wrapWithQueryProvider = (
  children: ReactElement,
  client?: QueryClient
) => (
  <QueryClientProvider
    client={
      client ??
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: false,
          },
        },
      })
    }
  >
    {children}
  </QueryClientProvider>
);

export const wrapWithUserProvider = (
  children: ReactElement,
  ctx: Partial<UserContextProps> = {}
) => (
  <UserContext.Provider
    value={{
      isLoading: false,
      ...ctx,
      user: {
        ...TEST_USER_ADMIN,
        ...ctx.user,
      },
    }}
  >
    {children}
  </UserContext.Provider>
);

export const wrapWithRouter = (
  children: ReactElement,
  props?: Partial<MemoryRouterProps>
) => <MemoryRouter {...props}>{children}</MemoryRouter>;

export const wrapWithSettings = (
  children: ReactElement,
  props?: {
    isLoading?: boolean;
    settings?: Partial<Settings>;
    frontend?: Partial<FrontendSettings>;
  }
) => (
  <SettingsContext.Provider
    value={{
      isLoading: props?.isLoading ?? false,
      settings: {
        newUIEnabled: true,
        updatesEnabled: false,
        telemetryEnabled: false,
        advisorEnabled: false,
        alertingEnabled: false,
        pmmPublicAddress: '',
        backupManagementEnabled: false,
        azurediscoverEnabled: false,
        enableAccessControl: false,
        updatesSnoozeDuration: '10s',
        ...props?.settings,
        frontend: {
          anonymousEnabled: false,
          appSubUrl: '',
          apps: {},
          buildInfo: {
            version: '',
            versionString: '',
          },
          exploreEnabled: true,
          disableLoginForm: false,
          unifiedAlertingEnabled: true,
          auth: {
            disableLogin: false,
          },
          featureToggles: {
            exploreMetrics: true,
          },
          ...props?.frontend,
        },
      },
    }}
  >
    {children}
  </SettingsContext.Provider>
);

export const wrapWithGrafana = (
  children: ReactElement,
  props: Partial<GrafanaContextProps> = {}
) => (
  <GrafanaContext.Provider
    value={{
      isFrameLoaded: true,
      isFullScreen: false,
      isOnGrafanaPage: true,
      ...props,
    }}
  >
    {children}
  </GrafanaContext.Provider>
);
