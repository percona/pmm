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

import { http, HttpResponse } from 'msw';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { SEP_BASE_PATH } from '../src/base';
import { setOnUnauthorized, setTokenProvider } from '../src/client';
import { ApiError } from '../src/errors';
import { mainApi, throwOnApiError } from '../src/typed-client';
import { server } from './msw-server';

const API = `http://localhost${SEP_BASE_PATH}/api`;

// openapi-fetch builds absolute URLs from a `baseUrl`. The generated paths
// already include SEP's own `/api/...` prefix, and our typed clients set
// `baseUrl` to an absolute origin plus `SEP_BASE_PATH`. Under Node we stub
// `globalThis.location` so the client's origin fallback resolves to
// `http://localhost` in these tests — matching the MSW handler URLs below.
//
// `typed-client.ts` evaluates `CLIENT_BASE_URL` at module load (i.e. before
// `beforeEach` runs), but the fallback path kicks in when `location` is
// absent — which it is the first time the module is imported under Node —
// so the computed base is already `http://localhost`. The stub here is
// belt-and-braces in case another test suite has set `location` first.
const ORIGINAL_LOCATION_DESCRIPTOR = Object.getOwnPropertyDescriptor(
  globalThis,
  'location'
);

beforeEach(() => {
  Object.defineProperty(globalThis, 'location', {
    configurable: true,
    value: {
      origin: 'http://localhost',
      href: 'http://localhost/',
    } as Location,
  });
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
});

afterEach(() => {
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
  if (ORIGINAL_LOCATION_DESCRIPTOR) {
    Object.defineProperty(globalThis, 'location', ORIGINAL_LOCATION_DESCRIPTOR);
  } else {
    // Node default: no `location` property at all. Remove the stub so other
    // test files observing `typeof location === 'undefined'` aren't fooled.
    delete (globalThis as { location?: unknown }).location;
  }
});

describe('typed-client — SEP mount point', () => {
  it('resolves a generated path under the prefix nginx exposes the side-car on', async () => {
    const seen = vi.fn();

    server.use(
      http.get('http://localhost/sep/api/users/me', ({ request }) => {
        seen(new URL(request.url).pathname);
        return HttpResponse.json({ id: 'x', username: 'u' });
      })
    );

    await mainApi.GET('/api/users/me');

    expect(seen).toHaveBeenCalledWith('/sep/api/users/me');
  });
});

describe('typed-client — auth middleware', () => {
  it('attaches Bearer token from the shared provider', async () => {
    setTokenProvider(() => 'shared-token');
    const seen = vi.fn();

    server.use(
      http.get(`${API}/users/me`, ({ request }) => {
        seen(request.headers.get('Authorization'));
        return HttpResponse.json({ id: 'x', username: 'u' });
      })
    );

    await mainApi.GET('/api/users/me');

    expect(seen).toHaveBeenCalledWith('Bearer shared-token');
  });

  it('calls the unauthorized handler on 401 (non-refresh endpoints)', async () => {
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.get(`${API}/users/me`, () =>
        HttpResponse.json({ detail: 'nope' }, { status: 401 })
      )
    );

    await mainApi.GET('/api/users/me');

    expect(onUnauth).toHaveBeenCalledOnce();
  });

  it('skips the unauthorized handler on refresh-endpoint 401', async () => {
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.post(`${API}/oauth/refresh`, () =>
        HttpResponse.json({ detail: 'bad' }, { status: 401 })
      )
    );

    await mainApi.POST('/api/oauth/refresh');

    expect(onUnauth).not.toHaveBeenCalled();
  });
});

describe('throwOnApiError', () => {
  it('returns typed data on 2xx', async () => {
    server.use(
      http.get(`${API}/users/me`, () =>
        HttpResponse.json({ id: 'abc', username: 'u' })
      )
    );

    const user = await throwOnApiError(mainApi.GET('/api/users/me'));

    expect(user).toMatchObject({ id: 'abc', username: 'u' });
  });

  it('maps a JSON 4xx body to ApiError with status + detail', async () => {
    server.use(
      http.get(`${API}/users/me`, () =>
        HttpResponse.json({ detail: 'not found' }, { status: 404 })
      )
    );

    await expect(
      throwOnApiError(mainApi.GET('/api/users/me'))
    ).rejects.toSatisfy((err) => {
      if (!(err instanceof ApiError)) {
        return false;
      }
      return (
        err.kind === 'http' && err.status === 404 && err.message === 'not found'
      );
    });
  });

  it('maps a network failure to ApiError with kind "network"', async () => {
    server.use(http.get(`${API}/users/me`, () => HttpResponse.error()));

    await expect(
      throwOnApiError(mainApi.GET('/api/users/me'))
    ).rejects.toSatisfy(
      (err) => err instanceof ApiError && err.kind === 'network'
    );
  });

  it('maps a malformed JSON body to ApiError (no raw SyntaxError leaks)', async () => {
    server.use(
      http.get(`${API}/users/me`, () =>
        HttpResponse.text('not-json', {
          status: 200,
          headers: { 'content-type': 'application/json' },
        })
      )
    );

    await expect(
      throwOnApiError(mainApi.GET('/api/users/me'))
    ).rejects.toBeInstanceOf(ApiError);
  });
});
