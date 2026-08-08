import { setTokenProvider, setOnUnauthorized } from '@sep/api';

/**
 * Interim SEP auth wiring (migration Option D).
 *
 * SEP's axios client delegates the bearer token via `setTokenProvider`. During the
 * migration the PMM dev proxy injects `PMM_DEV_SEP_INTERNAL_TOKEN` server-side,
 * so the browser sends no token — the provider returns `null`. `setOnUnauthorized` is a
 * no-op because there is no SEP login flow to redirect to (PMM owns the session).
 *
 * This is replaced by the token-exchange provider (Option B), which calls
 * `postSessionExchange()` (`POST /sep/api/oauth/session/exchange`, SEP-1692) to trade
 * PMM's session cookie for a short-lived SEP bearer, at which point `isAdmin` also
 * comes from the token's role claim rather than the internal token's service
 * principal, which hardcodes `is_admin = False`.
 */
export const initSepAuth = () => {
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
};
