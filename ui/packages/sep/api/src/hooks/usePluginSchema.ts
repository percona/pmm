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
import { ApiError } from '../errors';
import type { PluginSchema } from '../types/plugin-schema';

/**
 * Fetches a plugin's schema from the backend.
 *
 * Accepts an optional `mockSchema` fallback for development when the
 * backend doesn't yet serve schemas. When the real API is available,
 * it takes precedence over the mock.
 */
export function usePluginSchema(pluginName: string, mockSchema?: PluginSchema) {
  return useQuery<PluginSchema>({
    queryKey: ['plugins', pluginName, 'schema'],
    queryFn: async () => {
      try {
        const { data } = await apiClient.get<PluginSchema>(
          `/apps/${pluginName}/schema`
        );
        return data;
      } catch (error) {
        // Fall back to mock schema on 404 (not yet served) or network error
        if (mockSchema && error instanceof ApiError) {
          if (error.status === 404 || error.kind === 'network') {
            return mockSchema;
          }
        }
        throw error;
      }
    },
    ...(mockSchema && { placeholderData: mockSchema }),
    staleTime: 5 * 60 * 1000, // schemas rarely change
  });
}
