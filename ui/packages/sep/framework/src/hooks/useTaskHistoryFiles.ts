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
import { apiClient, type SepComponents } from '@sep/api';

export type FileMetadata = SepComponents['schemas']['FileMetadata'];
export type TaskHistoryFilesMap = Record<string, FileMetadata>;

export function useTaskHistoryFiles(taskHistoryId: number | null | undefined) {
  return useQuery<TaskHistoryFilesMap>({
    queryKey: ['task-history-files', taskHistoryId],
    enabled: taskHistoryId !== null && taskHistoryId !== undefined,
    staleTime: 0,
    queryFn: async () => {
      const { data } = await apiClient.get<TaskHistoryFilesMap>(
        `/files/${taskHistoryId}`,
        {
          baseURL: '',
        }
      );
      return data;
    },
  });
}
