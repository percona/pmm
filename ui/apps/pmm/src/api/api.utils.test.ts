import {
  AxiosError,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';
import { isNotifiedError } from './api.utils';

const failed = (
  status?: number,
  config: Partial<InternalAxiosRequestConfig> = {}
) =>
  new AxiosError(
    'failed',
    status ? String(status) : 'ERR_NETWORK',
    config as InternalAxiosRequestConfig,
    {},
    status ? ({ status, data: {} } as AxiosResponse) : undefined
  );

describe('isNotifiedError', () => {
  it('matches the errors the notifier reports', () => {
    expect(isNotifiedError(failed(400))).toBe(true);
    expect(isNotifiedError(failed(500))).toBe(true);
    expect(isNotifiedError(failed(429, { disableNotifications: false }))).toBe(
      true
    );
  });

  it('leaves out errors the notifier stays quiet about', () => {
    expect(isNotifiedError(failed())).toBe(false);
    expect(isNotifiedError(failed(429))).toBe(false);
    expect(isNotifiedError(failed(400, { disableNotifications: true }))).toBe(
      false
    );
    expect(
      isNotifiedError(failed(400, { disableNotifications: () => true }))
    ).toBe(false);
  });
});
