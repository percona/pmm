import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render } from '@testing-library/react';
import { api, grafanaApi } from 'api/api';
import type { CombinedSettings } from 'contexts/settings';
import type { PropsWithChildren, ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import type { ManagedService } from 'types/services.types';
import {
  buildProposedNavTree,
  DEFAULT_STARRED_APPS,
} from 'contexts/navigation/navigation.proposal';
import type { HAInfo } from 'types/ha.types';
import { UpdateStatus } from 'types/updates.types';
import type { User } from 'types/user.types';
import { DistributionMethod, type VersionResponse } from 'types/version.types';
import {
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import type { SearchItem } from './search.types';
import type { GrafanaSearchHit } from './sources/dashboards.source';
import type { ManagedNode } from './sources/inventory.source';
import { buildStaticIndex } from './useSearchIndex';

const HA_OFF: HAInfo = { enabled: false, health: 'unknown', nodes: [] };

export const ALL_FEATURES_ON = {
  advisorEnabled: true,
  backupManagementEnabled: true,
  extensionsEnabled: true,
  omEnabled: true,
} as CombinedSettings;

export const AMI_VERSION = {
  distributionMethod: DistributionMethod.ami,
} as VersionResponse;

export const navTreeFor = (user: User) =>
  buildProposedNavTree({
    user,
    isLoggedIn: true,
    haInfo: HA_OFF,
    updateStatus: UpdateStatus.UpToDate,
    colorMode: 'light',
    toggleColorMode: () => {},
    starredApps: DEFAULT_STARRED_APPS,
  });

export const staticIndexFor = (
  user: User = TEST_USER_ADMIN,
  {
    settings = ALL_FEATURES_ON,
    version = AMI_VERSION,
  }: { settings?: CombinedSettings; version?: VersionResponse } = {}
) =>
  buildStaticIndex(navTreeFor(user), {
    isAdmin: user.isPMMAdmin,
    isEditor: user.isEditor || user.isPMMAdmin,
    settings,
    version,
    startTour: () => {},
  });

export const ROLE_USERS = {
  admin: TEST_USER_ADMIN,
  editor: TEST_USER_EDITOR,
  viewer: TEST_USER_VIEWER,
};

export const testItem = (
  label: string,
  fields: Partial<SearchItem> = {}
): SearchItem => ({
  id: `page:${label}`,
  kind: 'page',
  group: 'pages',
  label,
  keywords: [],
  dedupeKey: `page:${label}`,
  order: 0,
  ...fields,
});

export const setPlatform = (value: string) =>
  Object.defineProperty(navigator, 'platform', { value, configurable: true });

const LocationDisplay = () => (
  <div data-testid="location">{useLocation().pathname}</div>
);

// Auth, user, router, theme and a fresh query client, with the current path shown in a `location` test id.
export const renderWithSearchContexts = (
  ui: ReactElement,
  {
    user = TEST_USER_ADMIN,
    path = '/help',
  }: { user?: User; path?: string } = {}
) => {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  return render(ui, {
    wrapper: ({ children }: PropsWithChildren) => (
      <TestWrapper
        userContext={{ isLoading: false, user }}
        routerProps={{ initialEntries: [path] }}
      >
        <QueryClientProvider client={queryClient}>
          {children}
          <LocationDisplay />
        </QueryClientProvider>
      </TestWrapper>
    ),
  });
};

export const TEST_HITS: GrafanaSearchHit[] = [
  {
    uid: 'mysql-custom',
    title: 'MySQL custom overview',
    url: '/graph/d/mysql-custom/mysql-custom-overview',
    type: 'dash-db',
    tags: ['mysql'],
  },
];

export const TEST_SERVICES: Partial<ManagedService>[] = [
  {
    serviceId: 'svc-1',
    serviceType: 'mysql',
    serviceName: 'mysql-prod',
    nodeName: 'db-host',
  },
];

export const TEST_NODES: ManagedNode[] = [
  {
    nodeId: 'n1',
    nodeName: 'db-host',
    nodeType: 'generic',
    address: '10.0.0.1',
  },
];

// Answers the remote sources from fixtures; `dashboards: 'error'` makes Grafana's search fail.
export const mockSearchApis = ({
  dashboards = TEST_HITS,
}: { dashboards?: GrafanaSearchHit[] | 'error' } = {}) => {
  const searchGet = vi
    .spyOn(grafanaApi, 'get')
    .mockImplementation(() =>
      dashboards === 'error'
        ? Promise.reject(new Error('unavailable'))
        : Promise.resolve({ data: dashboards })
    );
  const apiGet = vi.spyOn(api, 'get').mockImplementation((url: string) =>
    Promise.resolve({
      data: url.includes('/management/services')
        ? { services: TEST_SERVICES }
        : url.includes('/management/nodes')
          ? { nodes: TEST_NODES }
          : AMI_VERSION,
    })
  );

  return { searchGet, apiGet };
};
