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
  HubParams,
  HubSourceId,
  HubSourceState,
  HubViewModel,
  TechnologyHubConfig,
} from './hub.types';

/** How often the derived view's clock moves, so ages and thresholds stay current between fetches. */
export const HUB_NOW_TICK_MS = 30_000;

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
  /** Newest source update, or the sample data's clock. */
  updatedAt: number;
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

  // Refresh on sample data builds it again, so its ages restart like a fresh fetch.
  const [fixtureRun, setFixtureRun] = useState(0);
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

  const updatedAt = useMemo(() => {
    if (fixture) {
      return fixture.now;
    }
    const updated = Object.values(sources).flatMap((source) =>
      source.updatedAt ? [source.updatedAt] : []
    );

    return updated.length ? Math.max(...updated) : now;
  }, [fixture, sources, now]);

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

  return {
    view,
    params,
    setParams,
    refetch: live ? liveResult.refetch : refreshFixture,
    updatedAt,
    isInitialLoading,
    access,
  };
};
