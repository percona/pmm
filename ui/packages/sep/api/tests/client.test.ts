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
import { postSessionExchange } from '../src/auth';
import {
  apiClient,
  refreshAccessToken,
  setOnRefreshed,
  setOnUnauthorized,
  setTokenMinter,
  setTokenProvider,
} from '../src/client';
import { ApiError } from '../src/errors';
import { server } from './msw-server';

const BASE = 'http://localhost';

beforeEach(() => {
  // Axios default baseURL is '/api' which has no origin under Node.
  // Pin to a full URL so MSW can match requests predictably.
  apiClient.defaults.baseURL = `${BASE}/api`;
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
  setOnRefreshed(() => {});
});

afterEach(() => {
  setTokenProvider(() => null);
  setOnUnauthorized(() => {});
  setOnRefreshed(() => {});
  setTokenMinter(null);
});

describe('apiClient — Bearer token injection', () => {
  it('attaches Authorization header when provider returns a token', async () => {
    setTokenProvider(() => 'abc123');
    const seen = vi.fn();

    server.use(
      http.get(`${BASE}/api/ping`, ({ request }) => {
        seen(request.headers.get('Authorization'));
        return HttpResponse.json({ ok: true });
      })
    );

    const res = await apiClient.get('/ping');

    expect(res.status).toBe(200);
    expect(seen).toHaveBeenCalledWith('Bearer abc123');
  });

  it('omits Authorization header when provider returns null', async () => {
    setTokenProvider(() => null);
    const seen = vi.fn();

    server.use(
      http.get(`${BASE}/api/ping`, ({ request }) => {
        seen(request.headers.get('Authorization'));
        return HttpResponse.json({ ok: true });
      })
    );

    await apiClient.get('/ping');

    expect(seen).toHaveBeenCalledWith(null);
  });
});

describe('apiClient — payload pass-through (no case conversion)', () => {
  it('sends snake_case request bodies verbatim', async () => {
    const received = vi.fn();

    server.use(
      http.post(`${BASE}/api/widgets`, async ({ request }) => {
        received(await request.json());
        return HttpResponse.json({ ok: true });
      })
    );

    await apiClient.post('/widgets', { chunk_size: 1, replicate_check: true });

    expect(received).toHaveBeenCalledWith({
      chunk_size: 1,
      replicate_check: true,
    });
  });

  it('returns snake_case response bodies verbatim', async () => {
    server.use(
      http.get(`${BASE}/api/token`, () =>
        HttpResponse.json({
          access_token: 'tok',
          refresh_token: 'rtok',
          expires_in: 3600,
        })
      )
    );

    const { data } = await apiClient.get('/token');

    expect(data).toEqual({
      access_token: 'tok',
      refresh_token: 'rtok',
      expires_in: 3600,
    });
  });
});

describe('apiClient — error normalization', () => {
  it('maps a 404 response to ApiError with kind "http"', async () => {
    server.use(
      http.get(`${BASE}/api/missing`, () =>
        HttpResponse.json({ detail: 'not found' }, { status: 404 })
      )
    );

    await expect(apiClient.get('/missing')).rejects.toSatisfy((err) => {
      if (!(err instanceof ApiError)) {
        return false;
      }
      return (
        err.kind === 'http' && err.status === 404 && err.message === 'not found'
      );
    });
  });

  it('maps a 500 response to ApiError with kind "http"', async () => {
    server.use(
      http.get(`${BASE}/api/boom`, () =>
        HttpResponse.json({ detail: 'kaboom' }, { status: 500 })
      )
    );

    await expect(apiClient.get('/boom')).rejects.toSatisfy((err) => {
      if (!(err instanceof ApiError)) {
        return false;
      }
      return (
        err.kind === 'http' && err.status === 500 && err.message === 'kaboom'
      );
    });
  });

  it('maps a network error to ApiError with kind "network"', async () => {
    server.use(http.get(`${BASE}/api/offline`, () => HttpResponse.error()));

    await expect(apiClient.get('/offline')).rejects.toSatisfy((err) => {
      if (!(err instanceof ApiError)) {
        return false;
      }
      return err.kind === 'network';
    });
  });

  it('skips the unauthorized handler on refresh-endpoint 401', async () => {
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.post(`${BASE}/api/oauth/refresh`, () =>
        HttpResponse.json({ detail: 'bad refresh' }, { status: 401 })
      )
    );

    await expect(apiClient.post('/oauth/refresh')).rejects.toBeInstanceOf(
      ApiError
    );
    expect(onUnauth).not.toHaveBeenCalled();
  });
});

describe('apiClient — 401 refresh-retry', () => {
  it('retries the original request with the new token after a silent refresh', async () => {
    let currentToken = 'old';
    setTokenProvider(() => currentToken);
    const onRefreshed = vi.fn((token: string, _ttl: number) => {
      currentToken = token;
    });
    setOnRefreshed(onRefreshed);
    const seenAuth: Array<string | null> = [];

    server.use(
      http.get(`${BASE}/api/protected`, ({ request }) => {
        const auth = request.headers.get('Authorization');
        seenAuth.push(auth);
        if (auth === 'Bearer new') {
          return HttpResponse.json({ ok: true });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      }),
      http.post(`${BASE}/api/oauth/refresh`, () =>
        HttpResponse.json({ access_token: 'new', expires_in: 300 })
      )
    );

    const res = await apiClient.get('/protected');

    expect(res.status).toBe(200);
    expect(seenAuth).toEqual(['Bearer old', 'Bearer new']);
    expect(onRefreshed).toHaveBeenCalledWith('new', 300);
  });

  it('invokes the unauthorized handler when refresh itself fails', async () => {
    setTokenProvider(() => 'old');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.get(`${BASE}/api/protected`, () =>
        HttpResponse.json({ detail: 'expired' }, { status: 401 })
      ),
      http.post(`${BASE}/api/oauth/refresh`, () =>
        HttpResponse.json({ detail: 'no cookie' }, { status: 401 })
      )
    );

    await expect(apiClient.get('/protected')).rejects.toBeInstanceOf(ApiError);
    expect(onUnauth).toHaveBeenCalledOnce();
  });

  it('single-flights concurrent 401s into a single refresh call', async () => {
    let currentToken = 'old';
    setTokenProvider(() => currentToken);
    setOnRefreshed((token) => {
      currentToken = token;
    });

    let refreshCalls = 0;

    server.use(
      http.get(`${BASE}/api/a`, ({ request }) => {
        const auth = request.headers.get('Authorization');
        if (auth === 'Bearer new') {
          return HttpResponse.json({ r: 'a' });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      }),
      http.get(`${BASE}/api/b`, ({ request }) => {
        const auth = request.headers.get('Authorization');
        if (auth === 'Bearer new') {
          return HttpResponse.json({ r: 'b' });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      }),
      http.post(`${BASE}/api/oauth/refresh`, async () => {
        refreshCalls += 1;
        // Delay so both 401s are in-flight before refresh resolves.
        await new Promise((resolve) => setTimeout(resolve, 20));
        return HttpResponse.json({ access_token: 'new', expires_in: 300 });
      })
    );

    const [a, b] = await Promise.all([
      apiClient.get('/a'),
      apiClient.get('/b'),
    ]);

    expect(a.status).toBe(200);
    expect(b.status).toBe(200);
    expect(refreshCalls).toBe(1);
  });

  it('shares the single-flight coordinator between refreshAccessToken callers and 401 retries', async () => {
    let currentToken = 'old';
    setTokenProvider(() => currentToken);
    setOnRefreshed((token) => {
      currentToken = token;
    });
    let refreshCalls = 0;

    server.use(
      http.get(`${BASE}/api/protected`, ({ request }) => {
        const auth = request.headers.get('Authorization');
        if (auth === 'Bearer new') {
          return HttpResponse.json({ ok: true });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      }),
      http.post(`${BASE}/api/oauth/refresh`, async () => {
        refreshCalls += 1;
        await new Promise((resolve) => setTimeout(resolve, 20));
        return HttpResponse.json({ access_token: 'new', expires_in: 300 });
      })
    );

    // Simulates the timer and a concurrent protected request both needing
    // a fresh token at the same instant. Only one /oauth/refresh must fire.
    const [direct, retried] = await Promise.all([
      refreshAccessToken(),
      apiClient.get('/protected'),
    ]);

    expect(direct).toBe('new');
    expect(retried.status).toBe(200);
    expect(refreshCalls).toBe(1);
  });

  it('retries with the freshly-refreshed token even if the token provider was never updated', async () => {
    // Simulate a consumer that forgot to wire setOnRefreshed — the token
    // provider keeps returning the stale token. The retry must still
    // carry the new Bearer because the interceptor sets it explicitly.
    setTokenProvider(() => 'old');
    // Deliberately NOT calling setOnRefreshed.
    const seenAuth: Array<string | null> = [];

    server.use(
      http.get(`${BASE}/api/protected`, ({ request }) => {
        const auth = request.headers.get('Authorization');
        seenAuth.push(auth);
        if (auth === 'Bearer new') {
          return HttpResponse.json({ ok: true });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      }),
      http.post(`${BASE}/api/oauth/refresh`, () =>
        HttpResponse.json({ access_token: 'new', expires_in: 300 })
      )
    );

    const res = await apiClient.get('/protected');

    expect(res.status).toBe(200);
    expect(seenAuth).toEqual(['Bearer old', 'Bearer new']);
  });

  it('does not retry after a second 401 on the retried request', async () => {
    setTokenProvider(() => 'old');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);
    let refreshCalls = 0;

    server.use(
      http.get(`${BASE}/api/protected`, () =>
        HttpResponse.json({ detail: 'expired' }, { status: 401 })
      ),
      http.post(`${BASE}/api/oauth/refresh`, () => {
        refreshCalls += 1;
        return HttpResponse.json({ access_token: 'new', expires_in: 300 });
      })
    );

    await expect(apiClient.get('/protected')).rejects.toBeInstanceOf(ApiError);
    expect(refreshCalls).toBe(1);
    expect(onUnauth).toHaveBeenCalledOnce();
  });
});

describe('apiClient — pluggable token minter', () => {
  it('recovers through the registered minter instead of /oauth/refresh', async () => {
    let currentToken = 'old';
    setTokenProvider(() => currentToken);
    setOnRefreshed((token) => {
      currentToken = token;
    });
    // The embedded PMM wiring: exchange the host session cookie, no refresh
    // cookie exists. `/oauth/refresh` is deliberately left unhandled — MSW is
    // configured to error on unhandled requests, so reaching it fails the test.
    setTokenMinter(() => postSessionExchange());
    let exchanges = 0;

    server.use(
      http.get(`${BASE}/api/protected`, ({ request }) => {
        if (request.headers.get('Authorization') === 'Bearer minted') {
          return HttpResponse.json({ ok: true });
        }
        return HttpResponse.json({ detail: 'expired' }, { status: 401 });
      }),
      http.post(`${BASE}/api/oauth/session/exchange`, () => {
        exchanges += 1;
        return HttpResponse.json({ access_token: 'minted', expires_in: 300 });
      })
    );

    const res = await apiClient.get('/protected');

    expect(res.status).toBe(200);
    expect(exchanges).toBe(1);
    expect(currentToken).toBe('minted');
  });

  it('coalesces a burst of 401s into one exchange', async () => {
    let currentToken = 'old';
    setTokenProvider(() => currentToken);
    setOnRefreshed((token) => {
      currentToken = token;
    });
    setTokenMinter(() => postSessionExchange());
    let exchanges = 0;

    const protectedHandler = ({ request }: { request: Request }) =>
      request.headers.get('Authorization') === 'Bearer minted'
        ? HttpResponse.json({ ok: true })
        : HttpResponse.json({ detail: 'expired' }, { status: 401 });

    server.use(
      http.get(`${BASE}/api/a`, protectedHandler),
      http.get(`${BASE}/api/b`, protectedHandler),
      http.get(`${BASE}/api/c`, protectedHandler),
      http.post(`${BASE}/api/oauth/session/exchange`, async () => {
        exchanges += 1;
        // Hold the exchange open so all three 401s land while it is in flight.
        await new Promise((resolve) => setTimeout(resolve, 20));
        return HttpResponse.json({ access_token: 'minted', expires_in: 300 });
      })
    );

    const responses = await Promise.all([
      apiClient.get('/a'),
      apiClient.get('/b'),
      apiClient.get('/c'),
    ]);

    expect(responses.map((r) => r.status)).toEqual([200, 200, 200]);
    expect(exchanges).toBe(1);
  });

  it('does not re-enter the retry path when the exchange itself 401s', async () => {
    // Regression guard: the mint is single-flighted, so routing its own 401
    // back through the retry interceptor would hand that interceptor the very
    // promise it is running inside — an await on itself that never settles.
    // A hang here surfaces as a test timeout, not a failed assertion.
    setTokenProvider(() => 'old');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);
    setTokenMinter(() => postSessionExchange());
    let exchanges = 0;

    server.use(
      http.get(`${BASE}/api/protected`, () =>
        HttpResponse.json({ detail: 'expired' }, { status: 401 })
      ),
      http.post(`${BASE}/api/oauth/session/exchange`, () => {
        exchanges += 1;
        return HttpResponse.json({ detail: 'no session' }, { status: 401 });
      })
    );

    await expect(apiClient.get('/protected')).rejects.toBeInstanceOf(ApiError);
    expect(exchanges).toBe(1);
    // Once for the rejected exchange, once for the unrecoverable request.
    expect(onUnauth).toHaveBeenCalledTimes(2);
  });

  it('notifies the auth layer when the exchange endpoint rejects the session', async () => {
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);

    server.use(
      http.post(`${BASE}/api/oauth/session/exchange`, () =>
        HttpResponse.json({ detail: 'no session' }, { status: 401 })
      )
    );

    await expect(postSessionExchange()).rejects.toBeInstanceOf(ApiError);
    expect(onUnauth).toHaveBeenCalledOnce();
  });

  it('treats a minter resolving null as a failed mint', async () => {
    setTokenProvider(() => 'old');
    const onUnauth = vi.fn();
    setOnUnauthorized(onUnauth);
    const minter = vi.fn(async () => null);
    setTokenMinter(minter);

    server.use(
      http.get(`${BASE}/api/protected`, () =>
        HttpResponse.json({ detail: 'expired' }, { status: 401 })
      )
    );

    await expect(apiClient.get('/protected')).rejects.toBeInstanceOf(ApiError);
    expect(minter).toHaveBeenCalledOnce();
    expect(onUnauth).toHaveBeenCalledOnce();
  });

  it('restores the default /oauth/refresh minter when passed null', async () => {
    setTokenMinter(async () => ({ access_token: 'custom', expires_in: 1 }));
    setTokenMinter(null);

    server.use(
      http.post(`${BASE}/api/oauth/refresh`, () =>
        HttpResponse.json({ access_token: 'from-cookie', expires_in: 300 })
      )
    );

    await expect(refreshAccessToken()).resolves.toBe('from-cookie');
  });
});
