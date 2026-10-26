import { act, renderHook } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { useLocation, useNavigationType } from 'react-router-dom';
import type { Settings } from 'types/settings.types';
import type { User } from 'types/user.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { wrapWithSettings } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { testView } from './hub.testFixtures';
import type {
  HubDeriveContext,
  HubLiveContext,
  HubMessages,
  HubSourceId,
  HubSourceState,
  TechnologyHubConfig,
} from './hub.types';
import {
  HUB_STALE_FOR_MS,
  liveFreshness,
  useTechnologyHub,
} from './useTechnologyHub';

interface FakeSnapshot {
  from: 'live' | 'fixture';
  now?: number;
  ops?: boolean;
}

const sources = (
  overrides: Partial<Record<HubSourceId, HubSourceState>> = {}
): Record<HubSourceId, HubSourceState> => ({
  inventory: { status: 'ready', updatedAt: 1_000 },
  metrics: { status: 'ready', updatedAt: 3_000 },
  alerts: { status: 'disabled' },
  backups: { status: 'ready', updatedAt: 2_000 },
  qan: { status: 'ready' },
  realtime: { status: 'ready' },
  operations: { status: 'disabled' },
  dashboards: { status: 'ready' },
  ...overrides,
});

const LIVE: FakeSnapshot = { from: 'live' };

const MESSAGES: HubMessages = {
  name: 'MongoDB',
  unit: { cluster: ['cluster', 'clusters'] },
  empty: { title: 'Start', description: 'Add one.', viewer: 'Ask.' },
  clusters: { tableLabel: 'MongoDB clusters' },
  dashboards: { all: 'All MongoDB dashboards' },
};

const makeConfig = (
  liveSources: Record<HubSourceId, HubSourceState> = sources(),
  fetching = false
) => {
  const refetch = vi.fn();
  const useLiveSnapshot = vi.fn((_ctx: HubLiveContext) => ({
    snapshot: LIVE,
    sources: liveSources,
    refetch,
    fetching,
  }));
  const derive = vi.fn((_snapshot: FakeSnapshot, ctx: HubDeriveContext) =>
    testView({ scenario: ctx.scenario, role: ctx.role })
  );
  const fixture = (now: number, ops: boolean): FakeSnapshot => ({
    from: 'fixture',
    now,
    ops,
  });
  const config: TechnologyHubConfig<FakeSnapshot> = {
    id: 'mongodb',
    navId: 'mongo',
    path: '/mongodb',
    name: 'MongoDB',
    icon: () => null,
    messages: MESSAGES,
    treeColumns: [],
    attentionOrder: [],
    describeDestination: ({ label }) => ({ path: [label] }),
    useLiveSnapshot,
    fixtures: {
      empty: fixture,
      healthy: fixture,
      incomplete: fixture,
      unhealthy: fixture,
    },
    derive,
  };

  return { config, useLiveSnapshot, derive, refetch };
};

const renderTechnologyHub = (
  config: TechnologyHubConfig<FakeSnapshot>,
  {
    path = '/mongodb',
    user = TEST_USER_ADMIN,
    settings = {},
  }: { path?: string; user?: User; settings?: Partial<Settings> } = {}
) =>
  renderHook(
    () => ({
      hub: useTechnologyHub(config),
      location: useLocation(),
      navigationType: useNavigationType(),
    }),
    {
      wrapper: ({ children }: PropsWithChildren) => (
        <TestWrapper
          userContext={{ isLoading: false, user }}
          routerProps={{ initialEntries: [path] }}
        >
          {wrapWithSettings(<>{children}</>, { settings })}
        </TestWrapper>
      ),
    }
  );

const lastDeriveContext = (derive: ReturnType<typeof makeConfig>['derive']) =>
  derive.mock.calls[derive.mock.calls.length - 1][1];

describe('useTechnologyHub', () => {
  it('derives live data for the signed-in admin with Operations on', () => {
    const { config, useLiveSnapshot, derive } = makeConfig();
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?environment=prod&hubState=live&hubOps=live',
      settings: { omEnabled: true, backupManagementEnabled: true },
    });

    expect(useLiveSnapshot).toHaveBeenLastCalledWith({
      enabled: true,
      refreshEvery: 30_000,
      environment: 'prod',
      operations: true,
      role: 'admin',
    });
    expect(derive.mock.calls[0][0]).toBe(LIVE);
    expect(lastDeriveContext(derive)).toMatchObject({
      scenario: 'live',
      environment: 'prod',
      role: 'admin',
      operations: true,
      params: {
        environment: 'prod',
        scenario: 'live',
        ops: 'live',
        role: 'live',
      },
      capabilities: {
        addService: true,
        install: true,
        operations: true,
        setUpBackup: true,
        extensions: false,
      },
    });
    expect(result.current.hub.freshness).toEqual({
      state: 'live',
      at: 3_000,
      interval: '30s',
      refreshing: false,
    });
    expect(result.current.hub.isInitialLoading).toBe(false);
    expect(result.current.hub.access).toEqual({
      admin: true,
      operations: true,
    });
  });

  it('opens on the best-case sample with Operations on', () => {
    const { config, useLiveSnapshot, derive } = makeConfig();

    renderTechnologyHub(config);

    expect(useLiveSnapshot).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: false })
    );
    expect(lastDeriveContext(derive)).toMatchObject({
      scenario: 'healthy',
      operations: true,
    });
  });

  it('never raises a live user above what they have', () => {
    const { config, useLiveSnapshot } = makeConfig();

    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=live&hubOps=on&hubRole=admin',
      user: TEST_USER_VIEWER,
      settings: { omEnabled: true },
    });

    expect(useLiveSnapshot).toHaveBeenLastCalledWith(
      expect.objectContaining({ operations: false, role: 'viewer' })
    );
    expect(result.current.hub.access).toEqual({
      admin: false,
      operations: false,
    });
  });

  it('lets a live admin look as a viewer, without Operations', () => {
    const { config, derive } = makeConfig();

    renderTechnologyHub(config, {
      path: '/mongodb?hubState=live&hubRole=viewer',
      settings: { omEnabled: true },
    });

    expect(lastDeriveContext(derive)).toMatchObject({
      role: 'viewer',
      operations: false,
      capabilities: { addService: false, install: false },
    });
  });

  it('derives sample data with every prototype value honoured, live sources off', () => {
    const { config, useLiveSnapshot, derive } = makeConfig();
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=unhealthy&hubOps=on&hubRole=admin',
      user: TEST_USER_VIEWER,
    });
    const [snapshot, ctx] = derive.mock.calls[derive.mock.calls.length - 1];

    expect(useLiveSnapshot).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: false })
    );
    expect(snapshot).toMatchObject({ from: 'fixture', ops: true });
    expect(ctx).toMatchObject({ scenario: 'unhealthy', role: 'admin' });
    expect(
      Object.values(ctx.sources).every(
        (source: HubSourceState) => source.status === 'ready'
      )
    ).toBe(true);
    expect(result.current.hub.freshness).toEqual({
      state: 'live',
      at: snapshot.now,
      interval: '30s',
      refreshing: false,
    });
  });

  it('builds the sample data once per scenario', () => {
    const { config, derive } = makeConfig();
    const { result, rerender } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=healthy',
    });
    const first = derive.mock.calls[derive.mock.calls.length - 1][0];

    rerender();

    expect(derive.mock.calls[derive.mock.calls.length - 1][0]).toBe(first);

    act(() => result.current.hub.setParams({ scenario: 'unhealthy' }));

    expect(derive.mock.calls[derive.mock.calls.length - 1][0]).not.toBe(first);
  });

  it('refreshes sample data by building it again, and live data through its sources', () => {
    const { config, derive } = makeConfig();
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=unhealthy',
    });
    const first = derive.mock.calls[derive.mock.calls.length - 1][0];

    act(() => result.current.hub.refetch());

    expect(derive.mock.calls[derive.mock.calls.length - 1][0]).not.toBe(first);

    const live = makeConfig();
    const { result: liveHub } = renderTechnologyHub(live.config, {
      path: '/mongodb?hubState=live',
    });

    liveHub.current.hub.refetch();

    expect(live.refetch).toHaveBeenCalledTimes(1);
  });

  it('refreshes sample data at the picked interval, and not with auto-refresh off', () => {
    vi.useFakeTimers();
    try {
      const { config, derive, useLiveSnapshot } = makeConfig();
      const { result } = renderTechnologyHub(config, {
        path: '/mongodb?hubState=unhealthy&refresh=10s',
      });
      const latest = () => derive.mock.calls[derive.mock.calls.length - 1][0];
      const first = latest();

      act(() => vi.advanceTimersByTime(10_000));

      expect(latest()).not.toBe(first);
      expect(result.current.hub.freshness).toEqual({
        state: 'live',
        at: latest().now,
        interval: '10s',
        refreshing: false,
      });
      expect(useLiveSnapshot).toHaveBeenLastCalledWith(
        expect.objectContaining({ refreshEvery: 10_000 })
      );

      act(() => result.current.hub.setParams({ refresh: 'off' }));
      const off = latest();

      act(() => vi.advanceTimersByTime(10 * 60_000));

      // Off never goes stale: the reader chose not to refresh.
      expect(latest()).toBe(off);
      expect(result.current.hub.freshness).toMatchObject({
        state: 'off',
        at: off.now,
        interval: 'off',
      });
      expect(useLiveSnapshot).toHaveBeenLastCalledWith(
        expect.objectContaining({ refreshEvery: false })
      );

      // Refresh still works on demand.
      act(() => result.current.hub.refetch());

      expect(latest()).not.toBe(off);
    } finally {
      vi.useRealTimers();
    }
  });

  it('simulates stale data 4 minutes old, stopping every refresh until Live', () => {
    vi.useFakeTimers();
    try {
      const { config, derive, useLiveSnapshot } = makeConfig();
      const { result } = renderTechnologyHub(config, {
        path: '/mongodb?hubState=unhealthy&refresh=1m',
      });
      const latest = () => derive.mock.calls[derive.mock.calls.length - 1][0];

      act(() => result.current.hub.setParams({ freshness: 'stale' }));
      const stale = latest();

      expect(result.current.hub.freshness).toEqual({
        state: 'stale',
        at: Date.now() - HUB_STALE_FOR_MS,
        interval: '1m',
        refreshing: false,
      });
      expect(useLiveSnapshot).toHaveBeenLastCalledWith(
        expect.objectContaining({ refreshEvery: false })
      );

      act(() => {
        vi.advanceTimersByTime(5 * 60_000);
        result.current.hub.refetch();
      });

      // Retry finds nothing new, as a failed fetch would.
      expect(latest()).toBe(stale);

      // Back with the interval the reader had.
      act(() => result.current.hub.setParams({ freshness: 'live' }));
      act(() => vi.advanceTimersByTime(60_000));

      expect(latest()).not.toBe(stale);
      expect(result.current.hub.freshness).toMatchObject({
        state: 'live',
        interval: '1m',
      });
    } finally {
      vi.useRealTimers();
    }
  });

  it('says a refresh is in flight on live data only', () => {
    const { config } = makeConfig(sources(), true);
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=live',
    });

    expect(result.current.hub.freshness.refreshing).toBe(true);

    const sample = makeConfig(sources(), true);
    const { result: sampleHub } = renderTechnologyHub(sample.config);

    expect(sampleHub.current.hub.freshness.refreshing).toBe(false);
  });

  it('is loading at first until inventory and metrics answer', () => {
    const { config } = makeConfig(
      sources({ metrics: { status: 'loading' }, qan: { status: 'loading' } })
    );
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=live',
    });

    expect(result.current.hub.isInitialLoading).toBe(true);
  });

  it('writes params with replace, keeping other params byte for byte', () => {
    const { config } = makeConfig();
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?from=now-1h&q=a%20b&hubState=unhealthy#clusters',
    });

    act(() => result.current.hub.setParams({ environment: 'prod' }));

    expect(result.current.location.search).toBe(
      '?from=now-1h&q=a%20b&environment=prod&hubState=unhealthy'
    );
    expect(result.current.location.hash).toBe('');
    expect(result.current.hub.params.environment).toBe('prod');
    // Back leaves the page instead of stepping through each filter.
    expect(result.current.navigationType).toBe('REPLACE');

    act(() =>
      result.current.hub.setParams({ environment: null, scenario: 'healthy' })
    );

    expect(result.current.location.search).toBe('?from=now-1h&q=a%20b');
  });
});

describe('liveFreshness', () => {
  const NOW = 10 * 60_000;

  it('reads live from the newest update while every source keeps up', () => {
    expect(
      liveFreshness(
        sources({
          inventory: {
            status: 'ready',
            updatedAt: NOW - 50_000,
            refreshEvery: 30_000,
          },
          metrics: {
            status: 'ready',
            updatedAt: NOW - 20_000,
            refreshEvery: 30_000,
          },
        }),
        NOW,
        '30s'
      )
    ).toEqual({ state: 'live', at: NOW - 20_000 });
  });

  it('reads stale from the oldest source that missed two refreshes', () => {
    const late = sources({
      inventory: {
        status: 'ready',
        updatedAt: NOW - 3 * 60_000,
        refreshEvery: 60_000,
      },
      metrics: {
        status: 'ready',
        updatedAt: NOW - 61_000,
        refreshEvery: 30_000,
      },
      backups: { status: 'ready', updatedAt: NOW - 5_000 },
    });

    expect(liveFreshness(late, NOW, '30s')).toEqual({
      state: 'stale',
      at: NOW - 3 * 60_000,
    });
  });

  it('reads off with auto-refresh off, however old the data', () => {
    expect(
      liveFreshness(
        sources({
          inventory: {
            status: 'ready',
            updatedAt: NOW - 9 * 60_000,
            refreshEvery: 30_000,
          },
        }),
        NOW,
        'off'
      )
    ).toEqual({ state: 'off', at: NOW - 9 * 60_000 });
  });

  it('ignores sources that refresh only on demand, are off or failed outright', () => {
    expect(
      liveFreshness(
        sources({
          inventory: { status: 'ready', updatedAt: NOW - 9 * 60_000 },
          metrics: { status: 'error' },
          backups: { status: 'disabled' },
        }),
        NOW,
        '30s'
      ).state
    ).toBe('live');
  });
});
