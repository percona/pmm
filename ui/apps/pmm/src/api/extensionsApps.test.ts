import axios, {
  AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';
import type * as Axios from 'axios';
import { getExtensionsApps } from './extensionsApps';
import { initExtensionsAuth } from 'extensions/bootstrap';
import {
  getExtensionsAuthState,
  recordExtensionsToken,
  resetExtensionsAuthStore,
} from 'extensions/extensionsTokenStore';

// Every axios instance, `@pmm-extensions/api`'s `apiClient` included, sends
// through this adapter, so the interceptors installed on them really run.
const { adapter } = vi.hoisted(() => ({ adapter: vi.fn() }));

vi.mock('axios', async (importOriginal) => {
  const actual = await importOriginal<typeof Axios>();
  return {
    ...actual,
    default: {
      ...actual.default,
      create: vi.fn((config) => actual.default.create({ ...config, adapter })),
    },
  };
});

const APPS = [{ app_key: 'atw' }];

const respond =
  (routes: Record<string, number>) =>
  async (config: InternalAxiosRequestConfig): Promise<AxiosResponse> => {
    const status = routes[config.url ?? ''] ?? 404;
    const response = {
      data:
        config.url === '/apps/'
          ? APPS
          : { access_token: 'nav-bearer', expires_in: 300 },
      status,
      statusText: '',
      headers: {},
      config,
    };
    if (status >= 400) {
      throw new AxiosError(
        `Request failed with status code ${status}`,
        'ERR_BAD_REQUEST',
        config,
        null,
        response
      );
    }
    return response;
  };

const requests = () =>
  adapter.mock.calls.map(([config]) => ({
    url: config.url,
    authorization: config.headers.Authorization,
  }));

beforeEach(() => {
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'Date'] });
  adapter.mockReset();
  resetExtensionsAuthStore();
  initExtensionsAuth();
});

afterEach(() => {
  resetExtensionsAuthStore();
  vi.useRealTimers();
});

describe('getExtensionsApps', () => {
  it('talks to the side-car under the Extensions mount, with the session cookie', () => {
    // Exactly this config, which no other client shares: `apiClient` also sets
    // `validateStatus`.
    expect(axios.create).toHaveBeenCalledWith({
      baseURL: '/extensions/api',
      withCredentials: true,
      headers: { 'Content-Type': 'application/json' },
    });
  });

  it('reuses the bearer an open Extensions page holds', async () => {
    recordExtensionsToken('page-bearer', 300);
    adapter.mockImplementation(respond({ '/apps/': 200 }));

    await expect(getExtensionsApps()).resolves.toEqual(APPS);

    expect(requests()).toEqual([
      { url: '/apps/', authorization: 'Bearer page-bearer' },
    ]);
  });

  it('exchanges the session when no bearer is held', async () => {
    adapter.mockImplementation(
      respond({ '/oauth/session/exchange': 200, '/apps/': 200 })
    );

    await expect(getExtensionsApps()).resolves.toEqual(APPS);

    expect(requests()).toEqual([
      { url: '/oauth/session/exchange', authorization: undefined },
      { url: '/apps/', authorization: 'Bearer nav-bearer' },
    ]);
  });

  it('records no bearer, so no renewal ever starts', async () => {
    adapter.mockImplementation(
      respond({ '/oauth/session/exchange': 200, '/apps/': 200 })
    );

    await getExtensionsApps();
    await vi.advanceTimersByTimeAsync(600_000);

    expect(getExtensionsAuthState()).toEqual({ phase: 'idle', notice: null });
    expect(adapter).toHaveBeenCalledTimes(2);
  });

  it.each([
    ['the exchange', { '/oauth/session/exchange': 401 }],
    ['the list', { '/oauth/session/exchange': 200, '/apps/': 401 }],
  ])(
    'rejects without signing the user out when %s is refused',
    async (_, routes) => {
      adapter.mockImplementation(respond(routes));

      await expect(getExtensionsApps()).rejects.toBeInstanceOf(AxiosError);
      await vi.advanceTimersByTimeAsync(600_000);

      expect(getExtensionsAuthState()).toEqual({
        phase: 'idle',
        notice: null,
      });
      // No mint and no retry behind the nav's back.
      expect(adapter).toHaveBeenCalledTimes(Object.keys(routes).length);
    }
  );
});
