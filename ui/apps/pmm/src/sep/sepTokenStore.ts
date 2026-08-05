import {
  ApiError,
  type MintedToken,
  postSessionExchange,
  refreshAccessToken,
} from '@sep/api';

/**
 * In-memory holder for the SEP bearer PMM mints from its own session.
 *
 * `POST /api/oauth/session/exchange` (SEP-1692) trades the ambient `pmm_session`
 * cookie — attached automatically, same origin through PMM's proxy — for a
 * short-lived bearer. No cookie is set and no refresh token is issued, so the
 * holder re-exchanges before expiry instead of refreshing.
 *
 * The token never leaves this module: no `localStorage`, no `sessionStorage`, no
 * query cache. A page reload re-exchanges from the cookie, which is the point —
 * every exchange re-reads the identity, so a role change lands within one bearer
 * lifetime (5 minutes by default).
 *
 * Concurrency is not handled here. `refreshAccessToken()` in `@sep/api`
 * single-flights every caller — the renewal timer, the initial gate, and each
 * transport's 401 retry — so a burst of parallel SEP requests triggers one
 * exchange.
 */

/**
 * Renew this far before the bearer actually expires, so in-flight requests
 * carry a token that is still valid when SEP validates it.
 */
const EXPIRY_SKEW_MS = 30_000;

/** Floor for the renewal delay, in case SEP ever issues a very short TTL. */
const MIN_RENEWAL_DELAY_MS = 5_000;

export type SepAuthStatus =
  /** No exchange attempted yet. */
  | 'idle'
  /** First exchange in flight; nothing to authenticate with yet. */
  | 'exchanging'
  /** A usable bearer is held. */
  | 'ready'
  /**
   * SEP rejected the session. Sticky: minting is refused until
   * {@link retrySepAuth} clears it, so a rejected session cannot drive an
   * exchange loop.
   */
  | 'signedOut'
  /** The exchange failed for a reason that may not repeat (network, 5xx). */
  | 'error';

let token: string | null = null;
let expiresAtMs = 0;
let status: SepAuthStatus = 'idle';
let renewalTimer: ReturnType<typeof setTimeout> | null = null;

const listeners = new Set<() => void>();

const setStatus = (next: SepAuthStatus) => {
  if (status === next) {
    return;
  }
  status = next;
  listeners.forEach((listener) => listener());
};

const clearRenewalTimer = () => {
  if (renewalTimer !== null) {
    clearTimeout(renewalTimer);
    renewalTimer = null;
  }
};

/** Subscribe to status changes. Pairs with {@link getSepAuthStatus}. */
export const subscribeSepAuth = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

export const getSepAuthStatus = (): SepAuthStatus => status;

/**
 * Current bearer, or null once it has expired.
 *
 * Synchronous because `setTokenProvider` is: the transports read it while
 * building a request and cannot await. An expired token yields null rather than
 * a stale bearer, and the resulting 401 routes into the transports' retry, which
 * mints and replays.
 */
export const getSepToken = (): string | null =>
  token !== null && Date.now() < expiresAtMs ? token : null;

/**
 * Renew ahead of expiry so the bearer is replaced before any request can carry
 * a dead one.
 *
 * A backgrounded tab has its timers throttled and may miss the window; the 401
 * retry in both transports is the backstop for that.
 */
const scheduleRenewal = (expiresIn: number) => {
  clearRenewalTimer();
  const delay = Math.max(
    expiresIn * 1000 - EXPIRY_SKEW_MS,
    MIN_RENEWAL_DELAY_MS
  );
  renewalTimer = setTimeout(() => {
    renewalTimer = null;
    void renew();
  }, delay);
};

const renew = async () => {
  // Joins the shared single-flight, so a renewal racing a 401 retry is one call.
  const minted = await refreshAccessToken();
  if (!minted && getSepAuthStatus() !== 'signedOut') {
    // Nothing replaced a bearer that is at most EXPIRY_SKEW_MS from useless.
    // Drop it, but leave the status alone: demoting a mounted SEP page to an
    // error screen over a background blip would discard whatever the user was
    // in the middle of. The next SEP request 401s, mints, and replays — and if
    // that mint is rejected too, the unauthorized path reports it properly.
    clearSepToken();
  }
};

const clearSepToken = () => {
  token = null;
  expiresAtMs = 0;
  clearRenewalTimer();
};

/**
 * Record a freshly minted bearer. Wired to `setOnRefreshed`, so it runs whoever
 * triggered the exchange — the gate, the renewal timer, or a 401 retry.
 */
export const recordSepToken = (accessToken: string, expiresIn: number) => {
  token = accessToken;
  expiresAtMs = Date.now() + expiresIn * 1000;
  scheduleRenewal(expiresIn);
  setStatus('ready');
};

/**
 * Drop the bearer and refuse further exchanges until {@link retrySepAuth}.
 *
 * Wired to `setOnUnauthorized`, which fires when a SEP call 401s and no token
 * could be minted to replay it. Also reached directly when the exchange itself
 * 401s, so the sticky guarantee holds even if the transports' unauthorized
 * wiring changes.
 */
export const markSepSignedOut = () => {
  clearSepToken();
  setStatus('signedOut');
};

/**
 * Mint a bearer by exchanging PMM's session cookie. Wired to `setTokenMinter`,
 * replacing `@sep/api`'s default `POST /oauth/refresh` — PMM's embedding issues
 * no refresh cookie, so the default would 401 on every recovery attempt.
 */
export const mintSepToken = async (): Promise<MintedToken | null> => {
  if (status === 'signedOut') {
    return null;
  }
  try {
    return await postSessionExchange();
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) {
      markSepSignedOut();
    }
    return null;
  }
};

/**
 * Ensure a usable bearer exists, exchanging if needed. Resolves true when SEP
 * calls can be authenticated.
 *
 * Concurrent callers coalesce inside `refreshAccessToken()`.
 */
export const ensureSepToken = async (): Promise<boolean> => {
  if (getSepToken() !== null) {
    setStatus('ready');
    return true;
  }
  if (status === 'signedOut') {
    return false;
  }

  setStatus('exchanging');
  const minted = await refreshAccessToken();
  if (minted !== null) {
    return true;
  }
  // Read through the getter: the awaited mint may have flipped the status to
  // `signedOut`, which TypeScript cannot see through the await.
  if (getSepAuthStatus() !== 'signedOut') {
    setStatus('error');
  }
  return false;
};

/**
 * Clear a terminal state and exchange again. The only way out of `signedOut`,
 * so retrying stays an explicit user action rather than an automatic loop.
 */
export const retrySepAuth = (): Promise<boolean> => {
  clearSepToken();
  setStatus('idle');
  return ensureSepToken();
};

/** Reset every module-level field. Tests only. */
export const resetSepAuthStore = () => {
  clearSepToken();
  status = 'idle';
  listeners.clear();
};
