import { APIRequestContext, request } from '@playwright/test';
import { APIResponse } from 'playwright';

const encodeAuth = (username: string, password: string) => {
  return Buffer.from(`${username}:${password}`).toString(
    'base64',
  );
};

/**
 * Api Client with implemented HTTP(S) requests methods.
 */
class PmmRestClient {
  username: string;
  password: string;
  baseURL: string;

  constructor(username: string, password: string, port = 80, protocol = 'http') {
    this.username = username;
    this.password = password;
    this.baseURL = `${protocol}://localhost:${port}`;
  }

  async context(): Promise<APIRequestContext> {
    return request.newContext({
      baseURL: this.baseURL,
      extraHTTPHeaders: {
        Authorization: `Basic ${encodeAuth(this.username, this.password)}`,
      },
      ignoreHTTPSErrors: true,
    });
  }

  /**
   * Implements HTTP(S) POST to PMM Server API
   *
   * @param   path      API endpoint path
   * @param   payload   request body {@code Object}
   * @return            Promise<APIResponse> instance
   */
  async post(path: string, payload: unknown = {}): Promise<APIResponse> {
    console.log(`POST: ${this.baseURL}${path}\nPayload: ${JSON.stringify(payload)}`);
    const response = await (await this.context()).post(path, payload);
    console.log(`Status: ${response.status()} ${response.statusText()}`);
    return response;
  }
}
export default PmmRestClient;
