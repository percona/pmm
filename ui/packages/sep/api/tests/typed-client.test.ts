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
import {
  setOnRefreshed,
  setOnUnauthorized,
  setTokenMinter,
  setTokenProvider,
} from '../src/client';
import { ApiError } from '../src/errors';
import { mainApi, sepApi, throwOnApiError } from '../src/typed-client';
import { server } from './msw-server';

// openapi-fetch builds absolute URLs from a `baseUrl`. The generated paths
// already include the `/api/...` prefix, and our typed clients set
// `baseUrl` to an absolute origin. Under Node we stub `globalThis.location`
// so the client's origin fallback resolves to `http://localhost` in these
// tests — matching the MSW handler URLs below.
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
  setOnRefreshed(() => {});
  // Default to a minter that cannot recover, so the 401 tests below observe the
  // give-up path without reaching the network. The recovery suite opts in.
  setTokenMinter(async () => null);
});

afterEach(() => {
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
  setOnRefreshed(() => {});
  setTokenMinter(null);
  if (ORIGINAL_LOCATION_DESCRIPTOR) {
    Object.defineProperty(globalThis, 'location', ORIGINAL_LOCATION_DESCRIPTOR);
  } else {
    // Node default: no `location` property at all. Remove the stub so other
    // test files observing `typeof location === 'undefined'` aren't fooled.
    delete (globalThis as { location?: unknown }).location;
  }
});

describe('typed-client — auth middleware', () => {
  it('attaches Bearer token from the shared provider', async () => {
    setTokenProvider(() => 'shared-token');
    const seen = vi.fn();

    server.use(
      http.get('http://localhost/api/users/me', ({ request }) => {
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
      http.get('http://localhost/api/users/me', () =>
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
      http.post('http://localhost/api/oauth/refresh', () =>
        HttpResponse.json({ detail: 'bad' }, { status: 401 })
      )
    );

    await mainApi.POST('/api/oauth/refresh');

    expect(onUnauth).not.toHaveBeenCalled();
  });

  it('synthesises a 401 and notifies when the server returns HTML (session expired)', async () => {
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.get('http://localhost/api/users/me', () =>
        HttpResponse.html('<html>login</html>', { status: 200 })
      )
    );

    await expect(
      throwOnApiError(mainApi.GET('/api/users/me'))
    ).rejects.toSatisfy((err) => err instanceof ApiError && err.status === 401);
    expect(onUnauth).toHaveBeenCalledOnce();
  });
});

describe('typed-client — 401 recovery', () => {
  const mintOnce = (token: string) => {
    const minter = vi.fn(async () => ({
      access_token: token,
      expires_in: 300,
    }));
    setTokenMinter(minter);
    return minter;
  };

  it('mints a fresh token and replays the request', async () => {
    let currentToken = 'stale';
    setTokenProvider(() => currentToken);
    setOnRefreshed((token) => {
      currentToken = token;
    });
    const minter = mintOnce('fresh');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);
    const seenAuth: Array<string | null> = [];

    server.use(
      http.get('http://localhost/api/users/me', ({ request }) => {
        const auth = request.headers.get('Authorization');
        seenAuth.push(auth);
        if (auth === 'Bearer fresh') {
          return HttpResponse.json({ id: 'abc', username: 'u' });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      })
    );

    const user = await throwOnApiError(mainApi.GET('/api/users/me'));

    expect(user).toMatchObject({ id: 'abc' });
    expect(seenAuth).toEqual(['Bearer stale', 'Bearer fresh']);
    expect(minter).toHaveBeenCalledOnce();
    expect(onUnauth).not.toHaveBeenCalled();
  });

  it('replays a request body — `fetch` consumed the original stream', async () => {
    setTokenProvider(() => 'stale');
    mintOnce('fresh');
    const seenBodies: unknown[] = [];

    server.use(
      http.post(
        'http://localhost/api/apps/inventory/sync/',
        async ({ request }) => {
          const auth = request.headers.get('Authorization');
          seenBodies.push(await request.json());
          if (auth === 'Bearer fresh') {
            return HttpResponse.json({ status: 'queued' });
          }
          return HttpResponse.json({ detail: 'expired' }, { status: 401 });
        }
      )
    );

    await throwOnApiError(
      sepApi.POST('/api/apps/inventory/sync/', {
        body: { syncer: 'mod.Cls' },
      })
    );

    expect(seenBodies).toEqual([{ syncer: 'mod.Cls' }, { syncer: 'mod.Cls' }]);
  });

  it('replays at most once, then reports unauthorized', async () => {
    setTokenProvider(() => 'stale');
    const minter = mintOnce('fresh');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);
    let calls = 0;

    server.use(
      http.get('http://localhost/api/users/me', () => {
        calls += 1;
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      })
    );

    await expect(
      throwOnApiError(mainApi.GET('/api/users/me'))
    ).rejects.toSatisfy((err) => err instanceof ApiError && err.status === 401);
    expect(calls).toBe(2);
    expect(minter).toHaveBeenCalledOnce();
    expect(onUnauth).toHaveBeenCalledOnce();
  });

  it('shares one mint across concurrent 401s', async () => {
    let currentToken = 'stale';
    setTokenProvider(() => currentToken);
    setOnRefreshed((token) => {
      currentToken = token;
    });
    let mints = 0;
    setTokenMinter(async () => {
      mints += 1;
      await new Promise((resolve) => setTimeout(resolve, 20));
      return { access_token: 'fresh', expires_in: 300 };
    });

    server.use(
      http.get('http://localhost/api/users/me', ({ request }) =>
        request.headers.get('Authorization') === 'Bearer fresh'
          ? HttpResponse.json({ id: 'abc', username: 'u' })
          : HttpResponse.json({ detail: 'expired' }, { status: 401 })
      )
    );

    const results = await Promise.all([
      throwOnApiError(mainApi.GET('/api/users/me')),
      throwOnApiError(mainApi.GET('/api/users/me')),
    ]);

    expect(results).toHaveLength(2);
    expect(mints).toBe(1);
  });

  it('does not attempt recovery when the exchange endpoint itself 401s', async () => {
    const minter = mintOnce('fresh');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.post('http://localhost/api/oauth/session/exchange', () =>
        HttpResponse.json({ detail: 'no session' }, { status: 401 })
      )
    );

    await mainApi.POST('/api/oauth/session/exchange');

    expect(minter).not.toHaveBeenCalled();
    // A rejected exchange is "not signed in" and must reach the auth layer.
    expect(onUnauth).toHaveBeenCalledOnce();
  });
});

describe('throwOnApiError', () => {
  it('returns typed data on 2xx', async () => {
    server.use(
      http.get('http://localhost/api/users/me', () =>
        HttpResponse.json({ id: 'abc', username: 'u' })
      )
    );

    const user = await throwOnApiError(mainApi.GET('/api/users/me'));

    expect(user).toMatchObject({ id: 'abc', username: 'u' });
  });

  it('maps a JSON 4xx body to ApiError with status + detail', async () => {
    server.use(
      http.get('http://localhost/api/users/me', () =>
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
    server.use(
      http.get('http://localhost/api/users/me', () => HttpResponse.error())
    );

    await expect(
      throwOnApiError(mainApi.GET('/api/users/me'))
    ).rejects.toSatisfy(
      (err) => err instanceof ApiError && err.kind === 'network'
    );
  });

  it('maps a malformed JSON body to ApiError (no raw SyntaxError leaks)', async () => {
    server.use(
      http.get('http://localhost/api/users/me', () =>
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
