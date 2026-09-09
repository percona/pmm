import {
  ApiError,
  getToken,
  postSessionExchange,
  setTokenMinter,
} from '@sep/api';
import { initSepAuth } from './bootstrap';
import {
  ensureSepToken,
  getSepAuthState,
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
/** The renewal fires 30s before the 300s TTL. */
const UNTIL_RENEWAL_MS = 270_000;
/** Backoff is 2s, 4s, 8s, 16s; this clears all four plus slack. */
const PAST_ALL_RETRIES_MS = 60_000;

const mintedToken = (accessToken: string) => ({
  access_token: accessToken,
  expires_in: TTL_SECONDS,
});

const unauthorized = () =>
  new ApiError({ kind: 'http', status: 401, message: 'no session' });

const phase = () => getSepAuthState().phase;
const notice = () => getSepAuthState().notice;

/** Reach `ready` with a live bearer, as a mounted SEP page would be. */
const becomeReady = async (accessToken = 'bearer-1') => {
  exchange.mockResolvedValue(mintedToken(accessToken));
  await ensureSepToken();
  exchange.mockReset();
};

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
    expect(phase()).toBe('idle');
    expect(exchange).not.toHaveBeenCalled();
  });

  it('exchanges once and exposes the bearer synchronously', async () => {
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await expect(ensureSepToken()).resolves.toBe(true);

    expect(exchange).toHaveBeenCalledOnce();
    expect(getSepToken()).toBe('bearer-1');
    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: null });
  });

  it('serves the bearer through the token provider registered on @sep/api', async () => {
    await becomeReady();

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
    await becomeReady();

    expect(Object.keys(localStorage)).toHaveLength(0);
    expect(Object.keys(sessionStorage)).toHaveLength(0);
  });

  it('hands out a stable snapshot so subscribers do not re-render on no-ops', async () => {
    await becomeReady();
    const first = getSepAuthState();

    await ensureSepToken();

    expect(getSepAuthState()).toBe(first);
  });
});

describe('sepTokenStore — failing closed', () => {
  it('serves no token once the bearer has expired', async () => {
    await becomeReady();

    vi.setSystemTime(Date.now() + TTL_SECONDS * 1000 + 1);

    expect(getSepToken()).toBeNull();
    expect(getToken()).toBeNull();
  });

  it('drops the bearer when a renewal is rejected', async () => {
    await becomeReady();
    exchange.mockRejectedValue(unauthorized());

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(getSepToken()).toBeNull();
  });

  it('drops the bearer when a renewal cannot complete', async () => {
    await becomeReady();
    exchange.mockRejectedValue(new Error('offline'));

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(getSepToken()).toBeNull();
  });

  it('refuses to exchange again once the session is rejected', async () => {
    exchange.mockRejectedValue(unauthorized());
    await ensureSepToken();

    await expect(ensureSepToken()).resolves.toBe(false);
    await expect(ensureSepToken()).resolves.toBe(false);

    expect(exchange).toHaveBeenCalledOnce();
  });

  it('stops renewing after the session is rejected', async () => {
    exchange.mockRejectedValue(unauthorized());
    await ensureSepToken();

    await vi.advanceTimersByTimeAsync(600_000);

    expect(exchange).toHaveBeenCalledOnce();
  });
});

describe('sepTokenStore — bootstrap failure', () => {
  it('shows a signed-out page when the session is rejected at load', async () => {
    exchange.mockRejectedValue(unauthorized());

    await expect(ensureSepToken()).resolves.toBe(false);

    expect(getSepAuthState()).toEqual({ phase: 'signedOut', notice: null });
  });

  it('shows an unreachable page when the exchange cannot complete at load', async () => {
    exchange.mockRejectedValue(new Error('network down'));

    await expect(ensureSepToken()).resolves.toBe(false);

    expect(getSepAuthState()).toEqual({ phase: 'unreachable', notice: null });
  });

  it('recovers on an explicit retry', async () => {
    exchange.mockRejectedValue(unauthorized());
    await ensureSepToken();
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await expect(retrySepAuth()).resolves.toBe(true);

    expect(exchange).toHaveBeenCalledTimes(2);
    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: null });
  });

  it('retries a transient bootstrap failure on the next visit', async () => {
    exchange.mockRejectedValue(new Error('network down'));
    await ensureSepToken();
    exchange.mockResolvedValue(mintedToken('bearer-1'));

    await expect(ensureSepToken()).resolves.toBe(true);

    expect(getSepToken()).toBe('bearer-1');
  });
});

describe('sepTokenStore — renewal on a mounted page', () => {
  it('renews shortly before expiry', async () => {
    await becomeReady();
    exchange.mockResolvedValue(mintedToken('bearer-2'));

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(exchange).toHaveBeenCalledOnce();
    expect(getSepToken()).toBe('bearer-2');
    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: null });
  });

  it('keeps renewing across successive lifetimes', async () => {
    await becomeReady();
    exchange.mockResolvedValue(mintedToken('bearer-2'));
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);
    exchange.mockResolvedValue(mintedToken('bearer-3'));
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(exchange).toHaveBeenCalledTimes(2);
    expect(getSepToken()).toBe('bearer-3');
  });

  it('retries a transient renewal failure quietly, without a notice', async () => {
    await becomeReady();
    exchange.mockRejectedValue(new Error('offline'));

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(exchange).toHaveBeenCalledOnce();
    // Still `ready` with nothing on screen: a blip must not interrupt the user.
    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: null });
  });

  it('backs off across several quiet attempts before giving up', async () => {
    await becomeReady();
    exchange.mockRejectedValue(new Error('offline'));
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    await vi.advanceTimersByTimeAsync(2_000);
    expect(exchange).toHaveBeenCalledTimes(2);
    expect(notice()).toBeNull();

    await vi.advanceTimersByTimeAsync(4_000);
    expect(exchange).toHaveBeenCalledTimes(3);
    expect(notice()).toBeNull();

    await vi.advanceTimersByTimeAsync(8_000);
    expect(exchange).toHaveBeenCalledTimes(4);
    expect(notice()).toBeNull();
  });

  it('surfaces a transient failure only once it persists', async () => {
    await becomeReady();
    exchange.mockRejectedValue(new Error('offline'));

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS + PAST_ALL_RETRIES_MS);

    expect(exchange).toHaveBeenCalledTimes(1 + 4);
    // Reported beside the page, never instead of it.
    expect(getSepAuthState()).toEqual({
      phase: 'ready',
      notice: 'unreachable',
    });
  });

  it('recovers silently when a backoff attempt succeeds', async () => {
    await becomeReady();
    exchange.mockRejectedValueOnce(new Error('offline'));
    exchange.mockResolvedValue(mintedToken('bearer-2'));

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS + 2_000);

    expect(getSepToken()).toBe('bearer-2');
    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: null });
  });

  it('stops retrying once a backoff attempt succeeds', async () => {
    await becomeReady();
    exchange.mockRejectedValueOnce(new Error('offline'));
    exchange.mockResolvedValue(mintedToken('bearer-2'));
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS + 2_000);

    // Only the next scheduled renewal should fire, not a leftover backoff.
    await vi.advanceTimersByTimeAsync(1_000);

    expect(exchange).toHaveBeenCalledTimes(2);
  });

  it('reports a rejected session at once, without backing off', async () => {
    await becomeReady();
    exchange.mockRejectedValue(unauthorized());

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: 'signedOut' });
    // Terminal: retrying would only repeat the rejection.
    await vi.advanceTimersByTimeAsync(PAST_ALL_RETRIES_MS);
    expect(exchange).toHaveBeenCalledOnce();
  });

  it('never leaves the ready phase, whatever the failure', async () => {
    await becomeReady();
    exchange.mockRejectedValue(unauthorized());

    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS + PAST_ALL_RETRIES_MS);

    expect(phase()).toBe('ready');
  });

  it('clears the notice when the user retries successfully', async () => {
    await becomeReady();
    exchange.mockRejectedValue(unauthorized());
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);
    expect(notice()).toBe('signedOut');

    exchange.mockResolvedValue(mintedToken('bearer-2'));
    await expect(retrySepAuth()).resolves.toBe(true);

    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: null });
    expect(getSepToken()).toBe('bearer-2');
  });

  it('keeps the notice when the retry fails again', async () => {
    await becomeReady();
    exchange.mockRejectedValue(unauthorized());
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    await expect(retrySepAuth()).resolves.toBe(false);

    expect(getSepAuthState()).toEqual({ phase: 'ready', notice: 'signedOut' });
  });

  it('stops renewing once the store is cleared', async () => {
    await becomeReady();
    exchange.mockResolvedValue(mintedToken('bearer-2'));

    resetSepAuthStore();
    await vi.advanceTimersByTimeAsync(UNTIL_RENEWAL_MS);

    expect(exchange).not.toHaveBeenCalled();
  });
});
