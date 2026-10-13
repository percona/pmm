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

import { describe, expect, it } from 'vitest';
import { ApiError, normalizeBlobError, parseFieldErrors } from '../src/errors';

function http422(detail: unknown): ApiError {
  return new ApiError({
    kind: 'http',
    status: 422,
    message: 'HTTP 422',
    data: { detail },
  });
}

describe('parseFieldErrors', () => {
  it('strips a leading "body" segment from FastAPI request-body validation locs', () => {
    const error = http422([
      {
        loc: ['body', 'limit'],
        msg: 'ensure this value is greater than 0',
        type: 'greater_than',
      },
    ]);
    expect(parseFieldErrors(error)).toEqual([
      { path: 'limit', message: 'ensure this value is greater than 0' },
    ]);
  });

  it('keeps locs without a "body" prefix (manual model_validate paths)', () => {
    const error = http422([
      { loc: ['sleep'], msg: 'field required', type: 'missing' },
    ]);
    expect(parseFieldErrors(error)).toEqual([
      { path: 'sleep', message: 'field required' },
    ]);
  });

  it('joins nested object and array-index segments into a dot path', () => {
    const error = http422([
      {
        loc: ['body', 'source', 'host'],
        msg: 'invalid host',
        type: 'value_error',
      },
      {
        loc: ['body', 'items', 0, 'name'],
        msg: 'too short',
        type: 'string_too_short',
      },
    ]);
    expect(parseFieldErrors(error)).toEqual([
      { path: 'source.host', message: 'invalid host' },
      { path: 'items.0.name', message: 'too short' },
    ]);
  });

  it('yields an empty path for a body-level error so callers can surface it', () => {
    const error = http422([
      { loc: ['body'], msg: 'value is not a valid dict', type: 'dict_type' },
    ]);
    expect(parseFieldErrors(error)).toEqual([
      { path: '', message: 'value is not a valid dict' },
    ]);
  });

  it('falls back to a generic message when msg is missing or empty', () => {
    const error = http422([
      { loc: ['body', 'name'] },
      { loc: ['body', 'age'], msg: '' },
    ]);
    expect(parseFieldErrors(error)).toEqual([
      { path: 'name', message: 'Invalid value' },
      { path: 'age', message: 'Invalid value' },
    ]);
  });

  it('accepts a raw payload object as well as an ApiError', () => {
    expect(
      parseFieldErrors({ detail: [{ loc: ['body', 'x'], msg: 'bad' }] })
    ).toEqual([{ path: 'x', message: 'bad' }]);
  });

  it('returns [] when there is no recognizable detail array', () => {
    expect(parseFieldErrors(http422('Unprocessable Entity'))).toEqual([]);
    expect(parseFieldErrors(http422(undefined))).toEqual([]);
    expect(
      parseFieldErrors(new ApiError({ kind: 'network', message: 'offline' }))
    ).toEqual([]);
    expect(parseFieldErrors(null)).toEqual([]);
    expect(parseFieldErrors('boom')).toEqual([]);
  });
});

describe('normalizeBlobError', () => {
  function blobFailure(
    status: number,
    body: string,
    type = 'application/json'
  ): ApiError {
    return new ApiError({
      kind: 'http',
      status,
      message: `HTTP ${status}`,
      data: new Blob([body], { type }),
      url: '/files/7/download',
      method: 'GET',
    });
  }

  it("recovers a refusal's reason from a blob response body", async () => {
    const recovered = await normalizeBlobError(
      blobFailure(
        403,
        JSON.stringify({
          detail: "You don't have permission to perform this action",
        })
      )
    );

    expect(recovered.message).toBe(
      "You don't have permission to perform this action"
    );
    expect(recovered.status).toBe(403);
    expect(recovered.method).toBe('GET');
  });

  it("keeps a 422's detail array reachable through parseFieldErrors", async () => {
    const recovered = await normalizeBlobError(
      blobFailure(
        422,
        JSON.stringify({
          detail: [{ loc: ['body', 'name'], msg: 'field required' }],
        })
      )
    );

    expect(parseFieldErrors(recovered)).toEqual([
      { path: 'name', message: 'field required' },
    ]);
  });

  it('leaves the error untouched when the body is not JSON', async () => {
    const original = blobFailure(502, '<html>Bad gateway</html>', 'text/html');

    const recovered = await normalizeBlobError(original);

    expect(recovered.message).toBe('HTTP 502');
  });

  it('passes through an error with no blob body', async () => {
    const original = new ApiError({
      kind: 'network',
      message: 'Network error',
    });

    expect(await normalizeBlobError(original)).toBe(original);
  });
});
