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

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '@sep/api';
import type {
  SnippetExecutionRequest,
  SnippetExecutionResponse,
} from '../types/snippetPlugin';

export interface UseSnippetPluginExecutionOptions {
  /**
   * When set, invalidate per-snippet history after a successful execute
   * (`useSnippetHistory` in `@sep/plugins-snippets` uses this query-key shape).
   */
  invalidateHistoryForFilename?: string;
}

/**
 * POST snippet execution against an API path relative to `/api`
 * (e.g. `/apps/snippets/my-script.sh/execute`).
 */
export function useSnippetPluginExecution(
  executePath: string | null | undefined,
  options?: UseSnippetPluginExecutionOptions
) {
  const queryClient = useQueryClient();
  const historyFilename = options?.invalidateHistoryForFilename;

  return useMutation<SnippetExecutionResponse, Error, SnippetExecutionRequest>({
    mutationFn: async (body) => {
      if (!executePath) {
        throw new Error('Missing snippets plugin execute path');
      }
      const { data } = await apiClient.post<SnippetExecutionResponse>(
        executePath,
        body
      );
      return data;
    },
    onSuccess: () => {
      if (historyFilename) {
        queryClient.invalidateQueries({
          queryKey: ['snippets', historyFilename, 'history'],
        });
      }
    },
  });
}
