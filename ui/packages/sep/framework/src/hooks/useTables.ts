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

export interface TableOption {
  id: number;
  name: string;
}

export interface UseTablesOptions {
  schemaId: number | null | undefined;
  enabled?: boolean;
}

/**
 * Fetch tables for a schema via the SEP inventory gateway
 * (`GET /sep/schemas/{id}/tables` → `[{id, name}]`).
 *
 * Disabled when `schemaId` is nullish.
 */
export function useTables(
  options: UseTablesOptions
): UseQueryResult<TableOption[], Error> {
  const { schemaId, enabled = true } = options;
  return useQuery<TableOption[], Error>({
    queryKey: ['inventory', 'tables', schemaId ?? null],
    enabled: enabled && schemaId !== null && schemaId !== undefined,
    staleTime: 60_000,
    queryFn: async () => {
      const { data } = await apiClient.get<TableOption[]>(
        `/sep/schemas/${schemaId}/tables`
      );
      return data;
    },
  });
}
