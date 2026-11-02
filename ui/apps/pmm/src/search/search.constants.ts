import type { SearchGroupId } from './search.types';

// Groups in the order a query shows them, with the most items each one lists.
export const GROUP_CAPS: [Exclude<SearchGroupId, 'recent'>, number][] = [
  ['pages', 6],
  ['settings', 4],
  ['actions', 3],
  ['dashboards', 6],
  ['inventory', 5],
  ['help', 3],
];

// Groups that move first, best first, when their best match is a title prefix or better and beats every page.
export const PROMOTABLE_GROUPS: SearchGroupId[] = [
  'settings',
  'actions',
  'help',
];

export const MAX_QUERY_TOKENS = 6;

export const REMOTE_MIN_QUERY_LENGTH = 2;

export const RECENT_PREFIX = 'pmm-ui.search.recent';

export const RECENT_STORED = 20;

export const RECENT_SHOWN = 5;

export const ZERO_QUERY_ACTIONS = 3;

export const DASHBOARDS_QUERY_KEY = ['search', 'dashboards'];

export const NODES_QUERY_KEY = ['search', 'nodes'];

export const REMOTE_STALE_TIME = 300_000;

export const INVENTORY_STALE_TIME = 60_000;

export const VARIABLES_TIMEOUT_MS = 300;

export const GRAFANA_IFRAME_SELECTOR = '#grafana-iframe';

export const SEARCH_DIALOG_TEST_ID = 'pmm-search-dialog';

export const SEARCH_INPUT_TEST_ID = 'pmm-search-input';

export const SIDEBAR_TRIGGER_TEST_ID = 'sidebar-search-trigger';
