import { setTokenProvider, setOnUnauthorized } from '@sep/api';

/**
 * Interim SEP auth wiring (migration Option D).
 *
 * SEP's axios client delegates the bearer token via `setTokenProvider`. During the
 * migration the PMM dev proxy injects `SEP_INTERNAL_TOKEN` server-side, so the
 * browser sends no token — the provider returns `null`. `setOnUnauthorized` is a
 * no-op because there is no SEP login flow to redirect to (PMM owns the session).
 *
 * This is replaced by the token-exchange provider (Option B) that fetches
 * `/v1/sep/token`, at which point `isAdmin` also comes from the token's role claim.
 */
export const initSepAuth = () => {
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
};
