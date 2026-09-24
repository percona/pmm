import {
  setOnRefreshed,
  setOnUnauthorized,
  setTokenMinter,
  setTokenProvider,
} from '@pmm-extensions/api';
import {
  getExtensionsToken,
  markExtensionsSignedOut,
  mintExtensionsToken,
  recordExtensionsToken,
} from './extensionsTokenStore';

/**
 * PMM Extensions auth wiring for the embedded UI.
 *
 * PMM owns the session, so the side-car is authenticated as the actual PMM user by
 * exchanging the `pmm_session` cookie for a short-lived side-car bearer
 * (`POST /extensions/api/oauth/session/exchange`, SEP-1692) rather than by logging in.
 * This replaces the interim wiring in which the dev proxy injected
 * `PMM_DEV_EXTENSIONS_INTERNAL_TOKEN` server-side: that authenticated as the side-car's
 * internal service principal, which hardcodes `is_admin = False`, so every
 * admin-gated PMM Extensions surface answered 403.
 *
 * Registration is side-effect free — no network call happens here. The first
 * exchange is triggered by `ExtensionsAuthGate` when a PMM Extensions route mounts, so PMM users
 * who never open one never talk to the side-car. State and lifetime live in
 * `./extensionsTokenStore`.
 */
export const initExtensionsAuth = () => {
  setTokenProvider(getExtensionsToken);
  setTokenMinter(mintExtensionsToken);
  setOnRefreshed(recordExtensionsToken);
  setOnUnauthorized(markExtensionsSignedOut);
};
