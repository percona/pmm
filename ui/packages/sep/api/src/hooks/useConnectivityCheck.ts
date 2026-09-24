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

/**
 * React Query hook for the admin on-demand connectivity check (SEP-1413).
 *
 * Posts `POST /api/sep/admin/connectivity-check/` through `apiClient` so the
 * Bearer interceptor attaches the in-memory access token (the endpoint requires
 * `IsApiAdmin` + `RequireBearerForUnsafeMethods`). Returns one
 * `ConnectivityResult` per requested target; per-service probe failures are
 * classified into `status` server-side and never fail the whole request.
 */
import { useMutation } from '@tanstack/react-query';

import { apiClient } from '../client';
import type { ApiError } from '../errors';
import type { components } from '../generated/sep';

/** Request body: the services to probe (at least one; backend collapses dupes). */
export type ConnectivityCheckRequest =
  components['schemas']['ConnectivityCheckRequest'];
/** Per-service probe outcome. `detail` is guaranteed secret-free by the backend. */
export type ConnectivityResult = components['schemas']['ConnectivityResult'];
/** Machine-readable outcome state carried alongside `reachable`. */
export type ConnectivityStatus =
  components['schemas']['ConnectivityStatusEnum'];

// baseURL is already '/api'; trailing slash matches the FastAPI route (avoids a 307).
export const CONNECTIVITY_CHECK_PATH = '/sep/admin/connectivity-check/';

/**
 * Probe reachability of the named external / inter-service endpoints on demand.
 *
 * Admin-only on the backend; callers should gate the trigger on
 * `useAuth().isAdmin`.
 */
export function useConnectivityCheck() {
  return useMutation<ConnectivityResult[], ApiError, ConnectivityCheckRequest>({
    mutationFn: async (body) => {
      const response = await apiClient.post<ConnectivityResult[]>(
        CONNECTIVITY_CHECK_PATH,
        body
      );
      return response.data;
    },
  });
}
