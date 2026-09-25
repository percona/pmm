/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <https://www.gnu.org/licenses/>.
 */

import { AxiosError, AxiosHeaders } from 'axios';
import { describe, expect, it } from 'vitest';
import { defaultQueryClientConfig } from '../src/queryClient';
import { ApiError } from '../src/errors';

function makeAxios4xx(status: number): AxiosError {
  return new AxiosError(
    `HTTP ${status}`,
    'ERR_BAD_REQUEST',
    { headers: new AxiosHeaders() },
    undefined,
    {
      status,
      data: {},
      statusText: '',
      headers: {},
      config: { headers: new AxiosHeaders() },
    }
  );
}

function makeAxios5xx(status: number): AxiosError {
  return new AxiosError(
    `HTTP ${status}`,
    'ERR_BAD_RESPONSE',
    { headers: new AxiosHeaders() },
    undefined,
    {
      status,
      data: {},
      statusText: '',
      headers: {},
      config: { headers: new AxiosHeaders() },
    }
  );
}

describe('defaultQueryClientConfig.retry', () => {
  const retry = defaultQueryClientConfig.defaultOptions?.queries?.retry;

  it('is a function predicate', () => {
    expect(typeof retry).toBe('function');
  });

  it('does not retry 4xx errors (ApiError)', () => {
    const err = new ApiError({ kind: 'http', status: 404, message: 'nope' });
    expect((retry as (n: number, e: unknown) => boolean)(0, err)).toBe(false);
  });

  it('does not retry 4xx errors (AxiosError)', () => {
    const err = makeAxios4xx(404);
    expect((retry as (n: number, e: unknown) => boolean)(0, err)).toBe(false);
  });

  it('retries 5xx errors', () => {
    const err = new ApiError({ kind: 'http', status: 503, message: 'svc' });
    // AxiosError path too
    const axErr = makeAxios5xx(503);
    const fn = retry as (n: number, e: unknown) => boolean;
    expect(fn(0, err)).toBe(true);
    expect(fn(0, axErr)).toBe(true);
  });

  it('retries network errors', () => {
    const err = new ApiError({ kind: 'network', message: 'offline' });
    expect((retry as (n: number, e: unknown) => boolean)(0, err)).toBe(true);
  });

  it('gives up after 3 attempts', () => {
    const err = new ApiError({ kind: 'network', message: 'offline' });
    const fn = retry as (n: number, e: unknown) => boolean;
    expect(fn(3, err)).toBe(false);
  });
});

describe('defaultQueryClientConfig.retryDelay', () => {
  const delay = defaultQueryClientConfig.defaultOptions?.queries
    ?.retryDelay as (n: number) => number;

  it('grows exponentially from 500ms', () => {
    expect(delay(0)).toBe(500);
    expect(delay(1)).toBe(1000);
    expect(delay(2)).toBe(2000);
  });

  it('caps at 30s', () => {
    expect(delay(20)).toBe(30_000);
  });
});
