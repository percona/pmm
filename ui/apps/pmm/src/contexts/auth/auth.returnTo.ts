import { GRAFANA_DIRECT_PATH_PATTERN, isRenderingServer } from '@pmm/shared';
import { safeSessionStorage } from 'utils/storage.utils';
import {
  AUTH_RETURN_TO_LOOP_WINDOW_MS,
  AUTH_RETURN_TO_TTL_MS,
  GRAFANA_HOME_PATHS,
  PMM_BASE_PATH,
} from 'lib/constants';

const RETURN_TO_KEY = 'pmm-ui.auth.returnTo';
const RETURN_TO_LAST_KEY = 'pmm-ui.auth.returnTo.last';

/**
 * Grafana's own post-login target. The shell owns post-login navigation, so this is always
 * dropped: a leftover value makes the iframe navigate itself and desync the browser URL.
 */
const GRAFANA_REDIRECT_TO_KEY = 'redirectTo';

/** Timestamped: an untimed marker outlives its restore and drops the deep link on the next login. */
type StoredPath = {
  path: string;
  at: number;
};

const NOTHING_TO_RESTORE = ['/', ...GRAFANA_HOME_PATHS];

/** Targets are shell-relative (no /pmm-ui), ready to hand to navigate() under that basename. */
export const isRestorableReturnTo = (target: string) => {
  // Raw, before any normalisation: collapsing slashes first would launder //evil.com into
  // /evil.com.
  if (!target.startsWith('/') || target.startsWith('//')) {
    return false;
  }

  // Path only: a Grafana template variable may legitimately put '..' or a backslash in the query
  // (?var-version=1..2, ?var-node=/^web\d+$/).
  const rawPath = target.split(/[?#]/)[0];

  // Still raw - browsers fold /\evil.com into //evil.com.
  if (rawPath.includes('\\')) {
    return false;
  }

  let decoded: string;

  try {
    decoded = decodeURIComponent(rawPath);
  } catch {
    // A malformed escape has no single meaning, so there is nothing safe to classify.
    return false;
  }

  // /%2F%2Fevil.com and /%5Cevil.com only show their shape once decoded.
  if (decoded.startsWith('//') || decoded.includes('\\')) {
    return false;
  }

  // What nginx sees. Collapsing any earlier would have hidden the '//' checked above.
  const path = decoded.replace(/\/{2,}/g, '/');

  if (path.includes('..') || NOTHING_TO_RESTORE.includes(path)) {
    return false;
  }

  // Never store a target that bounces back out of the shell. The pattern, not
  // isGrafanaDirectPath(): `path` is already normalised, and decoding twice would judge
  // /graph/%2561pi more strictly than nginx, which decodes $uri once.
  return !GRAFANA_DIRECT_PATH_PATTERN.test(path);
};

const dropGrafanaReturnTo = () => {
  safeSessionStorage.removeItem(GRAFANA_REDIRECT_TO_KEY);
};

const toShellRelative = ({
  pathname,
  search,
  hash,
}: Pick<Location, 'pathname' | 'search' | 'hash'>) => {
  // slice, not replace: a later "/pmm-ui" inside the path must not be eaten.
  const relative = pathname.startsWith(PMM_BASE_PATH)
    ? pathname.slice(PMM_BASE_PATH.length) || '/'
    : pathname;

  return relative + search + hash;
};

const readStoredPath = (key: string): StoredPath | null => {
  const raw = safeSessionStorage.getItem(key);

  if (!raw) {
    return null;
  }

  try {
    const stored = JSON.parse(raw) as StoredPath;

    if (typeof stored?.path !== 'string' || typeof stored?.at !== 'number') {
      return null;
    }

    return stored;
  } catch {
    return null;
  }
};

const writeStoredPath = (key: string, path: string) => {
  const stored: StoredPath = { path, at: Date.now() };

  safeSessionStorage.setItem(key, JSON.stringify(stored));
};

/**
 * Remember where the user was before bouncing them to Grafana's login page. sessionStorage is
 * per-tab and survives the cross-document hop, and stays clear of the localStorage keys
 * ensureClientSessionListener() watches.
 */
export const saveReturnTo = (
  location: Pick<Location, 'pathname' | 'search' | 'hash'> = window.location
) => {
  dropGrafanaReturnTo();

  // The image renderer never runs the shell; mirrors the nginx $arg_render exclusion.
  if (isRenderingServer()) {
    return;
  }

  const target = toShellRelative(location);

  if (!isRestorableReturnTo(target)) {
    return;
  }

  // Bounce guard: don't retry a target we just restored and immediately bounced off. Windowed, so
  // a session that expires hours later on that page is still remembered. The marker is left in
  // place, not consumed - consuming it made this non-idempotent, and AuthProvider calls it once
  // per render.
  const marker = readStoredPath(RETURN_TO_LAST_KEY);

  if (
    marker?.path === target &&
    Date.now() - marker.at <= AUTH_RETURN_TO_LOOP_WINDOW_MS
  ) {
    return;
  }

  writeStoredPath(RETURN_TO_KEY, target);
};

export const clearReturnTo = () => {
  dropGrafanaReturnTo();
  safeSessionStorage.removeItem(RETURN_TO_KEY);
  safeSessionStorage.removeItem(RETURN_TO_LAST_KEY);
};

/** Read and discard the pending target, so it can only ever be acted on once. */
export const consumeReturnTo = (): string | null => {
  dropGrafanaReturnTo();

  const stored = readStoredPath(RETURN_TO_KEY);

  // Cleared up front so no early return below can leave a stale marker behind.
  safeSessionStorage.removeItem(RETURN_TO_KEY);
  safeSessionStorage.removeItem(RETURN_TO_LAST_KEY);

  if (
    !stored ||
    !isRestorableReturnTo(stored.path) ||
    Date.now() - stored.at > AUTH_RETURN_TO_TTL_MS
  ) {
    return null;
  }

  writeStoredPath(RETURN_TO_LAST_KEY, stored.path);

  return stored.path;
};
