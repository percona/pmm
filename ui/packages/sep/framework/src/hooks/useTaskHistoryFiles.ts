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
import { apiClient, SEP_BASE_PATH, type SepComponents } from '@sep/api';

export type FileMetadata = SepComponents['schemas']['FileMetadata'];
export type TaskHistoryFilesMap = Record<string, FileMetadata>;

export interface UseTaskHistoryFilesOptions {
  /**
   * React Query stale time in ms. The dialog keeps the default ``0`` so a
   * re-open always re-lists; the history-table download affordance can pass a
   * longer value to avoid re-hitting Nomad on every poll tick.
   */
  staleTime?: number;
}

export function useTaskHistoryFiles(
  taskHistoryId: number | null | undefined,
  options?: UseTaskHistoryFilesOptions
) {
  return useQuery<TaskHistoryFilesMap>({
    queryKey: ['task-history-files', taskHistoryId],
    enabled: taskHistoryId !== null && taskHistoryId !== undefined,
    staleTime: options?.staleTime ?? 0,
    queryFn: async () => {
      const { data } = await apiClient.get<TaskHistoryFilesMap>(
        `/files/${taskHistoryId}`,
        {
          baseURL: SEP_BASE_PATH,
        }
      );
      return data;
    },
  });
}
