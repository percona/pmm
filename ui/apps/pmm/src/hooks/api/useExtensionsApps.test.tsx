import type { ReactElement, ReactNode } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderHook, waitFor } from '@testing-library/react';
import { getExtensionsApps } from 'api/extensionsApps';
import { TEST_USER_VIEWER } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings } from 'utils/testUtils';
import type { User } from 'types/user.types';
import {
  EXTENSIONS_APPS_QUERY_KEY,
  useExtensionsApps,
  useRefreshExtensionsApps,
} from './useExtensionsApps';

vi.mock('api/extensionsApps', () => ({
  getExtensionsApps: vi.fn(),
}));

const fetchApps = vi.mocked(getExtensionsApps);

const LISTED = [
  {
    app_key: 'pg_backups',
    blocking_dependencies: [],
    custom_ui: false,
    display_name: 'PostgreSQL Backups',
    enabled: true,
    group: null,
    nav_icon: null,
    nav_order: 10,
    react_route: '/apps/pg_backups',
    sidebar: true,
    uri_path: '/pg_backups',
  },
];

const setup = ({
  user = TEST_USER_VIEWER,
  extensionsEnabled = true,
}: { user?: User; extensionsEnabled?: boolean } = {}) => {
  const queryClient = new QueryClient();
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <TestWrapper userContext={{ isLoading: false, user }}>
        {wrapWithSettings(children as ReactElement, {
          settings: { extensionsEnabled },
        })}
      </TestWrapper>
    </QueryClientProvider>
  );
  return { queryClient, wrapper };
};

beforeEach(() => {
  fetchApps.mockReset().mockResolvedValue(LISTED);
});

describe('useExtensionsApps', () => {
  it('returns the apps in the shape PMM owns', async () => {
    const { wrapper } = setup();

    const { result } = renderHook(() => useExtensionsApps(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toMatchObject([
      { key: 'pg_backups', slug: 'pg-backups', label: 'PostgreSQL Backups' },
    ]);
  });

  it('does not reach the side-car while PMM Extensions is disabled', () => {
    const { wrapper } = setup({ extensionsEnabled: false });

    renderHook(() => useExtensionsApps(), { wrapper });

    expect(fetchApps).not.toHaveBeenCalled();
  });

  it('keeps the list for the session instead of refetching on mount', async () => {
    const { wrapper } = setup();
    const first = renderHook(() => useExtensionsApps(), { wrapper });
    await waitFor(() => expect(first.result.current.isSuccess).toBe(true));

    renderHook(() => useExtensionsApps(), { wrapper });

    expect(fetchApps).toHaveBeenCalledOnce();
  });

  it('does not retry a failed fetch', async () => {
    fetchApps.mockRejectedValue(new Error('side-car down'));
    const { wrapper } = setup();

    const { result } = renderHook(() => useExtensionsApps(), { wrapper });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(fetchApps).toHaveBeenCalledOnce();
  });

  it('is refreshed when the side-car apps are invalidated', async () => {
    const { wrapper, queryClient } = setup();
    const { result } = renderHook(() => useExtensionsApps(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    // What `useAdminApps` does after an app is enabled or disabled.
    await queryClient.invalidateQueries({ queryKey: ['apps'] });

    expect(fetchApps).toHaveBeenCalledTimes(2);
  });
});

describe('useRefreshExtensionsApps', () => {
  it('fetches the list again when it mounts', async () => {
    const { wrapper, queryClient } = setup();
    const { result } = renderHook(() => useExtensionsApps(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    renderHook(() => useRefreshExtensionsApps(), { wrapper });

    await waitFor(() => expect(fetchApps).toHaveBeenCalledTimes(2));
    expect(
      queryClient.getQueryState(EXTENSIONS_APPS_QUERY_KEY)?.isInvalidated
    ).toBe(false);
  });

  it('fetches the list again for each app the user moves to', async () => {
    const { wrapper } = setup();
    const { result } = renderHook(() => useExtensionsApps(), { wrapper });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const refresh = renderHook(
      ({ appSlug }) => useRefreshExtensionsApps(appSlug),
      { wrapper, initialProps: { appSlug: 'pg-backups' } }
    );
    await waitFor(() => expect(fetchApps).toHaveBeenCalledTimes(2));

    refresh.rerender({ appSlug: 'pg-backups' });
    refresh.rerender({ appSlug: 'disk-free' });

    await waitFor(() => expect(fetchApps).toHaveBeenCalledTimes(3));
  });

  it('starts over instead of joining a fetch that is still in flight', async () => {
    let failFirst: (error: Error) => void = () => {};
    fetchApps.mockImplementationOnce(
      () =>
        new Promise((_, reject) => {
          failFirst = reject;
        })
    );
    const { wrapper } = setup();
    const { result } = renderHook(() => useExtensionsApps(), { wrapper });
    await waitFor(() => expect(fetchApps).toHaveBeenCalledOnce());

    renderHook(() => useRefreshExtensionsApps(), { wrapper });
    await waitFor(() => expect(fetchApps).toHaveBeenCalledTimes(2));
    failFirst(new Error('exchange failed'));

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toHaveLength(1);
  });
  it('aborts the requests of the fetch it replaces', async () => {
    fetchApps.mockImplementationOnce(() => new Promise(() => {}));
    const { wrapper } = setup();
    renderHook(() => useExtensionsApps(), { wrapper });
    await waitFor(() => expect(fetchApps).toHaveBeenCalledOnce());
    const [firstSignal] = fetchApps.mock.calls[0];
    expect(firstSignal?.aborted).toBe(false);

    renderHook(() => useRefreshExtensionsApps(), { wrapper });

    await waitFor(() => expect(fetchApps).toHaveBeenCalledTimes(2));
    expect(firstSignal?.aborted).toBe(true);
  });
});
