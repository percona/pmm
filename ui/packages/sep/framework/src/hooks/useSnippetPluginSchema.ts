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
import { apiClient, type PluginSchema } from '@sep/api';

const SNIPPET_PLUGIN_SCHEMA_STALE_MS = 5 * 60 * 1000;

/**
 * Load a snippets-plugin schema from an API path relative to `/api`
 * (e.g. `/apps/snippets/my-script.sh/schema`, including encoded slashes).
 *
 * Shared by the snippets detail page (path derived from filename) and flows
 * like ATW that compose the same URLs client-side.
 */
export function useSnippetPluginSchema(apiPath: string | null | undefined) {
  return useQuery<PluginSchema>({
    queryKey: ['plugins', 'snippets', 'schema', apiPath ?? ''],
    queryFn: async () => {
      if (!apiPath) {
        throw new Error('Missing snippets plugin schema path');
      }
      const { data } = await apiClient.get<PluginSchema>(apiPath);
      return data;
    },
    enabled: Boolean(apiPath),
    staleTime: SNIPPET_PLUGIN_SCHEMA_STALE_MS,
  });
}
