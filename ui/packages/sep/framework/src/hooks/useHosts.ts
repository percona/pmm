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

import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import { apiClient } from '@sep/api';
import { sepRetry } from './sepRetry';

export interface HostOption {
  /** Executor node name — the value sent to the dispatch payload as `executor_host`. */
  id: string;
  /** Human-readable label: inventory display name when available, else `id`. */
  name: string;
  /** Network address reported by the executor. */
  address: string;
}

export interface UseHostsOptions {
  enabled?: boolean;
}

/**
 * Fetch executor hosts merged with inventory display names.
 *
 * Calls the SEP-side proxy `GET /api/sep/hosts/`, which performs the
 * Tasks/Inventory merge server-side. Loading and error states are
 * first-class React Query states. When the Tasks API is unreachable the
 * route responds with `502` + `{"detail": "<upstream detail>"}`; the axios
 * interceptor turns that into an `ApiError` whose `.message` carries the
 * detail, so consumers handle it via React Query's `isError` / `error`
 * slot — no out-of-band header plumbing.
 */
export function useHosts(
  options: UseHostsOptions = {}
): UseQueryResult<HostOption[], Error> {
  const { enabled = true } = options;
  return useQuery<HostOption[], Error>({
    queryKey: ['sep', 'hosts'],
    enabled,
    staleTime: 60_000,
    queryFn: async () => {
      const response = await apiClient.get<HostOption[]>('/sep/hosts/');
      return response.data;
    },
    retry: sepRetry,
  });
}
