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

const failedPrecondition = () => {
  const error = new AxiosError('PMM updates are disabled');
  error.response = { status: 400 } as AxiosError['response'];
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

  it('does not notify on the full check it recovers from', async () => {
    checkForUpdates
      .mockRejectedValueOnce(failedPrecondition())
      .mockResolvedValueOnce(installedOnly);

    const { result } = renderHook(() => useCheckUpdates(), {
      wrapper: ({ children }) => wrapWithQueryProvider(children),
    });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(checkForUpdates).toHaveBeenNthCalledWith(
      1,
      { force: true },
      { disableNotifications: true }
    );
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
