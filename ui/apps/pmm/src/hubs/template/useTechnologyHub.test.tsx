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
  HUB_LOST_FOR_MS,
  HUB_SAMPLE_REFRESH_MS,
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
  backups: {
    emptyTitle: 'None',
    emptyDescription: 'Set up.',
    noServices: 'None.',
  },
  dashboards: { all: 'All MongoDB dashboards' },
};

const makeConfig = (
  liveSources: Record<HubSourceId, HubSourceState> = sources()
) => {
  const refetch = vi.fn();
  const useLiveSnapshot = vi.fn((_ctx: HubLiveContext) => ({
    snapshot: LIVE,
    sources: liveSources,
    refetch,
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
    useLiveSnapshot,
    fixtures: { empty: fixture, healthy: fixture, wrong: fixture },
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
    expect(result.current.hub.freshness).toEqual({ at: 3_000, stale: false });
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
      path: '/mongodb?hubState=wrong&hubOps=on&hubRole=admin',
      user: TEST_USER_VIEWER,
    });
    const [snapshot, ctx] = derive.mock.calls[derive.mock.calls.length - 1];

    expect(useLiveSnapshot).toHaveBeenLastCalledWith(
      expect.objectContaining({ enabled: false })
    );
    expect(snapshot).toMatchObject({ from: 'fixture', ops: true });
    expect(ctx).toMatchObject({ scenario: 'wrong', role: 'admin' });
    expect(
      Object.values(ctx.sources).every(
        (source: HubSourceState) => source.status === 'ready'
      )
    ).toBe(true);
    expect(result.current.hub.freshness).toEqual({
      at: snapshot.now,
      stale: false,
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

    act(() => result.current.hub.setParams({ scenario: 'wrong' }));

    expect(derive.mock.calls[derive.mock.calls.length - 1][0]).not.toBe(first);
  });

  it('refreshes sample data by building it again, and live data through its sources', () => {
    const { config, derive } = makeConfig();
    const { result } = renderTechnologyHub(config, {
      path: '/mongodb?hubState=wrong',
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

  it('refreshes sample data on its own like live data, and stops with the connection lost', () => {
    vi.useFakeTimers();
    try {
      const { config, derive } = makeConfig();
      const { result } = renderTechnologyHub(config, {
        path: '/mongodb?hubState=wrong',
      });
      const first = derive.mock.calls[derive.mock.calls.length - 1][0];

      act(() => vi.advanceTimersByTime(HUB_SAMPLE_REFRESH_MS));

      const second = derive.mock.calls[derive.mock.calls.length - 1][0];

      expect(second).not.toBe(first);
      expect(result.current.hub.freshness.stale).toBe(false);

      act(() => result.current.hub.setParams({ connection: 'lost' }));

      const lost = derive.mock.calls[derive.mock.calls.length - 1][0];

      expect(result.current.hub.freshness).toEqual({
        at: (lost.now ?? 0) - HUB_LOST_FOR_MS,
        stale: true,
      });

      act(() => {
        vi.advanceTimersByTime(HUB_SAMPLE_REFRESH_MS * 2);
        result.current.hub.refetch();
      });

      expect(derive.mock.calls[derive.mock.calls.length - 1][0]).toBe(lost);
    } finally {
      vi.useRealTimers();
    }
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
      path: '/mongodb?from=now-1h&q=a%20b&hubState=wrong#clusters',
    });

    act(() => result.current.hub.setParams({ environment: 'prod' }));

    expect(result.current.location.search).toBe(
      '?from=now-1h&q=a%20b&environment=prod&hubState=wrong'
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

  it('reads the newest update while every source keeps up', () => {
    expect(
      liveFreshness(
        sources({
          inventory: {
            status: 'ready',
            updatedAt: NOW - 50_000,
            refreshEvery: 60_000,
          },
          metrics: {
            status: 'ready',
            updatedAt: NOW - 20_000,
            refreshEvery: 30_000,
          },
        }),
        NOW
      )
    ).toEqual({ at: NOW - 20_000, stale: false });
  });

  it('reads stale from the oldest source that missed two refreshes, and not before two minutes', () => {
    const late = sources({
      inventory: {
        status: 'ready',
        updatedAt: NOW - 3 * 60_000,
        refreshEvery: 60_000,
      },
      metrics: {
        status: 'ready',
        updatedAt: NOW - 90_000,
        refreshEvery: 30_000,
      },
      backups: { status: 'ready', updatedAt: NOW - 5_000 },
    });

    // Metrics missed three refreshes but is not two minutes old; inventory is.
    expect(liveFreshness(late, NOW)).toEqual({
      at: NOW - 3 * 60_000,
      stale: true,
    });
  });

  it('ignores sources that refresh only on demand, are off or failed outright', () => {
    expect(
      liveFreshness(
        sources({
          inventory: { status: 'ready', updatedAt: NOW - 9 * 60_000 },
          metrics: { status: 'error' },
          backups: { status: 'disabled' },
        }),
        NOW
      ).stale
    ).toBe(false);
  });
});
