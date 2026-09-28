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

import {
  appStateErrorMessage,
  isTransitional,
} from '../src/hooks/useAdminApps';
import { ApiError } from '../src/errors';

describe('isTransitional', () => {
  it('is true for ENABLING and DISABLING', () => {
    expect(isTransitional('ENABLING')).toBe(true);
    expect(isTransitional('DISABLING')).toBe(true);
  });

  it('is false for terminal states', () => {
    expect(isTransitional('ENABLED')).toBe(false);
    expect(isTransitional('DISABLED')).toBe(false);
  });
});

describe('appStateErrorMessage', () => {
  it('returns null when there is no error', () => {
    expect(appStateErrorMessage(null)).toBeNull();
    expect(appStateErrorMessage(undefined)).toBeNull();
  });

  it('prefers the backend detail for a 409 illegal/protected transition', () => {
    const error = new ApiError({
      kind: 'http',
      status: 409,
      message: "App 'inventory' is protected and cannot be disabled.",
      data: { detail: "App 'inventory' is protected and cannot be disabled." },
    });
    expect(appStateErrorMessage(error)).toBe(
      "App 'inventory' is protected and cannot be disabled."
    );
  });

  it('prefers the backend detail for a 404 unknown app key', () => {
    const error = new ApiError({
      kind: 'http',
      status: 404,
      message: 'Not found',
      data: { detail: "App 'ghost' not found." },
    });
    expect(appStateErrorMessage(error)).toBe("App 'ghost' not found.");
  });

  it('falls back to a generic message when 404 has no detail', () => {
    const error = new ApiError({
      kind: 'http',
      status: 404,
      message: 'HTTP 404',
    });
    expect(appStateErrorMessage(error)).toBe('That app no longer exists.');
  });

  it('reports an admin-access message for a 403', () => {
    const error = new ApiError({
      kind: 'http',
      status: 403,
      message: 'Forbidden',
    });
    expect(appStateErrorMessage(error)).toMatch(/administrator access/i);
  });

  it('reports a credentials hint for a 401 (missing bearer token)', () => {
    const error = new ApiError({
      kind: 'http',
      status: 401,
      message: 'Unauthorized',
    });
    expect(appStateErrorMessage(error)).toMatch(/credentials/i);
  });

  it('falls back to the error message for other statuses', () => {
    const error = new ApiError({ kind: 'network', message: 'Network error' });
    expect(appStateErrorMessage(error)).toBe('Network error');
  });
});
