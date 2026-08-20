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

export interface SchemaOption {
  id: number;
  name: string;
}

export interface UseSchemasOptions {
  serviceId: number | null | undefined;
  enabled?: boolean;
}

/**
 * Fetch schemas for a service via the SEP inventory gateway
 * (`GET /sep/services/{id}/schemas` → `[{id, name}]`).
 *
 * Disabled when `serviceId` is nullish.
 */
export function useSchemas(
  options: UseSchemasOptions
): UseQueryResult<SchemaOption[], Error> {
  const { serviceId, enabled = true } = options;
  return useQuery<SchemaOption[], Error>({
    queryKey: ['inventory', 'schemas', serviceId ?? null],
    enabled: enabled && serviceId !== null && serviceId !== undefined,
    staleTime: 60_000,
    queryFn: async () => {
      const { data } = await apiClient.get<SchemaOption[]>(
        `/sep/services/${serviceId}/schemas`
      );
      return data;
    },
  });
}
