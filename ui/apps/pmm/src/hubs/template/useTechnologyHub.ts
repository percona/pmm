import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useSettings } from 'contexts/settings';
import { useUser } from 'contexts/user';
import {
  parseHubParams,
  resolveHubCapabilities,
  resolveHubOperations,
  resolveHubRole,
  writeHubSearch,
} from './hub.params';
import type {
  HubFreshness,
  HubParams,
  HubSourceId,
  HubSourceState,
  HubViewModel,
  TechnologyHubConfig,
} from './hub.types';

/** How often the derived view's clock moves, so ages and thresholds stay current between fetches. */
export const HUB_NOW_TICK_MS = 30_000;

/** Sample data refreshes as often as live metrics do, so its ages read like a live page's. */
export const HUB_SAMPLE_REFRESH_MS = 30_000;

/** A source reads stale once it misses two refreshes, and never sooner than this. */
export const HUB_STALE_MIN_MS = 2 * 60_000;

/** How old the sample data reads with the prototype's connection lost. */
export const HUB_LOST_FOR_MS = 32 * 60_000;

const isOverdue = (source: HubSourceState, now: number) =>
  source.status === 'ready' &&
  source.updatedAt !== undefined &&
  source.refreshEvery !== undefined &&
  now - source.updatedAt > Math.max(2 * source.refreshEvery, HUB_STALE_MIN_MS);

/** The newest update while every source keeps up; the oldest overdue one, and stale, once any falls behind. */
export const liveFreshness = (
  sources: Record<HubSourceId, HubSourceState>,
  now: number
): HubFreshness => {
  const states = Object.values(sources);
  const overdue = states
    .filter((source) => isOverdue(source, now))
    .map((source) => source.updatedAt as number);

  if (overdue.length) {
    return { at: Math.min(...overdue), stale: true };
  }
  const updated = states.flatMap((source) =>
    source.updatedAt ? [source.updatedAt] : []
  );

  return { at: updated.length ? Math.max(...updated) : now, stale: false };
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
  const connectionLost = !live && params.connection === 'lost';

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
    environment: params.environment,
    operations,
    role,
  });

  // Refresh on sample data builds it again, so its ages restart like a fresh fetch; a lost connection stops it.
  const [fixtureRun, setFixtureRun] = useState(0);

  useEffect(() => {
    if (live || connectionLost) {
      return;
    }
    const id = setInterval(
      () => setFixtureRun((run) => run + 1),
      HUB_SAMPLE_REFRESH_MS
    );

    return () => clearInterval(id);
  }, [live, connectionLost]);
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

  const freshness = useMemo<HubFreshness>(() => {
    if (fixture) {
      return connectionLost
        ? { at: fixture.now - HUB_LOST_FOR_MS, stale: true }
        : { at: fixture.now, stale: false };
    }

    return liveFreshness(sources, now);
  }, [fixture, connectionLost, sources, now]);

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

  // With the connection lost, Refresh finds nothing new, as a failed fetch would.
  const refreshFixture = useCallback(() => {
    if (!connectionLost) {
      setFixtureRun((run) => run + 1);
    }
  }, [connectionLost]);

  return {
    view,
    params,
    setParams,
    refetch: live ? liveResult.refetch : refreshFixture,
    freshness,
    isInitialLoading,
    access,
  };
};
