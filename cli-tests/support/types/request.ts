import { APIRequest, expect, request } from '@playwright/test';
import PromiseRetry from 'promise-retry';

type PMMRequest = {
  port?: number,
  username?: string,
  password?: string,
  data?: unknown
};

const encodeAuth = (username: string, password: string) => {
  return Buffer.from(`${username}:${password}`).toString(
    'base64',
  );
};

export const pmmRequest = async (path: string, opts?: PMMRequest) => {
  const {
    password = 'admin', username = 'admin', port = 80, data = {},
  } = opts;
  const ctx = await request.newContext({
    extraHTTPHeaders: {
      Authorization: `Basic ${encodeAuth(username, password)}`,
    },
  });

  return ctx.post(`http://localhost:${port}${path}`, { data });
};

class PMMRestClient {
  username: string;
  password: string;
  port: number;
  requestOpts: Parameters<APIRequest['newContext']>[0];

  constructor(username: string, password: string, port = 80, requestOpts: Parameters<APIRequest['newContext']>[0] = {}) {
    this.username = username;
    this.password = password;
    this.port = port;
    this.requestOpts = requestOpts
  }

  async context() {
    return request.newContext({
      baseURL: `http://localhost:${this.port}`,
      extraHTTPHeaders: {
        Authorization: `Basic ${encodeAuth(this.username, this.password)}`,
      },
      ...this.requestOpts,
    });
  }

  async doPost(path: string, data: unknown = {}) {
    const ctx = await this.context();

    return ctx.post(path, { data });
  }

  async works() {
    await PromiseRetry(async retry => {
      const resp = await this.doPost('/v1/Settings/Get').catch(err => retry(err))
      const respBody = await resp.json().catch(err => retry(err))

      try {
        expect(resp.ok()).toBeTruthy()
        expect(respBody).toHaveProperty('settings')
      } catch(err) {
        return retry(err)
      }
    }, {
      retries: 30,
      minTimeout: 1000,
      maxTimeout: 1000,
    })
  }
}

export default PMMRestClient;
