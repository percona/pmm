/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { createContext, useContext, useMemo } from 'react';

/**
 * Session state the embedding application owns.
 *
 * The context lives here, at the root of the SEP dependency graph, so the
 * framework and every plugin package can read it — they all depend on
 * `@sep/api` and none of them may depend on the host application. The provider
 * itself stays in the host (`apps/pmm/src/sep/SepAuthProvider.tsx`).
 *
 * Narrower than SEP's own `AuthSession`, which also carries the user record,
 * the access token and `login`/`logout`. PMM owns the session: the bearer lives
 * in `apps/pmm/src/sep/sepTokenStore.ts` and there is no SEP login form, so
 * those fields would have nothing truthful to hold. Only the identity the UI
 * actually gates on is modelled here.
 *
 * One context carries both the session and the capability derived from it, so a
 * role change re-renders every capability consumer too.
 */
export interface AuthSession {
  /**
   * Administrator identity. Read this only for genuinely admin-only surfaces.
   * Per-plugin write controls gate on {@link AuthState.canMutate} instead.
   */
  isAdmin: boolean;
}

/** {@link AuthSession} plus the capabilities derived from it. */
export interface AuthState extends AuthSession {
  /**
   * Whether this session may mutate: the gate every per-plugin create /
   * execute / stop / retry / delete control reads.
   *
   * Semantically distinct from {@link AuthSession.isAdmin} even though it is
   * exactly that today. The server already resolves a minimum role per route
   * rather than one administrator flag, so widening the UI to match is an edit
   * to {@link deriveCanMutate} — per-control minimum roles — and to no call
   * site.
   */
  canMutate: boolean;
}

/**
 * Single derivation of "may this session mutate?" from session state.
 *
 * Deliberately the administrator flag and nothing finer: most unsafe SEP routes
 * require `admin`, so keying on a lesser role here would put back the controls
 * that answer 403.
 */
export function deriveCanMutate(session: AuthSession): boolean {
  return session.isAdmin;
}

/**
 * Resolved state for a consumer rendered outside an `AuthProvider`: non-admin,
 * and therefore unable to mutate. Tests mount framework and plugin components
 * without the host's provider, so a missing provider must degrade to the
 * least-privileged state rather than throw.
 */
export const UNAUTHENTICATED_SESSION: AuthSession = Object.freeze({
  isAdmin: false,
});

export const AuthContext = createContext<AuthSession | null>(null);

let warnedMissingProvider = false;

/**
 * Warn once per bundle when the provider is missing. Hiding controls is a quiet
 * failure, so a stray consumer mounted outside the provider would otherwise
 * silently look like a non-admin session.
 */
function warnMissingProvider(): void {
  if (warnedMissingProvider || !import.meta.env?.DEV) {
    return;
  }
  warnedMissingProvider = true;
  // eslint-disable-next-line no-console -- surface a silently degraded session in dev
  console.warn(
    'useAuth() was called outside an AuthProvider — falling back to a ' +
      'non-admin session. Mutation controls will be hidden.'
  );
}

/**
 * Read the current session and its derived capabilities.
 *
 * Resolves to {@link UNAUTHENTICATED_SESSION} when no provider is mounted.
 */
export function useAuth(): AuthState {
  const session = useContext(AuthContext);
  if (!session) {
    warnMissingProvider();
  }
  const resolved = session ?? UNAUTHENTICATED_SESSION;
  return useMemo(
    () => ({ ...resolved, canMutate: deriveCanMutate(resolved) }),
    [resolved]
  );
}
