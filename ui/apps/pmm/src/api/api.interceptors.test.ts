import { AxiosError, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  clearClientSession,
  establishClientSession,
} from 'contexts/auth/auth.clientSession';
import { api, grafanaApi } from './api';
import { ROTATE_TOKEN_URL, rotateToken } from './auth';
// registers the interceptors under test on the axios instances above
import './api.interceptors';

const { redirectToLogin, enqueueSnackbar } = vi.hoisted(() => ({
  redirectToLogin: vi.fn(),
  enqueueSnackbar: vi.fn(),
}));

vi.mock('contexts/auth/auth.utils', async (importOriginal) => ({
  ...(await importOriginal<typeof import('contexts/auth/auth.utils')>()),
  redirectToLogin,
}));

vi.mock('notistack', () => ({ enqueueSnackbar }));

interface StubbedResponse {
  status: number;
  data?: unknown;
}

const originalApiAdapter = api.defaults.adapter;
const originalGrafanaAdapter = grafanaApi.defaults.adapter;

const respond = (
  config: InternalAxiosRequestConfig,
  { status, data = {} }: StubbedResponse
) => {
  const response = {
    data,
    status,
    statusText: '',
    headers: {},
    config,
  } as AxiosResponse;

  if (status >= 400) {
    return Promise.reject(
      new AxiosError(
        `Request failed with status code ${status}`,
        String(status),
        config,
        {},
        response
      )
    );
  }

  return Promise.resolve(response);
};

/** Answers each successive request with the next stubbed response. */
const stubApi = (...responses: StubbedResponse[]) => {
  const calls: InternalAxiosRequestConfig[] = [];

  api.defaults.adapter = (config) => {
    calls.push(config);
    return respond(config, responses[calls.length - 1] ?? { status: 200 });
  };

  return calls;
};

/** Held closed until `release()` so concurrent rotations can be observed mid-flight. */
const stubRotate = (response: StubbedResponse = { status: 200 }) => {
  const calls: InternalAxiosRequestConfig[] = [];
  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });

  grafanaApi.defaults.adapter = async (config) => {
    calls.push(config);
    await gate;
    return respond(config, response);
  };

  return { calls, release: () => release() };
};

/** Serves the grafana instance, routing the rotation apart from the data requests. */
const stubGrafana = (
  dataResponses: StubbedResponse[],
  rotateResponse: StubbedResponse = { status: 200 }
) => {
  const rotateCalls: InternalAxiosRequestConfig[] = [];
  const dataCalls: InternalAxiosRequestConfig[] = [];

  grafanaApi.defaults.adapter = (config) => {
    if (config.url === ROTATE_TOKEN_URL) {
      rotateCalls.push(config);
      return respond(config, rotateResponse);
    }

    dataCalls.push(config);
    return respond(
      config,
      dataResponses[dataCalls.length - 1] ?? { status: 200 }
    );
  };

  return { rotateCalls, dataCalls };
};

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

describe('auth retry interceptor', () => {
  beforeEach(() => {
    establishClientSession();
  });

  afterEach(() => {
    api.defaults.adapter = originalApiAdapter;
    grafanaApi.defaults.adapter = originalGrafanaAdapter;
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('rotates the token and replays the request on 401', async () => {
    const calls = stubApi(
      { status: 401 },
      { status: 200, data: { nodes: [] } }
    );
    const rotate = stubRotate();
    rotate.release();

    await expect(api.get('/ha/nodes')).resolves.toMatchObject({ status: 200 });

    expect(calls).toHaveLength(2);
    expect(rotate.calls).toHaveLength(1);
    expect(rotate.calls[0].url).toBe(ROTATE_TOKEN_URL);
    expect(redirectToLogin).not.toHaveBeenCalled();
  });

  it('recovers a 401 on the grafana api too', async () => {
    const { rotateCalls, dataCalls } = stubGrafana([
      { status: 401 },
      { status: 200 },
    ]);

    await expect(grafanaApi.get('/folders')).resolves.toMatchObject({
      status: 200,
    });

    expect(dataCalls).toHaveLength(2);
    expect(rotateCalls).toHaveLength(1);
  });

  it('does not retry a 401 from the rotate endpoint itself', async () => {
    const { rotateCalls } = stubGrafana([], { status: 401 });

    await expect(rotateToken()).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(rotateCalls).toHaveLength(1);
  });

  it('rotates once for concurrent 401s', async () => {
    stubApi({ status: 401 }, { status: 401 }, { status: 200 }, { status: 200 });
    const rotate = stubRotate();

    const requests = Promise.all([api.get('/ha/nodes'), api.get('/ha/status')]);

    // both requests have failed and both handlers have asked for a rotation by now
    await flush();
    await flush();
    expect(rotate.calls).toHaveLength(1);

    rotate.release();

    await expect(requests).resolves.toHaveLength(2);
    expect(rotate.calls).toHaveLength(1);
  });

  it('redirects to login when the rotation itself is refused', async () => {
    const calls = stubApi({ status: 401 });
    const rotate = stubRotate({ status: 401 });
    rotate.release();

    await expect(api.get('/ha/nodes')).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(calls).toHaveLength(1);
    expect(redirectToLogin).toHaveBeenCalledTimes(1);
  });

  it('does not send an anonymous session to login', async () => {
    clearClientSession();
    stubApi({ status: 401 });
    const rotate = stubRotate({ status: 401 });
    rotate.release();

    await expect(api.get('/ha/nodes')).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(redirectToLogin).not.toHaveBeenCalled();
  });

  it('does not redirect or loop when the replay fails again', async () => {
    const calls = stubApi({ status: 401 }, { status: 401 });
    const rotate = stubRotate();
    rotate.release();

    await expect(api.get('/ha/nodes')).rejects.toMatchObject({
      response: { status: 401 },
    });

    expect(calls).toHaveLength(2);
    expect(rotate.calls).toHaveLength(1);
    expect(redirectToLogin).not.toHaveBeenCalled();
  });

  it('leaves non-401 failures alone', async () => {
    const calls = stubApi({ status: 500 });
    const rotate = stubRotate();
    rotate.release();

    await expect(api.get('/ha/nodes')).rejects.toMatchObject({
      response: { status: 500 },
    });

    expect(calls).toHaveLength(1);
    expect(rotate.calls).toHaveLength(0);
  });

  it('replays the request body without converting it twice', async () => {
    const calls = stubApi({ status: 401 }, { status: 200 });
    const rotate = stubRotate();
    rotate.release();

    await api.post('/server/settings', { pmmPublicAddress: 'pmm.example.com' });

    expect(calls).toHaveLength(2);
    expect(calls[0].data).toBe('{"pmm_public_address":"pmm.example.com"}');
    expect(calls[1].data).toBe(calls[0].data);
  });

  it('does not notify for a 401 it recovers from', async () => {
    stubApi({ status: 401 }, { status: 200 });
    const rotate = stubRotate();
    rotate.release();

    await api.get('/ha/nodes');

    expect(enqueueSnackbar).not.toHaveBeenCalled();
  });

  it('still notifies when the replay fails', async () => {
    stubApi({ status: 401 }, { status: 500, data: { message: 'boom' } });
    const rotate = stubRotate();
    rotate.release();

    await expect(api.get('/ha/nodes')).rejects.toMatchObject({
      response: { status: 500 },
    });

    expect(enqueueSnackbar).toHaveBeenCalledWith('boom', expect.anything());
  });
});
