import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import { UserContext } from 'contexts/user';
import type { ReactNode } from 'react';
import { MemoryRouter, useNavigate } from 'react-router-dom';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { DASHBOARDS_QUERY_KEY } from './search.constants';
import { getRecentKey, readRecent, resolveRecent } from './search.recent';
import { staticIndexFor, testItem } from './search.testFixtures';
import type { SearchItem } from './search.types';
import { fetchDashboardItems } from './sources/dashboards.source';
import type * as DashboardsSource from './sources/dashboards.source';
import { useRecentVisits } from './useRecentVisits';

vi.mock('./sources/dashboards.source', async (importOriginal) => ({
  ...(await importOriginal<typeof DashboardsSource>()),
  fetchDashboardItems: vi.fn(),
}));

const KEY = getRecentKey(TEST_USER_ADMIN);

const STATIC_ITEMS = [
  testItem('Settings', {
    id: 'page:configuration-settings',
    to: '/settings',
    dedupeKey: '/settings',
  }),
  testItem('Switch to dark mode', {
    id: 'action:theme-toggle',
    kind: 'action',
    group: 'actions',
  }),
];

const CUSTOM = testItem('Custom', {
  id: 'dashboard:custom',
  kind: 'dashboard',
  group: 'dashboards',
  to: '/graph/d/custom/custom',
  dedupeKey: 'dashboard:custom',
});

const recentIds = () => readRecent(KEY).map((entry) => entry.id);

// Lets a macrotask pass, so every resolved lookup has run.
const settle = () =>
  act(() => new Promise<void>((resolve) => setTimeout(resolve)));

const render = (
  path: string,
  {
    items = STATIC_ITEMS,
    queryClient = new QueryClient(),
  }: { items?: SearchItem[]; queryClient?: QueryClient } = {}
) => {
  const { result } = renderHook(
    () => {
      useRecentVisits(items);

      return useNavigate();
    },
    {
      wrapper: ({ children }: { children: ReactNode }) => (
        <QueryClientProvider client={queryClient}>
          <UserContext.Provider
            value={{ isLoading: false, user: TEST_USER_ADMIN }}
          >
            <MemoryRouter initialEntries={[path]}>{children}</MemoryRouter>
          </UserContext.Provider>
        </QueryClientProvider>
      ),
    }
  );

  return {
    navigate: (to: string) =>
      act(() => {
        void result.current(to, { replace: true });
      }),
  };
};

describe('useRecentVisits', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.mocked(fetchDashboardItems).mockReset();
  });

  it('records a page of the static index', () => {
    render('/settings/');

    expect(readRecent(KEY).map((entry) => entry.id)).toEqual([
      'page:configuration-settings',
    ]);
    expect(fetchDashboardItems).not.toHaveBeenCalled();
  });

  it('looks an unknown dashboard up in Grafana’s list and records it', async () => {
    vi.mocked(fetchDashboardItems).mockResolvedValue([CUSTOM]);

    render('/graph/d/custom/custom-slug?orgId=1');

    await waitFor(() => expect(recentIds()).toEqual(['dashboard:custom']));
    expect(fetchDashboardItems).toHaveBeenCalledTimes(1);
  });

  it('refetches a cached list that predates the dashboard, once per dashboard', async () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(DASHBOARDS_QUERY_KEY, []);
    vi.mocked(fetchDashboardItems).mockResolvedValue([CUSTOM]);

    render('/graph/d/custom/custom-slug', { queryClient });

    await waitFor(() => expect(recentIds()).toEqual(['dashboard:custom']));
    expect(fetchDashboardItems).toHaveBeenCalledTimes(1);

    queryClient.setQueryData(DASHBOARDS_QUERY_KEY, []);
    vi.mocked(fetchDashboardItems).mockResolvedValue([]);
    const { navigate } = render('/graph/d/unknown/x', { queryClient });

    await waitFor(() => expect(fetchDashboardItems).toHaveBeenCalledTimes(2));
    navigate('/settings');
    navigate('/graph/d/unknown/x');
    await settle();

    expect(fetchDashboardItems).toHaveBeenCalledTimes(2);
  });

  it('records Overview once across Grafana’s redirect to the home dashboard', async () => {
    const items = staticIndexFor();
    vi.mocked(fetchDashboardItems).mockResolvedValue([
      testItem('Home Dashboard', {
        id: 'dashboard:pmm-home',
        kind: 'dashboard',
        group: 'dashboards',
        to: '/graph/d/pmm-home/home-dashboard',
        dedupeKey: 'dashboard:pmm-home',
        remote: true,
      }),
    ]);
    const { navigate } = render('/graph', { items });

    navigate('/graph/d/pmm-home/home-dashboard');
    await settle();

    expect(recentIds()).toEqual(['page:home-page']);
    expect(fetchDashboardItems).not.toHaveBeenCalled();
    expect(
      resolveRecent(readRecent(KEY), items, '/graph/d/pmm-home/home-dashboard')
    ).toEqual([]);
  });

  it('skips a path it cannot resolve', async () => {
    vi.mocked(fetchDashboardItems).mockResolvedValue([]);

    render('/graph/d/unknown/x');
    render('/nowhere');

    await waitFor(() => expect(fetchDashboardItems).toHaveBeenCalledTimes(1));
    await settle();
    expect(fetchDashboardItems).toHaveBeenCalledTimes(1);
    expect(readRecent(KEY)).toEqual([]);
  });
});
