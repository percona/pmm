import {
  HUB_NO_ENVIRONMENT,
  type HubCapabilities,
  type HubFreshnessMode,
  type HubOpsMode,
  type HubParams,
  type HubRefresh,
  type HubRole,
  type HubRoleMode,
  type HubScenario,
} from './hub.types';

/** URL names of the hub's own params; the `hub*` ones are prototype only and removed before anything ships. */
export const HUB_PARAM_KEYS = {
  environment: 'environment',
  // Grafana's name for it, so a bookmark reopens with the same interval.
  refresh: 'refresh',
  scenario: 'hubState',
  ops: 'hubOps',
  role: 'hubRole',
  freshness: 'hubFreshness',
} as const satisfies Record<keyof HubParams, string>;

export const DEFAULT_HUB_PARAMS: HubParams = {
  environment: null,
  // The prototype opens on the best case, with Operations, so the bare URL shows the whole page; Live data is a choice.
  scenario: 'healthy',
  ops: 'on',
  role: 'live',
  refresh: '30s',
  freshness: 'live',
};

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;

const REFRESH_MS: Record<Exclude<HubRefresh, 'off'>, number> = {
  '5s': 5 * SECOND,
  '10s': 10 * SECOND,
  '30s': 30 * SECOND,
  '1m': MINUTE,
  '5m': 5 * MINUTE,
  '15m': 15 * MINUTE,
  '30m': 30 * MINUTE,
  '1h': HOUR,
  '2h': 2 * HOUR,
  '1d': 24 * HOUR,
};

/** The auto-refresh choices, in the selector's order. */
export const HUB_REFRESHES: readonly HubRefresh[] = [
  'off',
  ...(Object.keys(REFRESH_MS) as Array<Exclude<HubRefresh, 'off'>>),
];

/** An auto-refresh choice in ms; false for off. */
export const refreshInterval = (refresh: HubRefresh): number | false =>
  refresh === 'off' ? false : REFRESH_MS[refresh];

const SCENARIOS: readonly HubScenario[] = [
  'live',
  'empty',
  'healthy',
  'incomplete',
  'unhealthy',
];
const OPS_MODES: readonly HubOpsMode[] = ['live', 'on', 'off'];
const ROLE_MODES: readonly HubRoleMode[] = ['live', 'admin', 'viewer'];
const FRESHNESS_MODES: readonly HubFreshnessMode[] = ['live', 'stale'];

const oneOf = <T extends string>(
  values: readonly T[],
  value: string | null,
  fallback: T
): T => (values.includes(value as T) ? (value as T) : fallback);

/** Reads the hub params; unknown values fall back to the defaults, an empty environment to all. */
export const parseHubParams = (search: URLSearchParams | string): HubParams => {
  const params =
    typeof search === 'string' ? new URLSearchParams(search) : search;

  return {
    environment: params.get(HUB_PARAM_KEYS.environment) || null,
    scenario: oneOf(
      SCENARIOS,
      params.get(HUB_PARAM_KEYS.scenario),
      DEFAULT_HUB_PARAMS.scenario
    ),
    ops: oneOf(
      OPS_MODES,
      params.get(HUB_PARAM_KEYS.ops),
      DEFAULT_HUB_PARAMS.ops
    ),
    role: oneOf(
      ROLE_MODES,
      params.get(HUB_PARAM_KEYS.role),
      DEFAULT_HUB_PARAMS.role
    ),
    refresh: oneOf(
      HUB_REFRESHES,
      params.get(HUB_PARAM_KEYS.refresh),
      DEFAULT_HUB_PARAMS.refresh
    ),
    freshness: oneOf(
      FRESHNESS_MODES,
      params.get(HUB_PARAM_KEYS.freshness),
      DEFAULT_HUB_PARAMS.freshness
    ),
  };
};

const HUB_KEYS = new Set<string>(Object.values(HUB_PARAM_KEYS));

const keyOf = (pair: string) => {
  const name = pair.split('=', 1)[0];

  try {
    return decodeURIComponent(name.replace(/\+/g, ' '));
  } catch {
    return name;
  }
};

/** The query string with `next` applied, defaults left out and other params byte for byte. */
export const writeHubSearch = (
  search: string,
  next: Partial<HubParams>
): string => {
  const merged = { ...parseHubParams(search), ...next };
  const kept = search
    .replace(/^\?/, '')
    .split('&')
    .filter((pair) => pair && !HUB_KEYS.has(keyOf(pair)));
  const own = new URLSearchParams();

  if (merged.environment) {
    own.set(HUB_PARAM_KEYS.environment, merged.environment);
  }
  (['refresh', 'scenario', 'ops', 'role', 'freshness'] as const).forEach(
    (key) => {
      if (merged[key] !== DEFAULT_HUB_PARAMS[key]) {
        own.set(HUB_PARAM_KEYS[key], merged[key]);
      }
    }
  );

  const ownString = own.toString();

  return [...kept, ...(ownString ? [ownString] : [])].join('&');
};

/** Signed-in role, which `hubRole=viewer` can lower; `admin` raises it only for sample data. */
export const resolveHubRole = (
  params: Pick<HubParams, 'scenario' | 'role'>,
  isPMMAdmin: boolean
): HubRole => {
  const live: HubRole = isPMMAdmin ? 'admin' : 'viewer';

  if (params.role === 'viewer') {
    return 'viewer';
  }
  if (params.role === 'admin' && params.scenario !== 'live') {
    return 'admin';
  }

  return live;
};

/** Operations for an admin who has it on; `hubOps=off` lowers it, `on` raises it only for sample data. */
export const resolveHubOperations = (
  params: Pick<HubParams, 'scenario' | 'ops'>,
  role: HubRole,
  enabledForUser: boolean
): boolean => {
  if (role !== 'admin' || params.ops === 'off') {
    return false;
  }
  if (params.ops === 'on' && params.scenario !== 'live') {
    return true;
  }

  return enabledForUser;
};

export interface HubAccessInput {
  role: HubRole;
  operations: boolean;
  backupManagementEnabled: boolean;
  extensionsEnabled: boolean;
  signedIn: boolean;
}

export const resolveHubCapabilities = ({
  role,
  operations,
  backupManagementEnabled,
  extensionsEnabled,
  signedIn,
}: HubAccessInput): HubCapabilities => ({
  addService: role === 'admin',
  install: role === 'admin' && operations,
  operations,
  setUpBackup: role === 'admin' && backupManagementEnabled,
  extensions: signedIn && extensionsEnabled,
});

/** True when a service's environment passes the filter; null passes everything. */
export const matchesEnvironment = (
  environment: string | undefined | null,
  filter: string | null
): boolean => {
  if (filter === null) {
    return true;
  }
  if (filter === HUB_NO_ENVIRONMENT) {
    return !environment;
  }

  return environment === filter;
};

/** Each environment alphabetically, then "No environment" when a service has none, keeping a selected one no service has; the select adds "All environments" itself. */
export const environmentOptions = (
  environments: Array<string | undefined | null>,
  noEnvironmentLabel: string,
  selected: string | null = null
): Array<{ value: string; label: string }> => {
  const named = [
    ...new Set(
      [
        ...environments,
        selected === HUB_NO_ENVIRONMENT ? null : selected,
      ].filter((env): env is string => !!env)
    ),
  ].sort((a, b) => a.localeCompare(b));
  const options = named.map((value) => ({ value, label: value }));

  return environments.some((env) => !env) || selected === HUB_NO_ENVIRONMENT
    ? [...options, { value: HUB_NO_ENVIRONMENT, label: noEnvironmentLabel }]
    : options;
};
