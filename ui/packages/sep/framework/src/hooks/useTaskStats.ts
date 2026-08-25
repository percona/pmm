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
import { sepApi, throwOnApiError } from '@sep/api';
import { sepRetry } from './sepRetry';

/**
 * Consumer-side view of the task-stats payload.
 *
 * The SEP proxy at ``app/sep/api/routes/task_stats.py`` returns the raw
 * upstream payload (``dict[str, Any]``) on success and a ``502`` with a
 * ``{"detail": ...}`` body on upstream failure (surfaced here as an
 * ``ApiError`` on the React Query error slot — not as ``{}``). Every field
 * is optional because the upstream success payload itself is untyped.
 * Consumers thus see two distinct empty-state signals: upstream failure as
 * ``isError`` (no ``data`` to inspect), and a successful-but-empty stats
 * payload as ``data`` present with ``total === 0``.
 */
export interface TaskStatsView {
  engine?: string;
  total?: number;
  status?: { pass?: number; fail?: number };
  duration?: {
    average_seconds?: number | null;
    last_seconds?: number | null;
    total_seconds?: number | null;
  };
  last_finished_at?: string | null;
}

/**
 * Query the aggregated execution statistics for a task by name.
 *
 * The task identifier is the task's ``name`` (string), not the database id.
 * The hook is a no-op when ``taskName`` is missing or whitespace-only.
 */
export function useTaskStats(taskName: string | undefined, enabled = true) {
  const trimmed = taskName?.trim();
  return useQuery<TaskStatsView>({
    queryKey: ['task-stats', trimmed],
    enabled: enabled && Boolean(trimmed),
    queryFn: async () => {
      const data = await throwOnApiError(
        sepApi.GET('/api/sep/task-stats/{task_name}', {
          params: { path: { task_name: trimmed as string } },
        })
      );
      return data as unknown as TaskStatsView;
    },
    refetchOnWindowFocus: false,
    retry: sepRetry,
    staleTime: 30_000,
  });
}
