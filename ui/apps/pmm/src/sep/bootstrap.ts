import {
  setOnRefreshed,
  setOnUnauthorized,
  setTokenMinter,
  setTokenProvider,
} from '@sep/api';
import {
  getSepToken,
  markSepSignedOut,
  mintSepToken,
  recordSepToken,
} from './sepTokenStore';

/**
 * SEP auth wiring for the embedded UI.
 *
 * PMM owns the session, so SEP is authenticated as the actual PMM user by
 * exchanging the `pmm_session` cookie for a short-lived SEP bearer
 * (`POST /api/oauth/session/exchange`, SEP-1692) rather than by logging in.
 * This replaces the interim wiring in which the dev proxy injected
 * `PMM_DEV_SEP_INTERNAL_TOKEN` server-side: that authenticated as SEP's
 * internal service principal, which hardcodes `is_admin = False`, so every
 * admin-gated SEP surface answered 403.
 *
 * Registration is side-effect free — no network call happens here. The first
 * exchange is triggered by `SepAuthGate` when a SEP route mounts, so PMM users
 * who never open one never talk to SEP. State and lifetime live in
 * `./sepTokenStore`.
 */
export const initSepAuth = () => {
  setTokenProvider(getSepToken);
  setTokenMinter(mintSepToken);
  setOnRefreshed(recordSepToken);
  setOnUnauthorized(markSepSignedOut);
};
