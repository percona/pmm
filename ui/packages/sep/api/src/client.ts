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

/// <reference path="./vite-env.d.ts" />
import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { SEP_BASE_PATH } from './base';
import { normalizeAxiosError } from './errors';

// ── Token provider ─────────────────────────────────────────────────────
// Instead of owning a copy of the access token, the API layer delegates
// token retrieval to a provider injected by the auth layer (AuthProvider).
// This ensures a single source of truth and prevents desync between
// AuthProvider state and the API client.
type TokenProvider = () => string | null;
type OnUnauthorized = () => void;
type OnRefreshed = (accessToken: string, expiresIn: number) => void;

let _getToken: TokenProvider = () => null;
let _onUnauthorized: OnUnauthorized = () => {};
let _onRefreshed: OnRefreshed = () => {};

/** Inject a callback that returns the current access token. */
export function setTokenProvider(provider: TokenProvider) {
  _getToken = provider;
}

/** Inject a callback invoked when the API receives an unauthorized response. */
export function setOnUnauthorized(handler: OnUnauthorized) {
  _onUnauthorized = handler;
}

/**
 * Inject a callback invoked after a successful silent refresh triggered by
 * the 401 retry interceptor. Lets the auth layer sync state + re-schedule
 * its own refresh timer without owning the refresh network call.
 */
export function setOnRefreshed(handler: OnRefreshed) {
  _onRefreshed = handler;
}

/** Read the currently registered token provider. Used by the typed client. */
export function getToken(): string | null {
  return _getToken();
}

/** Invoke the currently registered unauthorized handler. */
export function emitUnauthorized(): void {
  _onUnauthorized();
}

// ── Dev logging flag ───────────────────────────────────────────────────
// Vite statically replaces `import.meta.env.DEV` at build time, so the
// logging branches are dead-code-eliminated in production bundles.
const IS_DEV = import.meta.env.DEV;

/**
 * Primary API client. Request/response payloads are passed through verbatim —
 * field casing matches the OpenAPI spec (see `src/generated/`).
 */
export const apiClient = axios.create({
  baseURL: `${SEP_BASE_PATH}/api`,
  // Send the HttpOnly refresh cookie on /sep/api/oauth/* requests (same-origin
  // requests already send cookies, but make the intent explicit so tests
  // and any future cross-origin config keep working).
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
  validateStatus: (status) => status >= 200 && status < 300,
});

// ── Request: attach Bearer token, optionally log ───────────────────────
apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  // Do not overwrite an already-set Authorization header so the 401 retry
  // path (which sets the header explicitly from the freshly-refreshed
  // token) is correct even if the external token provider has not yet
  // observed the refresh.
  if (!config.headers.Authorization) {
    const token = _getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }
  if (IS_DEV) {
    const method = config.method?.toUpperCase() ?? 'GET';
    // eslint-disable-next-line no-console
    console.debug(`[api] → ${method} ${config.url ?? ''}`);
  }
  return config;
});

// ── Response: 401 refresh-retry, unauthorized handling, error normalization ──
// Refresh endpoint is the recovery mechanism — its failures must not trigger
// the retry loop or the global unauthorized handler, or a genuine refresh
// rejection would hard-redirect instead of letting the AuthProvider bootstrap
// handle it cleanly.
const isRefreshRequest = (url: string | undefined) =>
  !!url && url.includes('/oauth/refresh');
const isLoginRequest = (url: string | undefined) =>
  !!url && url.includes('/oauth/login');

// Internal marker so retried requests don't loop through the refresh path
// again on a second 401.
type RetriableConfig = InternalAxiosRequestConfig & { _retried?: boolean };

// Single-flight refresh: concurrent callers (401 retry path + background
// timer + bootstrap) share one in-flight /oauth/refresh call. The promise
// resolves to the new access token on success and null on failure so
// callers can decide whether to retry, surface the 401, or force logout.
//
// All refresh traffic must funnel through here — the refresh token cookie
// rotates on every successful call, so parallel refreshes from different
// code paths would invalidate each other.
let refreshInFlight: Promise<string | null> | null = null;

/**
 * Trigger (or join) the shared silent refresh. Resolves with the new
 * access token, or null if the refresh failed (missing/invalid cookie,
 * network error, Casdoor rejection).
 */
export function refreshAccessToken(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      // Only the network call is inside the try/catch — a synchronous throw
      // from the externally-injected _onRefreshed handler must NOT be reported
      // as a failed refresh, otherwise the auth layer would force-logout a
      // user whose cookie rotation succeeded on the backend.
      let data: { access_token: string; expires_in: number };
      try {
        const response = await apiClient.post<{
          access_token: string;
          expires_in: number;
        }>('/oauth/refresh');
        data = response.data;
      } catch {
        return null;
      } finally {
        // Clear after the microtask so all concurrent awaiters see the same
        // resolved value before a fresh attempt becomes possible.
        queueMicrotask(() => {
          refreshInFlight = null;
        });
      }
      try {
        _onRefreshed(data.access_token, data.expires_in);
      } catch (handlerError) {
        // A throwing auth-layer handler must not invalidate a cookie rotation
        // that already succeeded on the backend: it would reject the shared
        // promise and force-logout every awaiting caller. It does leave the
        // auth layer without the new token or its expiry while this function
        // still returns the token, so record that inconsistency — never the
        // token or expiry themselves.
        // eslint-disable-next-line no-console
        console.error(
          '[api] onRefreshed handler threw; the rotated token was not recorded',
          handlerError
        );
      }
      return data.access_token;
    })();
  }
  return refreshInFlight;
}

apiClient.interceptors.response.use(
  (response) => {
    if (IS_DEV) {
      const method = response.config?.method?.toUpperCase() ?? 'GET';
      // eslint-disable-next-line no-console
      console.debug(
        `[api] ← ${response.status} ${method} ${response.config?.url ?? ''}`
      );
    }
    return response;
  },
  async (error: AxiosError) => {
    if (axios.isAxiosError(error)) {
      const status = error.response?.status;
      const config = error.config as RetriableConfig | undefined;
      const url = config?.url;

      // 401 on a normal request: attempt one silent refresh, then retry.
      // Skip the refresh/login endpoints themselves and already-retried requests.
      if (
        status === 401 &&
        config &&
        !config._retried &&
        !isRefreshRequest(url) &&
        !isLoginRequest(url)
      ) {
        const newToken = await refreshAccessToken();
        if (newToken) {
          config._retried = true;
          // Set the header explicitly from the refreshed token so the
          // retry is self-contained and not dependent on external token-
          // provider wiring. The request interceptor preserves an already-
          // set Authorization header.
          config.headers = config.headers ?? {};
          config.headers.Authorization = `Bearer ${newToken}`;
          return apiClient.request(config);
        }
        // Refresh failed — treat as unauthorized.
        _onUnauthorized();
      } else if ((status === 401 || status === 303) && !isRefreshRequest(url)) {
        // Retried request still 401, or non-retriable 401 (e.g. login itself).
        _onUnauthorized();
      }
    }
    const apiError = normalizeAxiosError(error);
    if (IS_DEV) {
      const loc =
        apiError.method && apiError.url
          ? `${apiError.method} ${apiError.url}`
          : '';
      // eslint-disable-next-line no-console
      console.debug(
        `[api] ✗ ${apiError.kind}${apiError.status ? ` ${apiError.status}` : ''} ${loc} — ${apiError.message}`
      );
    }
    return Promise.reject(apiError);
  }
);
