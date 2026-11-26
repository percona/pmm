/**
 * Copyright (C) 2026 Percona LLC
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 */

import { describe, expect, it } from 'vitest';
import { ApiError, type ApiErrorKind } from '@pmm-extensions/api';

import { extensionsRetry } from './extensionsRetry';

function httpError(status: number): ApiError {
  return new ApiError({ kind: 'http', status, message: `HTTP ${status}` });
}

function nonHttpError(kind: Exclude<ApiErrorKind, 'http'>): ApiError {
  return new ApiError({ kind, message: kind });
}

describe('extensionsRetry', () => {
  it.each([401, 403, 404, 502])(
    'does not retry deterministic %i responses',
    (status) => {
      expect(extensionsRetry(0, httpError(status))).toBe(false);
      expect(extensionsRetry(1, httpError(status))).toBe(false);
      expect(extensionsRetry(5, httpError(status))).toBe(false);
    }
  );

  it('retries transient 5xx (non-502) up to a total of three attempts', () => {
    const err = httpError(500);
    expect(extensionsRetry(0, err)).toBe(true);
    expect(extensionsRetry(1, err)).toBe(true);
    expect(extensionsRetry(2, err)).toBe(false);
  });

  it('retries 503 / 504 as transient', () => {
    expect(extensionsRetry(0, httpError(503))).toBe(true);
    expect(extensionsRetry(0, httpError(504))).toBe(true);
  });

  it('retries network / timeout / unknown ApiErrors', () => {
    expect(extensionsRetry(0, nonHttpError('network'))).toBe(true);
    expect(extensionsRetry(1, nonHttpError('timeout'))).toBe(true);
    expect(extensionsRetry(0, nonHttpError('unknown'))).toBe(true);
  });

  it('does not retry canceled requests', () => {
    expect(extensionsRetry(0, nonHttpError('canceled'))).toBe(false);
    expect(extensionsRetry(1, nonHttpError('canceled'))).toBe(false);
  });

  it('retries plain Error (non-ApiError) up to the same cap', () => {
    const err = new Error('boom');
    expect(extensionsRetry(0, err)).toBe(true);
    expect(extensionsRetry(1, err)).toBe(true);
    expect(extensionsRetry(2, err)).toBe(false);
  });

  it('respects the count cap regardless of count overflow', () => {
    expect(extensionsRetry(100, httpError(500))).toBe(false);
  });
});
