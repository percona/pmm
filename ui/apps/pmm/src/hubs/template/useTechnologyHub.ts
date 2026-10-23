import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';
import {
  parseHubParams,
  refreshInterval,
  resolveHubCapabilities,
  resolveHubOperations,
  resolveHubRole,
  writeHubSearch,
} from './hub.params';
import type {
  HubFreshness,
  HubParams,
  HubRefresh,
  HubSourceId,
  HubSourceState,
  HubViewModel,
  TechnologyHubConfig,
} from './hub.types';

/** How often the derived view's clock moves, so ages and thresholds stay current between fetches. */
export const HUB_NOW_TICK_MS = 30_000;

/** How old the data reads when the prototype shows it stale. */
export const HUB_STALE_FOR_MS = 4 * 60_000;

// A source reads stale once it misses two refreshes.
const isOverdue = (source: HubSourceState, now: number) =>
  source.status === 'ready' &&
  source.updatedAt !== undefined &&
  source.refreshEvery !== undefined &&
  now - source.updatedAt > 2 * source.refreshEvery;

/** Off with auto-refresh off, which never goes stale; otherwise stale from the oldest overdue source, or live from the newest update. */
export const liveFreshness = (
  sources: Record<HubSourceId, HubSourceState>,
  now: number,
  interval: HubRefresh
): Pick<HubFreshness, 'state' | 'at'> => {
  const states = Object.values(sources).filter(
    (source) => source.status === 'ready' || source.status === 'error'
  );
  const updated = states.flatMap((source) =>
    source.updatedAt ? [source.updatedAt] : []
  );
  const at = updated.length ? Math.max(...updated) : null;

  if (interval === 'off') {
    return { state: 'off', at };
  }
  const overdue = states
    .filter((source) => isOverdue(source, now))
    .map((source) => source.updatedAt as number);

  return overdue.length
    ? { state: 'stale', at: Math.min(...overdue) }
    : { state: 'live', at };
};

const SOURCE_IDS: HubSourceId[] = [
  'inventory',
  'metrics',
  'alerts',
  'backups',
  'qan',
  'realtime',
  'operations',
  'dashboards',
];

const FIXTURE_SOURCES = Object.fromEntries(
  SOURCE_IDS.map((id) => [id, { status: 'ready' }])
) as Record<HubSourceId, HubSourceState>;

// The sources the status line and the tree need before they can say anything.
const INITIAL_SOURCES: HubSourceId[] = ['inventory', 'metrics'];

const useNow = (intervalMs: number) => {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), intervalMs);

    return () => clearInterval(id);
  }, [intervalMs]);

  return now;
};

export interface TechnologyHub {
  view: HubViewModel;
  params: HubParams;
  /** Writes the params with `replace`, keeping every other param as it is. */
  setParams: (next: Partial<HubParams>) => void;
  refetch: () => void;
  /** How current the data is, live or sample. */
  freshness: HubFreshness;
  isInitialLoading: boolean;
  /** What the signed-in user has, which the prototype switches lower but never raise on live data. */
  access: { admin: boolean; operations: boolean };
}

/** A hub's view from the URL, the signed-in user and either live sources or sample data. */
export const useTechnologyHub = <TSnapshot>(
  config: TechnologyHubConfig<TSnapshot>
): TechnologyHub => {
  const { search } = useLocation();
  const navigate = useNavigate();
  const { user } = useUser();
  const { settings } = useSettings();
  const params = useMemo(() => parseHubParams(search), [search]);
  const { scenario } = params;
  const live = scenario === 'live';
  // Simulated stale data stops the real refresh, live or sample.
  const simulatedStale = params.freshness === 'stale';
  const refreshEvery = simulatedStale ? false : refreshInterval(params.refresh);

  const isPMMAdmin = !!user?.isPMMAdmin;
  const operationsForUser = isPMMAdmin && settings?.omEnabled === true;
  const role = resolveHubRole(params, isPMMAdmin);
  const operations = resolveHubOperations(params, role, operationsForUser);
  const access = useMemo(
    () => ({ admin: isPMMAdmin, operations: operationsForUser }),
    [isPMMAdmin, operationsForUser]
  );
  const backupManagementEnabled = settings?.backupManagementEnabled === true;
  const extensionsEnabled = settings?.extensionsEnabled === true;
  const signedIn = !!user && !user.isAnonymous;
  const capabilities = useMemo(
    () =>
      resolveHubCapabilities({
        role,
        operations,
        backupManagementEnabled,
        extensionsEnabled,
        signedIn,
      }),
    [role, operations, backupManagementEnabled, extensionsEnabled, signedIn]
  );

  const liveResult = config.useLiveSnapshot({
    enabled: live,
    refreshEvery,
    environment: params.environment,
    operations,
    role,
  });

  // Refresh on sample data builds it again, so its ages restart like a fresh fetch, at the picked interval.
  const [fixtureRun, setFixtureRun] = useState(0);

  useEffect(() => {
    if (live || refreshEvery === false) {
      return;
    }
    const id = setInterval(() => setFixtureRun((run) => run + 1), refreshEvery);

    return () => clearInterval(id);
  }, [live, refreshEvery]);
  const fixture = useMemo(() => {
    if (scenario === 'live') {
      return null;
    }
    const now = Date.now();

    return {
      run: fixtureRun,
      now,
      snapshot: config.fixtures[scenario](now, operations, params.environment),
    };
  }, [config, scenario, operations, params.environment, fixtureRun]);

  const tick = useNow(HUB_NOW_TICK_MS);
  const now = fixture ? Math.max(tick, fixture.now) : tick;
  const snapshot = fixture ? fixture.snapshot : liveResult.snapshot;
  const sources = fixture ? FIXTURE_SOURCES : liveResult.sources;

  const view = useMemo(
    () =>
      config.derive(snapshot, {
        now,
        params,
        scenario,
        environment: params.environment,
        role,
        operations,
        capabilities,
        sources,
      }),
    [
      config,
      snapshot,
      now,
      params,
      scenario,
      role,
      operations,
      capabilities,
      sources,
    ]
  );

  // When the simulated failure began, so a stale page's age keeps growing from there.
  const simulatedFrom = useMemo(
    () => (params.freshness === 'stale' ? Date.now() : null),
    [params.freshness]
  );
  const fetching = live && liveResult.fetching;
  const freshness = useMemo<HubFreshness>(() => {
    const interval = params.refresh;

    if (simulatedStale) {
      return {
        state: 'stale',
        at: (simulatedFrom ?? now) - HUB_STALE_FOR_MS,
        interval,
        refreshing: false,
      };
    }
    const current = fixture
      ? {
          state: interval === 'off' ? ('off' as const) : ('live' as const),
          at: fixture.now,
        }
      : liveFreshness(sources, now, interval);

    return { ...current, interval, refreshing: fetching };
  }, [
    fixture,
    params.refresh,
    simulatedStale,
    simulatedFrom,
    sources,
    now,
    fetching,
  ]);

  const isInitialLoading =
    live && INITIAL_SOURCES.some((id) => sources[id].status === 'loading');

  // Through navigate, not setSearchParams, so the other params keep their exact bytes.
  const setParams = useCallback(
    (next: Partial<HubParams>) => {
      const nextSearch = writeHubSearch(search, next);

      navigate(
        { search: nextSearch ? `?${nextSearch}` : '' },
        { replace: true }
      );
    },
    [navigate, search]
  );

  const refreshFixture = useCallback(() => setFixtureRun((run) => run + 1), []);
  // Simulated stale data finds nothing new on Retry, as a failed fetch would.
  const noRefresh = useCallback(() => undefined, []);

  return {
    view,
    params,
    setParams,
    refetch: simulatedStale
      ? noRefresh
      : live
        ? liveResult.refetch
        : refreshFixture,
    freshness,
    isInitialLoading,
    access,
  };
};
