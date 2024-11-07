import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { UpdatesContext, UpdatesContextProps } from 'contexts/updates';
import { ReactElement } from 'react';
import { UpdateStatus } from 'types/updates.types';

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

export const wrapWithQueryProvider = (children: ReactElement) => (
  <QueryClientProvider
    client={
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
