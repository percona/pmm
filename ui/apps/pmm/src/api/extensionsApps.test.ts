import axios from 'axios';
import type * as Axios from 'axios';
import type * as TokenStore from 'extensions/extensionsTokenStore';
import { getExtensionsApps } from './extensionsApps';
import {
  getExtensionsAuthState,
  getExtensionsToken,
} from 'extensions/extensionsTokenStore';

const { get, post } = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }));

vi.mock('axios', async (importOriginal) => {
  const actual = await importOriginal<typeof Axios>();
  return {
    default: {
      ...actual.default,
      // A real instance, so `@pmm-extensions/api` can still install its
      // interceptors, with the transport replaced.
      create: vi.fn((config) =>
        Object.assign(actual.default.create(config), { get, post })
      ),
    },
  };
});

vi.mock('extensions/extensionsTokenStore', async (importOriginal) => ({
  ...(await importOriginal<typeof TokenStore>()),
  getExtensionsToken: vi.fn(() => null),
}));

const heldToken = vi.mocked(getExtensionsToken);

const APPS = [{ app_key: 'atw' }];

beforeEach(() => {
  get.mockReset().mockResolvedValue({ data: APPS });
  post.mockReset().mockResolvedValue({
    data: { access_token: 'nav-bearer', expires_in: 300 },
  });
  heldToken.mockReturnValue(null);
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
    heldToken.mockReturnValue('page-bearer');

    await expect(getExtensionsApps()).resolves.toEqual(APPS);

    expect(post).not.toHaveBeenCalled();
    expect(get).toHaveBeenCalledWith('/apps/', {
      headers: { Authorization: 'Bearer page-bearer' },
    });
  });

  it('exchanges the session when no bearer is held', async () => {
    await getExtensionsApps();

    expect(post).toHaveBeenCalledOnce();
    expect(post).toHaveBeenCalledWith('/oauth/session/exchange');
    expect(get).toHaveBeenCalledWith('/apps/', {
      headers: { Authorization: 'Bearer nav-bearer' },
    });
  });

  it('leaves the token store untouched, so no renewal starts', async () => {
    await getExtensionsApps();

    expect(getExtensionsAuthState()).toEqual({ phase: 'idle', notice: null });
  });

  it('rejects without marking the session signed out', async () => {
    post.mockRejectedValue(
      Object.assign(new Error('unauthorized'), { response: { status: 401 } })
    );

    await expect(getExtensionsApps()).rejects.toThrow('unauthorized');

    expect(get).not.toHaveBeenCalled();
    expect(getExtensionsAuthState()).toEqual({ phase: 'idle', notice: null });
  });
});
