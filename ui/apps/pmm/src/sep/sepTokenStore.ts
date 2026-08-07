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
 * Two rules shape everything below.
 *
 * **Fail closed.** Any exchange failure drops the bearer immediately. Nothing
 * ever proceeds on a stale, expired, or unverified credential, and there is no
 * cached fallback to reach for. A session SEP has rejected is sticky: minting is
 * refused until the user retries, so a rejection cannot drive an exchange loop.
 *
 * **Never destroy user work.** Once a bearer has been held, the page is mounted
 * and may hold a half-filled form. From that point a failure is reported through
 * {@link SepAuthState.notice} — an inline notice beside the still-mounted page —
 * rather than by moving the phase to a full-screen state. Before that point
 * there is nothing to preserve, so a bootstrap failure takes over the page.
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

/**
 * Backoff for a renewal that failed for a reason that may not repeat. Quiet
 * while it retries; the user is only told once the attempts run out.
 */
const RENEWAL_RETRY_BASE_MS = 2_000;
const RENEWAL_RETRY_MAX_MS = 30_000;
const MAX_RENEWAL_RETRIES = 4;

/** What the page as a whole is doing. Drives which UI the gate renders. */
export type SepAuthPhase =
  /** No exchange attempted yet. */
  | 'idle'
  /** First exchange in flight; nothing to authenticate with yet. */
  | 'exchanging'
  /** A bearer has been held. The page is mounted and stays mounted. */
  | 'ready'
  /** SEP rejected the session before a bearer was ever held. */
  | 'signedOut'
  /** The exchange could not be completed before a bearer was ever held. */
  | 'unreachable';

/**
 * A failure that arrived after the page was already mounted. Surfaced beside
 * the page instead of replacing it, so in-progress work survives.
 */
export type SepAuthNotice = 'signedOut' | 'unreachable';

export interface SepAuthState {
  phase: SepAuthPhase;
  notice: SepAuthNotice | null;
}

let token: string | null = null;
let expiresAtMs = 0;
let phase: SepAuthPhase = 'idle';
let notice: SepAuthNotice | null = null;

/**
 * Sticky once SEP has rejected the session. Blocks minting outright — without
 * it, every subsequent request would 401, trigger a mint, be rejected, and
 * repeat. Only {@link retrySepAuth} clears it.
 */
let sessionRejected = false;

let renewalTimer: ReturnType<typeof setTimeout> | null = null;
let retryTimer: ReturnType<typeof setTimeout> | null = null;
let renewalRetries = 0;

const listeners = new Set<() => void>();

// `useSyncExternalStore` compares snapshots by identity, so hand out a cached
// object and only replace it when something actually changed.
let snapshot: SepAuthState = { phase, notice };

const publish = () => {
  if (snapshot.phase === phase && snapshot.notice === notice) {
    return;
  }
  snapshot = { phase, notice };
  listeners.forEach((listener) => listener());
};

const setPhase = (next: SepAuthPhase) => {
  phase = next;
  publish();
};

const clearTimer = (timer: ReturnType<typeof setTimeout> | null) => {
  if (timer !== null) {
    clearTimeout(timer);
  }
  return null;
};

/** Subscribe to state changes. Pairs with {@link getSepAuthState}. */
export const subscribeSepAuth = (listener: () => void) => {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
};

export const getSepAuthState = (): SepAuthState => snapshot;

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

/** Drop the bearer and stop every pending renewal. */
const clearSepToken = () => {
  token = null;
  expiresAtMs = 0;
  renewalTimer = clearTimer(renewalTimer);
  retryTimer = clearTimer(retryTimer);
};

/**
 * Drop the bearer and report the failure at the right altitude.
 *
 * Before a bearer has ever been held there is no work in progress, so the
 * failure takes over the page. After that the page stays exactly as it is and
 * the failure becomes an inline notice — a background renewal must never
 * discard what the user was typing.
 */
const failClosed = (kind: SepAuthNotice) => {
  clearSepToken();
  if (phase === 'ready') {
    notice = kind;
  } else {
    phase = kind;
    notice = null;
  }
  publish();
};

/**
 * Renew ahead of expiry so the bearer is replaced before any request can carry
 * a dead one.
 *
 * A backgrounded tab has its timers throttled and may miss the window; the 401
 * retry in both transports is the backstop for that.
 */
const scheduleRenewal = (expiresIn: number) => {
  renewalTimer = clearTimer(renewalTimer);
  const delay = Math.max(
    expiresIn * 1000 - EXPIRY_SKEW_MS,
    MIN_RENEWAL_DELAY_MS
  );
  renewalTimer = setTimeout(() => {
    renewalTimer = null;
    void renew();
  }, delay);
};

const scheduleRenewalRetry = () => {
  retryTimer = clearTimer(retryTimer);
  const delay = Math.min(
    RENEWAL_RETRY_BASE_MS * 2 ** (renewalRetries - 1),
    RENEWAL_RETRY_MAX_MS
  );
  retryTimer = setTimeout(() => {
    retryTimer = null;
    void renew();
  }, delay);
};

const renew = async () => {
  // Joins the shared single-flight, so a renewal racing a 401 retry is one call.
  const minted = await refreshAccessToken();
  if (minted !== null) {
    return;
  }
  if (sessionRejected) {
    // A rejected session is terminal and `markSepSignedOut` already reported it.
    // Retrying would only repeat the rejection.
    return;
  }

  // Transient: the bearer is gone either way (fail closed), but keep quiet and
  // back off — a blip should not put a notice in front of someone mid-form.
  clearSepToken();
  if (renewalRetries < MAX_RENEWAL_RETRIES) {
    renewalRetries += 1;
    scheduleRenewalRetry();
    return;
  }
  failClosed('unreachable');
};

/**
 * Record a freshly minted bearer. Wired to `setOnRefreshed`, so it runs whoever
 * triggered the exchange — the gate, the renewal timer, or a 401 retry.
 */
export const recordSepToken = (accessToken: string, expiresIn: number) => {
  token = accessToken;
  expiresAtMs = Date.now() + expiresIn * 1000;
  // A successful exchange proves the session is good and clears whatever the
  // last failure said about it.
  sessionRejected = false;
  renewalRetries = 0;
  retryTimer = clearTimer(retryTimer);
  scheduleRenewal(expiresIn);
  phase = 'ready';
  notice = null;
  publish();
};

/**
 * Record that SEP rejected the session, and refuse to exchange again until
 * {@link retrySepAuth}.
 *
 * Wired to `setOnUnauthorized`, which fires when a SEP call 401s and no token
 * could be minted to replay it. Also called directly when the exchange itself
 * 401s, so the sticky guarantee holds even if the transports' unauthorized
 * wiring changes.
 */
export const markSepSignedOut = () => {
  sessionRejected = true;
  failClosed('signedOut');
};

/**
 * Mint a bearer by exchanging PMM's session cookie. Wired to `setTokenMinter`,
 * replacing `@sep/api`'s default `POST /oauth/refresh` — PMM's embedding issues
 * no refresh cookie, so the default would 401 on every recovery attempt.
 */
export const mintSepToken = async (): Promise<MintedToken | null> => {
  if (sessionRejected) {
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
    return true;
  }
  if (sessionRejected) {
    return false;
  }

  // Only show the spinner before the page exists. Once mounted it stays put.
  if (phase !== 'ready') {
    setPhase('exchanging');
  }

  const minted = await refreshAccessToken();
  if (minted !== null) {
    return true;
  }
  if (!sessionRejected) {
    failClosed('unreachable');
  }
  return false;
};

/**
 * Clear a terminal state and exchange again. The only way out of a rejected
 * session, so recovery stays an explicit user action rather than a loop.
 */
export const retrySepAuth = (): Promise<boolean> => {
  sessionRejected = false;
  renewalRetries = 0;
  clearSepToken();
  if (phase !== 'ready') {
    setPhase('idle');
  }
  return ensureSepToken();
};

/** Reset every module-level field. Tests only. */
export const resetSepAuthStore = () => {
  clearSepToken();
  phase = 'idle';
  notice = null;
  sessionRejected = false;
  renewalRetries = 0;
  snapshot = { phase, notice };
  listeners.clear();
};
