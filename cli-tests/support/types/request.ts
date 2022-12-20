import { request } from '@playwright/test';

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

  constructor(username: string, password: string, port = 80) {
    this.username = username;
    this.password = password;
    this.port = port;
  }

  async context() {
    return request.newContext({
      baseURL: `http://localhost:${this.port}`,
      extraHTTPHeaders: {
        Authorization: `Basic ${encodeAuth(this.username, this.password)}`,
      },
    });
  }

  async doPost(path: string, data: unknown = {}) {
    const ctx = await this.context();

    return ctx.post(path, { data });
  }
}

export default PMMRestClient;
