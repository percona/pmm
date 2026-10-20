import {
  HUB_NO_ENVIRONMENT,
  type HubCapabilities,
  type HubOpsMode,
  type HubParams,
  type HubRole,
  type HubRoleMode,
  type HubScenario,
} from './hub.types';

/** URL names of the hub's own params; the three prototype ones are removed before anything ships. */
export const HUB_PARAM_KEYS = {
  environment: 'environment',
  patches: 'patches',
  scenario: 'hubState',
  ops: 'hubOps',
  role: 'hubRole',
} as const satisfies Record<keyof HubParams, string>;

export const DEFAULT_HUB_PARAMS: HubParams = {
  environment: null,
  patches: null,
  scenario: 'live',
  ops: 'live',
  role: 'live',
};

const SCENARIOS: readonly HubScenario[] = ['live', 'empty', 'healthy', 'wrong'];
const OPS_MODES: readonly HubOpsMode[] = ['live', 'on', 'off'];
const ROLE_MODES: readonly HubRoleMode[] = ['live', 'admin', 'viewer'];

const oneOf = <T extends string>(
  values: readonly T[],
  value: string | null,
  fallback: T
): T => (values.includes(value as T) ? (value as T) : fallback);

/** Reads the hub params; unknown values fall back to live, an empty environment to all, any patches value but behind to none. */
export const parseHubParams = (search: URLSearchParams | string): HubParams => {
  const params =
    typeof search === 'string' ? new URLSearchParams(search) : search;

  return {
    environment: params.get(HUB_PARAM_KEYS.environment) || null,
    patches: params.get(HUB_PARAM_KEYS.patches) === 'behind' ? 'behind' : null,
    scenario: oneOf(SCENARIOS, params.get(HUB_PARAM_KEYS.scenario), 'live'),
    ops: oneOf(OPS_MODES, params.get(HUB_PARAM_KEYS.ops), 'live'),
    role: oneOf(ROLE_MODES, params.get(HUB_PARAM_KEYS.role), 'live'),
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
  if (merged.patches) {
    own.set(HUB_PARAM_KEYS.patches, merged.patches);
  }
  (['scenario', 'ops', 'role'] as const).forEach((key) => {
    if (merged[key] !== DEFAULT_HUB_PARAMS[key]) {
      own.set(HUB_PARAM_KEYS[key], merged[key]);
    }
  });

  const ownString = own.toString();

  return [...kept, ...(ownString ? [ownString] : [])].join('&');
};

/** Only the hub params, in a stable order, for doors back into the hub. */
export const serializeHubParams = (params: HubParams): string =>
  writeHubSearch('', params);

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
