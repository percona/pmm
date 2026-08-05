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

import { apiClient } from './client';
import type {
  SessionExchangeTokenResponse,
  SPAOAuthTokenResponse,
  User,
} from './types/api';

/**
 * POST /api/oauth/login
 *
 * SPA login. Backend runs the Casdoor password grant server-side, stores
 * the refresh token in an `HttpOnly` cookie scoped to `/api/oauth`, and
 * returns the access token (plus TTL) in JSON. The refresh token is never
 * seen by JavaScript.
 */
export async function postLogin(
  username: string,
  password: string
): Promise<SPAOAuthTokenResponse> {
  const { data } = await apiClient.post<SPAOAuthTokenResponse>('/oauth/login', {
    username,
    password,
  });
  return data;
}

/**
 * POST /api/oauth/refresh
 *
 * Mint a fresh access token from the `HttpOnly` refresh cookie. The cookie
 * is sent automatically by the browser; there is no request body. The
 * backend rotates the cookie on success.
 *
 * Returns the slim SPA shape — `access_token` and `expires_in` only. The
 * refresh token is never in the response body.
 *
 * Note: most callers should funnel through `refreshAccessToken()` in
 * `./client` instead, which single-flights concurrent refresh calls so the
 * cookie is not rotated twice by parallel code paths.
 */
export async function postRefresh(): Promise<SPAOAuthTokenResponse> {
  const { data } =
    await apiClient.post<SPAOAuthTokenResponse>('/oauth/refresh');
  return data;
}

/**
 * POST /api/oauth/session
 *
 * Ambient auto-login from an existing PMM/Grafana session cookie. The browser
 * sends the `grafana_session` cookie automatically; there is no request body.
 * On success the backend sets the `HttpOnly` refresh cookie and returns the
 * slim access shape; a 401 (no valid ambient session) means "not signed in".
 */
export async function postSession(): Promise<SPAOAuthTokenResponse> {
  const { data } =
    await apiClient.post<SPAOAuthTokenResponse>('/oauth/session');
  return data;
}

/**
 * POST /api/oauth/session/exchange
 *
 * Exchange an ambient PMM/Grafana session cookie for a short-lived SEP bearer.
 * The browser sends the `grafana_session` cookie automatically; there is no
 * request body. Unlike `postSession`, no cookie is set and no refresh token is
 * issued — the caller holds the token in memory and re-exchanges before
 * `expires_in` elapses. A 401 means "no valid ambient session".
 *
 * This is the endpoint the PMM-embedded token provider will call to replace
 * the interim `PMM_DEV_SEP_INTERNAL_TOKEN` wiring in
 * `apps/pmm/src/sep/bootstrap.ts`,
 * whose service principal hardcodes `is_admin = False`.
 */
export async function postSessionExchange(): Promise<SessionExchangeTokenResponse> {
  const { data } = await apiClient.post<SessionExchangeTokenResponse>(
    '/oauth/session/exchange'
  );
  return data;
}

/**
 * POST /api/oauth/logout
 *
 * End the SPA session server-side: clears/rotates the `HttpOnly` refresh
 * cookie so a subsequent bootstrap refresh cannot mint a new access token,
 * and revokes the access token at Casdoor. Returns 204 on success;
 * callers should still clear local state even if this request fails.
 */
export async function postLogout(): Promise<void> {
  await apiClient.post('/oauth/logout');
}

/**
 * GET /api/users/me
 *
 * Fetch the currently authenticated user profile.
 */
export async function fetchCurrentUser(): Promise<User> {
  const { data } = await apiClient.get<User>('/users/me');
  return data;
}
