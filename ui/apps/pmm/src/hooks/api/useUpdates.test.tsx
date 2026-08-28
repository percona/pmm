import { renderHook, waitFor } from '@testing-library/react';
import { AxiosError } from 'axios';
import { useCheckUpdates } from './useUpdates';
import { wrapWithQueryProvider } from 'utils/testUtils';

const { checkForUpdates } = vi.hoisted(() => ({
  checkForUpdates: vi.fn(),
}));

vi.mock('api/updates', () => ({
  checkForUpdates,
  getChangeLogs: vi.fn(),
  startUpdate: vi.fn(),
}));

const installedOnly = {
  lastCheck: null,
  latest: null,
  installed: {
    version: '3.10.0',
    fullVersion: '3.10.0',
    timestamp: '2026-07-30T00:00:00Z',
  },
  latestNewsUrl: '',
  updateAvailable: false,
};

const errorWithStatus = (status: number, message = 'failed') => {
  const error = new AxiosError(message);
  error.response = { status } as AxiosError['response'];
  return error;
};

describe('useCheckUpdates', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('asks only for the installed version when updates are disabled', async () => {
    checkForUpdates.mockResolvedValue(installedOnly);

    const { result } = renderHook(
      () => useCheckUpdates({ onlyInstalledVersion: true }),
      { wrapper: ({ children }) => wrapWithQueryProvider(children) }
    );

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(checkForUpdates).toHaveBeenCalledTimes(1);
    expect(checkForUpdates).toHaveBeenCalledWith({
      force: false,
      onlyInstalledVersion: true,
    });
  });

  it('suppresses the notification only for a disabled-updates response', async () => {
    checkForUpdates
      .mockRejectedValueOnce(errorWithStatus(400, 'PMM updates are disabled'))
      .mockResolvedValueOnce(installedOnly);

    const { result } = renderHook(() => useCheckUpdates(), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    const [params, config] = checkForUpdates.mock.calls[0];
    expect(params).toEqual({ force: true });
    // 400 is the disabled case and stays quiet; a real failure must not
    expect(config.disableNotifications(errorWithStatus(400))).toBe(true);
    expect(config.disableNotifications(errorWithStatus(503))).toBe(false);
    expect(config.disableNotifications(errorWithStatus(500))).toBe(false);

    expect(checkForUpdates).toHaveBeenNthCalledWith(2, {
      force: true,
      onlyInstalledVersion: true,
    });
  });

  it('rethrows a 401 instead of falling back', async () => {
    const unauthorized = new AxiosError('unauthorized');
    unauthorized.response = { status: 401 } as AxiosError['response'];
    checkForUpdates.mockRejectedValue(unauthorized);

    const { result } = renderHook(() => useCheckUpdates(), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });

    await waitFor(() => expect(result.current.isError).toBe(true));

    expect(checkForUpdates).toHaveBeenCalledTimes(1);
  });
});
