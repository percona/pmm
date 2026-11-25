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
import type { components } from '../generated/extensions';

/** Per-app entry returned by the public ``GET /api/apps/`` endpoint. */
export type EnabledApp = components['schemas']['AppKeyResponse'];

export const ENABLED_APPS_QUERY_KEY = ['apps'] as const;

/**
 * Fetches per-app navigation metadata for the current user.
 *
 * Powers registry-driven sidebar labels (``display_name``), visibility
 * (``enabled``, ``sidebar``), and routing hints. Cached for 30s — admins
 * toggle apps infrequently and brief staleness is acceptable for navigation.
 */
export function useEnabledApps() {
  return useQuery<EnabledApp[]>({
    queryKey: ENABLED_APPS_QUERY_KEY,
    queryFn: async () => {
      const { data } = await apiClient.get<EnabledApp[]>('/apps/');
      return data;
    },
    staleTime: 30_000,
  });
}
