import { render, waitFor } from '@testing-library/react';
import { AxiosError, type InternalAxiosRequestConfig } from 'axios';
import { UpdatesProvider } from './updates.provider';
import {
  api,
  addApiErrorInterceptor,
  removeApiErrorInterceptor,
} from 'api/api';
import {
  wrapWithQueryProvider,
  wrapWithSettings,
  wrapWithUserProvider,
} from 'utils/testUtils';

const { enqueueSnackbar } = vi.hoisted(() => ({
  enqueueSnackbar: vi.fn(),
}));

vi.mock('notistack', () => ({
  enqueueSnackbar,
}));

// captured verbatim from a live PMM 3.8.0 server with updates disabled
const DISABLED_ERROR_BODY = {
  error: 'PMM updates are disabled',
  code: 9,
  message: 'PMM updates are disabled',
  details: [],
};

const INSTALLED_ONLY_BODY = {
  installed: {
    version: '3.8.0',
    full_version: '3.8.0',
    timestamp: '2026-05-15T10:11:22Z',
  },
  latest: null,
  update_available: false,
  latest_news_url: '',
  last_check: null,
};

let seenParams: Record<string, unknown>[] = [];

/** Stands in for pmm-managed: refuses a full check, answers the installed-only one. */
const updatesDisabledAdapter = async (config: InternalAxiosRequestConfig) => {
  const params = (config.params ?? {}) as Record<string, unknown>;

  if (!config.url?.includes('/server/updates')) {
    return { data: {}, status: 200, statusText: 'OK', headers: {}, config };
  }

  seenParams.push(params);

  if (params.only_installed_version) {
    return {
      data: INSTALLED_ONLY_BODY,
      status: 200,
      statusText: 'OK',
      headers: {},
      config,
    };
  }

  throw new AxiosError('Request failed', '400', config, {}, {
    data: DISABLED_ERROR_BODY,
    status: 400,
    statusText: 'Bad Request',
    headers: {},
    config,
  } as AxiosError['response']);
};

const renderProvider = (updatesEnabled: boolean) =>
  render(
    wrapWithQueryProvider(
      wrapWithUserProvider(
        wrapWithSettings(
          <UpdatesProvider>
            <div>content</div>
          </UpdatesProvider>,
          { settings: { updatesEnabled } }
        )
      )
    )
  );

describe('UpdatesProvider (PMM-15274)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    seenParams = [];
    api.defaults.adapter = updatesDisabledAdapter;
    addApiErrorInterceptor();
  });

  afterEach(() => {
    removeApiErrorInterceptor();
  });

  it('raises no error toast when updates are disabled', async () => {
    renderProvider(false);

    await waitFor(() => expect(seenParams.length).toBeGreaterThan(0));

    expect(enqueueSnackbar).not.toHaveBeenCalled();
  });

  it('never asks for a full check when updates are disabled', async () => {
    renderProvider(false);

    await waitFor(() => expect(seenParams.length).toBeGreaterThan(0));

    expect(seenParams).toEqual([
      { force: false, only_installed_version: true },
    ]);
  });

  it('still surfaces a genuine failure when updates are enabled', async () => {
    // nothing recovers this one, so the user must be told
    api.defaults.adapter = async (config: InternalAxiosRequestConfig) => {
      if (!config.url?.includes('/server/updates')) {
        return { data: {}, status: 200, statusText: 'OK', headers: {}, config };
      }
      seenParams.push((config.params ?? {}) as Record<string, unknown>);
      throw new AxiosError('Request failed', '503', config, {}, {
        data: { message: 'failed to check for updates' },
        status: 503,
        statusText: 'Service Unavailable',
        headers: {},
        config,
      } as AxiosError['response']);
    };

    renderProvider(true);

    await waitFor(() => expect(enqueueSnackbar).toHaveBeenCalled());

    // the full check is attempted, fails quietly, then the fallback reports
    expect(seenParams[0]).toEqual({ force: true });
    expect(seenParams[1]).toEqual({
      force: true,
      only_installed_version: true,
    });
    expect(enqueueSnackbar).toHaveBeenCalledTimes(1);
    expect(enqueueSnackbar).toHaveBeenCalledWith(
      'failed to check for updates',
      expect.objectContaining({ variant: 'error' })
    );
  });
});
