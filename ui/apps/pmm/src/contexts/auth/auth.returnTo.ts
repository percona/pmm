import { isGrafanaDirectPath, isRenderingServer } from '@pmm/shared';
import {
  AUTH_RETURN_TO_TTL_MS,
  GRAFANA_SUB_PATH,
  PMM_BASE_PATH,
} from 'lib/constants';

const RETURN_TO_KEY = 'pmm-ui.auth.returnTo';
const RETURN_TO_LAST_KEY = 'pmm-ui.auth.returnTo.last';

/**
 * Grafana's own post-login target, stashed by its app init when it sees ?redirectTo= or when its
 * token rotation fails inside our iframe. The shell owns post-login navigation, so we always drop
 * it: a leftover value makes the iframe navigate itself after login and desync the browser URL.
 */
const GRAFANA_REDIRECT_TO_KEY = 'redirectTo';

type StoredReturnTo = {
  path: string;
  at: number;
};

const NOTHING_TO_RESTORE = ['/', GRAFANA_SUB_PATH, `${GRAFANA_SUB_PATH}/`];

/**
 * Targets are shell-relative (no /pmm-ui), so they can be handed straight to react-router's
 * navigate() under basename /pmm-ui. Both Grafana routes (/graph/d/...) and native PMM UI pages
 * (/settings/advanced, /help) qualify.
 */
export const isRestorableReturnTo = (target: string) => {
  // Exactly one leading slash keeps this a same-origin path: rejects //evil.com and absolute URLs.
  if (!target.startsWith('/') || target.startsWith('//')) {
    return false;
  }

  // Validate the path portion only: nginx matches its exclusions on $uri for the same reason, the
  // home-path check has to agree with useFirstLoginRedirect's pathname check, and a query value may
  // legitimately contain '..' (Grafana template variables such as ?var-version=1..2).
  const path = target.split(/[?#]/)[0];

  if (path.includes('\\') || path.includes('..')) {
    return false;
  }

  if (NOTHING_TO_RESTORE.includes(path)) {
    return false;
  }

  // Never store a target that itself bounces back out of the shell.
  return !isGrafanaDirectPath(path);
};

const dropGrafanaReturnTo = () => {
  sessionStorage.removeItem(GRAFANA_REDIRECT_TO_KEY);
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

/**
 * Remember where the user was before we bounce them to Grafana's login page. sessionStorage is
 * per-tab and survives the cross-document hop to /graph/login and back, which is exactly the
 * lifetime we need. It also stays clear of the localStorage keys that
 * ensureClientSessionListener() watches, so writing here cannot kick the client-session store.
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

  // Loop guard: a target we just restored and immediately bounced off of is not worth retrying.
  if (sessionStorage.getItem(RETURN_TO_LAST_KEY) === target) {
    sessionStorage.removeItem(RETURN_TO_LAST_KEY);
    return;
  }

  const stored: StoredReturnTo = { path: target, at: Date.now() };
  sessionStorage.setItem(RETURN_TO_KEY, JSON.stringify(stored));
};

/** Read and discard the pending target. Destructive, so it can only ever be acted on once. */
export const consumeReturnTo = (): string | null => {
  dropGrafanaReturnTo();

  const raw = sessionStorage.getItem(RETURN_TO_KEY);
  sessionStorage.removeItem(RETURN_TO_KEY);

  if (!raw) {
    sessionStorage.removeItem(RETURN_TO_LAST_KEY);
    return null;
  }

  let stored: StoredReturnTo;

  try {
    stored = JSON.parse(raw) as StoredReturnTo;
  } catch {
    sessionStorage.removeItem(RETURN_TO_LAST_KEY);
    return null;
  }

  const isUsable =
    typeof stored?.path === 'string' &&
    typeof stored?.at === 'number' &&
    isRestorableReturnTo(stored.path) &&
    Date.now() - stored.at <= AUTH_RETURN_TO_TTL_MS;

  if (!isUsable) {
    sessionStorage.removeItem(RETURN_TO_LAST_KEY);
    return null;
  }

  sessionStorage.setItem(RETURN_TO_LAST_KEY, stored.path);

  return stored.path;
};
