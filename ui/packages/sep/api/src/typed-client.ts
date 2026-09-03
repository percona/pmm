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
/**
 * Typed request clients built on `openapi-fetch` + generated `paths` types.
 *
 * One client per OpenAPI spec (see `scripts/codegen.ts`):
 *   - `mainApi`      — core API (oauth, users)
 *   - `sepApi`       — SEP app
 *
 * The generated `paths` keys include SEP's own mount prefix (e.g.
 * `/api/users/me`) but not the `SEP_BASE_PATH` the side-car is reached under,
 * so every client uses `baseUrl: CLIENT_BASE_URL`, which appends it to the
 * current origin in the browser and falls back to a `http://localhost`
 * sentinel under Node/test environments — identical same-origin behaviour
 * to the axios `apiClient` in the browser.
 *
 * We share auth state with `apiClient` via `getToken()`/`emitUnauthorized()`
 * from `./client` — both the axios and fetch transports observe the same
 * `setTokenProvider()`/`setOnUnauthorized()` registration. Errors are
 * normalised to `ApiError` (via `throwOnApiError`) so callers see the same
 * shape regardless of which client they use.
 */
import createClient, { type Client, type Middleware } from 'openapi-fetch';
import { SEP_BASE_PATH } from './base';
import {
  emitUnauthorized,
  getToken,
  isTokenMintRequest,
  refreshAccessToken,
} from './client';
import { ApiError } from './errors';
import type { paths as MainPaths } from './generated/main';
import type { paths as SepPaths } from './generated/sep';

const IS_DEV = import.meta.env.DEV;

const isRefreshRequest = (url: string) => url.includes('/oauth/refresh');
const isLoginRequest = (url: string) => url.includes('/oauth/login');

/**
 * Whether a 401 on this URL is worth one silent mint-and-replay. Minting
 * endpoints are the recovery mechanism itself and login carries its own
 * credentials, so a 401 from either is the answer, not a stale token.
 */
const isReplayEligible = (url: string) =>
  !isTokenMintRequest(url) && !isLoginRequest(url);

/**
 * A 200 HTML response (e.g. a follow of a login redirect) means the session
 * is gone. The browser can't observe the 303, so content-type is the only
 * signal. Synthesise a 401 so the normal error path runs.
 */
function isHtmlLoginResponse(response: Response): boolean {
  const ct = response.headers.get('content-type') ?? '';
  return response.ok && ct.includes('text/html');
}

// `fetch` consumes a Request's body stream, so the instance handed to
// `onResponse` can no longer be re-sent. Stash an untouched clone taken before
// dispatch, keyed weakly so requests that never come back are not retained.
//
// Only replay-eligible requests are cloned: cloning buffers the body, and the
// endpoints excluded from the retry would never use theirs.
const pristineRequests = new WeakMap<Request, Request>();

/**
 * One silent recovery attempt for a 401: mint a fresh token — single-flighted
 * with every other caller, including the axios transport — and replay the
 * request with it.
 *
 * The replay goes through raw `fetch` rather than the typed client so it cannot
 * re-enter this middleware; that bounds recovery to a single extra round-trip
 * without needing a retry marker. Returns null when there is nothing to replay
 * or no token could be minted.
 */
async function replayWithFreshToken(
  request: Request
): Promise<Response | null> {
  const pristine = pristineRequests.get(request);
  if (!pristine) {
    return null;
  }
  pristineRequests.delete(request);

  const token = await refreshAccessToken();
  if (!token) {
    return null;
  }

  pristine.headers.set('Authorization', `Bearer ${token}`);
  return lazyFetch(pristine);
}

const authMiddleware: Middleware = {
  onRequest({ request }) {
    const token = getToken();
    if (token) {
      request.headers.set('Authorization', `Bearer ${token}`);
    }
    if (isReplayEligible(request.url)) {
      pristineRequests.set(request, request.clone());
    }
    if (IS_DEV) {
      // eslint-disable-next-line no-console
      console.debug(
        `[api] → ${request.method} ${new URL(request.url).pathname}`
      );
    }
    return request;
  },
  async onResponse({ request, response }) {
    if (IS_DEV) {
      // eslint-disable-next-line no-console
      console.debug(
        `[api] ← ${response.status} ${request.method} ${new URL(request.url).pathname}`
      );
    }

    if (isHtmlLoginResponse(response) && !isRefreshRequest(request.url)) {
      emitUnauthorized();
      return new Response(null, {
        status: 401,
        statusText: 'Session expired (redirected to login page)',
      });
    }

    if (response.status === 401 && isReplayEligible(request.url)) {
      const replayed = await replayWithFreshToken(request);
      if (replayed && replayed.status !== 401) {
        return replayed;
      }
      // Minting failed, or the replay was rejected too — the session is gone.
      emitUnauthorized();
      return replayed ?? response;
    }

    if (
      (response.status === 401 || response.status === 303) &&
      !isRefreshRequest(request.url)
    ) {
      // A 401 left here is a minting endpoint rejecting the ambient session —
      // "not signed in", which the auth layer must hear about. A 303 is the
      // login redirect on any endpoint.
      emitUnauthorized();
    }

    return response;
  },
};

// openapi-fetch builds absolute URLs as `new URL(baseUrl + path)`, which
// requires an absolute base — `baseUrl: ''` throws under Node because there
// is no implicit document origin to resolve against. Read `location.origin`
// when available (browser, jsdom-style tests), otherwise use a
// `http://localhost` sentinel; MSW intercepts by URL in either case.
const CLIENT_ORIGIN =
  typeof globalThis !== 'undefined' &&
  typeof (globalThis as { location?: Location }).location?.origin === 'string'
    ? (globalThis as { location: Location }).location.origin
    : 'http://localhost';
const CLIENT_BASE_URL = `${CLIENT_ORIGIN}${SEP_BASE_PATH}`;

// Resolve `fetch` lazily rather than capturing `globalThis.fetch` at module
// load. Test harnesses (MSW, happy-dom) patch the global after this file is
// imported, and openapi-fetch would otherwise call the pre-patch reference.
const lazyFetch: typeof fetch = (...args) => globalThis.fetch(...args);

function makeClient<Paths extends object>(): Client<Paths> {
  const client = createClient<Paths>({
    baseUrl: CLIENT_BASE_URL,
    fetch: lazyFetch,
  });
  client.use(authMiddleware);
  return client;
}

export const mainApi = makeClient<MainPaths>();
export const sepApi = makeClient<SepPaths>();

interface FetchResult<T> {
  data?: T;
  error?: unknown;
  response: Response;
}

function detailFromBody(body: unknown): string | undefined {
  if (body && typeof body === 'object' && 'detail' in body) {
    const detail = (body as { detail: unknown }).detail;
    if (typeof detail === 'string') {
      return detail;
    }
  }
  return undefined;
}

/**
 * Raise `ApiError` on any failure instead of returning openapi-fetch's
 * `{ data, error }` tuple. Covers three failure modes:
 *   1. Resolved tuple with `response.ok === false` (server error body).
 *   2. Resolved tuple with a synthesised 401 (HTML login redirect).
 *   3. Rejected promise (network failure, JSON parse error).
 *
 * Use inside React Query `queryFn`s so errors propagate to `error` on the
 * hook result and consumers can `instanceof ApiError` on them.
 */
export async function throwOnApiError<T>(
  promise: Promise<FetchResult<T>>
): Promise<T> {
  let result: FetchResult<T>;
  try {
    result = await promise;
  } catch (err) {
    if (err instanceof ApiError) {
      throw err;
    }
    if (err instanceof SyntaxError) {
      throw new ApiError(
        { kind: 'unknown', message: 'Malformed response body' },
        err
      );
    }
    // `TypeError` is the standard `fetch` rejection for network failures.
    const message = err instanceof Error ? err.message : 'Network error';
    throw new ApiError({ kind: 'network', message }, err);
  }

  const { data, error, response } = result;
  if (response.ok && error === undefined) {
    return data as T;
  }

  const url = (() => {
    try {
      return new URL(response.url).pathname;
    } catch {
      return undefined;
    }
  })();

  const detail =
    detailFromBody(error) ?? response.statusText ?? `HTTP ${response.status}`;
  throw new ApiError({
    kind: 'http',
    status: response.status,
    message: detail,
    data: error,
    url,
  });
}
