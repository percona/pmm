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

import { ApiError } from '@sep/api';

/**
 * Shared React Query `retry` predicate for SEP API hooks.
 *
 * Never retry on auth failures (401/403), missing resources (404),
 * upstream-proxy failures (502 from `/api/sep/*` gateway routes), or
 * user-driven aborts (``kind === 'canceled'``) — those are deterministic
 * signals that won't change between retries. For anything else (transient
 * network blips, 5xx other than 502), retry up to twice for a total of 3
 * attempts.
 */
export function sepRetry(count: number, err: Error): boolean {
  if (err instanceof ApiError) {
    const { status, kind } = err;
    if (status === 401 || status === 403 || status === 404 || status === 502) {
      return false;
    }
    if (kind === 'canceled') {
      return false;
    }
  }
  return count < 2;
}
