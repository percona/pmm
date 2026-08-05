import {
  ApiError,
  getToken,
  postSessionExchange,
  setTokenMinter,
} from '@sep/api';
import { initSepAuth } from './bootstrap';
import {
  ensureSepToken,
  getSepAuthStatus,
  getSepToken,
  resetSepAuthStore,
  retrySepAuth,
} from './sepTokenStore';

// Mock only the network boundary. `refreshAccessToken`'s single-flight, the
// token-minter seam, and the unauthorized wiring stay real, so these exercise
// the store against the coordinator it actually runs against.
vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  postSessionExchange: vi.fn(),
}));

const exchange = vi.mocked(postSessionExchange);

const TTL_SECONDS = 300;
const mintedToken = (accessToken: string) => ({
  access_token: accessToken,
  expires_in: TTL_SECONDS,
});

const unauthorized = () =>
  new ApiError({ kind: 'http', status: 401, message: 'no session' });

beforeEach(() => {
  // Leave `queueMicrotask` real: `refreshAccessToken` clears its single-flight
  // slot in a microtask, and faking that would deadlock the second exchange.
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout', 'Date'] });
  exchange.mockReset();
  resetSepAuthStore();
  initSepAuth();
});

afterEach(() => {
  resetSepAuthStore();
  setTokenMinter(null);
  vi.useRealTimers();
});

describe('sepTokenStore — acquiring a bearer', () => {
  it('holds no token until an exchange runs', () => {
    expect(getSepToken()).toBeNull();
    expect(getSepAuthStatus()).toBe('idle');
    expect(exchange).not.toHaveBeenCalled();
  });

  it('exchanges once and exposes the bearer synchronously', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await expect(ensureSepToken()).resolves.toBe(true);

    expect(exchange).toHaveBeenCalledOnce();
    expect(getSepToken()).toBe('bearer-1');
    expect(getSepAuthStatus()).toBe('ready');
  });

  it('serves the bearer through the token provider registered on @sep/api', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await ensureSepToken();

    expect(getToken()).toBe('bearer-1');
  });

  it('reuses the held bearer instead of exchanging again', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await ensureSepToken();
    await ensureSepToken();

    expect(exchange).toHaveBeenCalledOnce();
  });

  it('coalesces concurrent callers into one exchange', async () => {
    let resolveExchange: (
      value: ReturnType<typeof mintedToken>
    ) => void = () => {};
    exchange.mockReturnValue(
      new Promise((resolve) => {
        resolveExchange = resolve;
      })
    );

    const pending = Promise.all([
      ensureSepToken(),
      ensureSepToken(),
      ensureSepToken(),
    ]);
    resolveExchange(mintedToken('bearer-1'));

    await expect(pending).resolves.toEqual([true, true, true]);
    expect(exchange).toHaveBeenCalledOnce();
  });

  it('never writes the bearer to web storage', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await ensureSepToken();

    expect(Object.keys(localStorage)).toHaveLength(0);
    expect(Object.keys(sessionStorage)).toHaveLength(0);
  });
});

describe('sepTokenStore — lifetime', () => {
  it('stops serving the bearer once it has expired', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));
    await ensureSepToken();

    vi.setSystemTime(Date.now() + TTL_SECONDS * 1000 + 1);

    expect(getSepToken()).toBeNull();
  });

  it('re-exchanges shortly before expiry', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));
    await ensureSepToken();
    exchange.mockResolvedValue(mintedToken('bearer-2'));

    // 30s of skew ahead of the 300s TTL.
    await vi.advanceTimersByTimeAsync(270_000);

    expect(exchange).toHaveBeenCalledTimes(2);
    expect(getSepToken()).toBe('bearer-2');
    expect(getSepAuthStatus()).toBe('ready');
  });

  it('keeps renewing across successive lifetimes', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));
    await ensureSepToken();
    exchange.mockResolvedValue(mintedToken('bearer-2'));
    await vi.advanceTimersByTimeAsync(270_000);
    exchange.mockResolvedValue(mintedToken('bearer-3'));
    await vi.advanceTimersByTimeAsync(270_000);

    expect(exchange).toHaveBeenCalledTimes(3);
    expect(getSepToken()).toBe('bearer-3');
  });

  it('drops the bearer on a failed renewal without tearing down the page', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));
    await ensureSepToken();
    exchange.mockRejectedValue(new Error('offline'));

    await vi.advanceTimersByTimeAsync(270_000);

    expect(getSepToken()).toBeNull();
    // Still `ready`, so a mounted SEP page keeps its state; the next request
    // 401s and mints through the transports' retry.
    expect(getSepAuthStatus()).toBe('ready');
  });

  it('re-acquires on the next visit after a failed renewal', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));
    await ensureSepToken();
    exchange.mockRejectedValue(new Error('offline'));
    await vi.advanceTimersByTimeAsync(270_000);

    exchange.mockResolvedValue(mintedToken('bearer-2'));
    await expect(ensureSepToken()).resolves.toBe(true);

    expect(getSepToken()).toBe('bearer-2');
  });

  it('stops renewing once the store is cleared', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));
    await ensureSepToken();

    resetSepAuthStore();
    await vi.advanceTimersByTimeAsync(270_000);

    expect(exchange).toHaveBeenCalledOnce();
  });
});

describe('sepTokenStore — rejected session', () => {
  it('treats a 401 from the exchange as signed out', async () => {
    exchange.mockRejectedValue(unauthorized());

    await expect(ensureSepToken()).resolves.toBe(false);

    expect(getSepAuthStatus()).toBe('signedOut');
    expect(getSepToken()).toBeNull();
  });

  it('refuses to exchange again while signed out', async () => {
    exchange.mockRejectedValue(unauthorized());
    await ensureSepToken();

    await expect(ensureSepToken()).resolves.toBe(false);
    await expect(ensureSepToken()).resolves.toBe(false);

    expect(exchange).toHaveBeenCalledOnce();
  });

  it('exchanges again only when the user explicitly retries', async () => {
    exchange.mockRejectedValue(unauthorized());
    await ensureSepToken();
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await expect(retrySepAuth()).resolves.toBe(true);

    expect(exchange).toHaveBeenCalledTimes(2);
    expect(getSepToken()).toBe('bearer-1');
  });

  it('does not schedule a renewal after a rejected session', async () => {
    exchange.mockRejectedValue(unauthorized());
    await ensureSepToken();

    await vi.advanceTimersByTimeAsync(600_000);

    expect(exchange).toHaveBeenCalledOnce();
  });
});

describe('sepTokenStore — transient failure', () => {
  it('reports an error without going sticky', async () => {
    exchange.mockRejectedValue(new Error('network down'));

    await expect(ensureSepToken()).resolves.toBe(false);

    expect(getSepAuthStatus()).toBe('error');
  });

  it('retries on the next attempt', async () => {
    exchange.mockRejectedValue(new Error('network down'));
    await ensureSepToken();
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await expect(ensureSepToken()).resolves.toBe(true);

    expect(exchange).toHaveBeenCalledTimes(2);
    expect(getSepToken()).toBe('bearer-1');
  });
});
