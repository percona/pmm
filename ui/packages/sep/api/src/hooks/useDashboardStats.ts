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

import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../client';

interface DashboardStatsRaw {
  nodes: number;
  tasks: number;
  snippets: number;
  targets: number;
}

export interface DashboardStats extends DashboardStatsRaw {
  /** Source names that failed and returned 0. Mirrors ``X-Sep-Upstream-Error``. */
  degraded?: string[];
}

export function useDashboardStats() {
  return useQuery<DashboardStats>({
    queryKey: ['sep', 'dashboard', 'stats'],
    queryFn: async () => {
      const response =
        await apiClient.get<DashboardStatsRaw>('/sep/dashboard/');
      const errorHeader = response.headers['x-sep-upstream-error'] as
        | string
        | undefined;
      return {
        ...response.data,
        degraded: errorHeader ? errorHeader.split(',') : undefined,
      };
    },
  });
}
