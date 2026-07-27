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

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  apiClient,
  isRunningStatus,
  RUNNING_STATUSES,
  type SepComponents,
  type TaskHistoryStatus,
  type TasksComponents,
} from '@sep/api';

export type TaskHistoryEntry =
  TasksComponents['schemas']['TaskHistoryResponse'];
export type PaginatedTaskHistory =
  TasksComponents['schemas']['PaginatedResponse_TaskHistoryResponse_'];

/** Optional JSON body for ``POST .../execute`` (chain wiring, schedule ETA, etc.). */
export type TaskExecuteBody = SepComponents['schemas']['TaskExecuteWrite'];

// The poll-while-running status set is owned by the ``api`` package (the lower
// layer) so the schema-driven list page and this hook share one definition.
// Re-exported here to keep the ``@sep/framework`` import surface stable.
export { isRunningStatus, RUNNING_STATUSES };
export type { TaskHistoryStatus };

export interface UseTaskHistoryOptions {
  /** Filter by exact status (server-side ?status=). */
  status?: TaskHistoryStatus | null;
  /** Override polling interval (ms). Defaults to 5000. */
  pollingIntervalMs?: number;
  /** Disable polling regardless of running tasks. */
  disablePolling?: boolean;
  /** Optional offset/limit for client-side prefetch (server pagination is deferred). */
  offset?: number;
  limit?: number;
  /** Disable the underlying query (no fetch, no polling). */
  enabled?: boolean;
  /** Exclude internal maintenance task rows before server-side pagination. */
  excludeInternal?: boolean;
}

interface ListParams {
  status?: TaskHistoryStatus | null;
  offset?: number;
  limit?: number;
  excludeInternal?: boolean;
}

function buildParams({
  status,
  offset,
  limit,
  excludeInternal,
}: ListParams): Record<string, string | number | boolean> {
  const params: Record<string, string | number | boolean> = {};
  if (status) {
    params.status = status;
  }
  if (typeof offset === 'number') {
    params.offset = offset;
  }
  if (typeof limit === 'number') {
    params.limit = limit;
  }
  if (excludeInternal) {
    params.exclude_internal = true;
  }
  return params;
}

function refetchIntervalFor(
  data: PaginatedTaskHistory | undefined,
  pollingMs: number,
  disabled: boolean
): number | false {
  if (disabled) {
    return false;
  }
  if (!data) {
    return false;
  }
  const hasRunning = data.items.some((entry) => isRunningStatus(entry.status));
  return hasRunning ? pollingMs : false;
}

/**
 * List task history across all tasks.
 *
 * Requests are routed through ``GET /api/sep/task-history/`` (with no
 * `task_names`) so the browser never calls the Tasks sub-app directly. Polling
 * activates only when at least one row is in a running state, matching the
 * legacy Jinja2 page reload behaviour. Server cursor pagination is deferred;
 * callers paginate client-side for now.
 */
export function useTaskHistory(options: UseTaskHistoryOptions = {}) {
  const {
    status,
    pollingIntervalMs = 5000,
    disablePolling = false,
    offset,
    limit,
    enabled = true,
    excludeInternal = false,
  } = options;
  return useQuery<PaginatedTaskHistory>({
    queryKey: [
      'task-history',
      { status: status ?? null, offset, limit, excludeInternal },
    ],
    enabled,
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedTaskHistory>(
        '/sep/task-history/',
        {
          params: buildParams({ status, offset, limit, excludeInternal }),
        }
      );
      return data;
    },
    refetchInterval: (query) =>
      refetchIntervalFor(query.state.data, pollingIntervalMs, disablePolling),
  });
}

/**
 * List task history filtered to a specific task by name.
 */
export function useTaskHistoryByName(
  taskName: string | undefined,
  options: UseTaskHistoryOptions = {}
) {
  return useTaskHistoryByNames(taskName ? [taskName] : undefined, options);
}

/**
 * List task history for one or more tasks, merged newest-first.
 *
 * Used by plugins whose detail page represents a task group (parent plus derived
 * siblings) while executions are recorded on individual task names. Requests are
 * routed through ``GET /api/sep/task-history/`` so the browser never calls the
 * Tasks sub-app directly.
 */
export function useTaskHistoryByNames(
  taskNames: string[] | undefined,
  options: UseTaskHistoryOptions = {}
) {
  const {
    status,
    pollingIntervalMs = 5000,
    disablePolling = false,
    offset,
    limit,
    enabled = true,
  } = options;
  const names = [
    ...new Set((taskNames ?? []).map((name) => name.trim()).filter(Boolean)),
  ].sort();
  return useQuery<PaginatedTaskHistory>({
    queryKey: [
      'task-history',
      'merged',
      names,
      { status: status ?? null, offset, limit },
    ],
    enabled: enabled && names.length > 0,
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedTaskHistory>(
        '/sep/task-history/',
        {
          params: {
            ...buildParams({ status, offset, limit }),
            task_names: names,
          },
          // FastAPI list query params use repeated keys (`task_names=a&task_names=b`),
          // not axios' default bracket form (`task_names[]=a&task_names[]=b`).
          paramsSerializer: { indexes: null },
        }
      );
      return data;
    },
    refetchInterval: (query) =>
      refetchIntervalFor(query.state.data, pollingIntervalMs, disablePolling),
  });
}

/**
 * Stop a running task history execution.
 */
export function useStopTaskHistory() {
  const queryClient = useQueryClient();
  return useMutation<TaskHistoryEntry, Error, number>({
    mutationFn: async (taskHistoryId) => {
      const { data } = await apiClient.post<TaskHistoryEntry>(
        `/sep/task-history/${taskHistoryId}/stop/`
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['task-history'] });
    },
  });
}

/**
 * Dispatch a saved task for immediate execution.
 *
 * The request is routed through the SEP-level plugin gateway
 * (``POST /api/apps/{pluginName}/{taskName}/execute``) — the FE must not
 * call ``/api/tasks/*`` directly, as the Tasks sub-app is not exposed to the
 * browser in a production deployment. The plugin-task detail query is also
 * refreshed so the status chip on the detail page updates immediately after
 * execution.
 */
export function useExecuteTask(pluginName: string) {
  const queryClient = useQueryClient();
  const pluginPath = pluginName.split('/').map(encodeURIComponent).join('/');
  return useMutation<
    unknown,
    Error,
    { taskName: string; executeBody?: TaskExecuteBody }
  >({
    mutationFn: async ({ taskName, executeBody }) => {
      const { data } = await apiClient.post<unknown>(
        `/apps/${pluginPath}/${encodeURIComponent(taskName)}/execute`,
        executeBody ?? {}
      );
      return data;
    },
    onSuccess: (_data, { taskName }) => {
      queryClient.invalidateQueries({ queryKey: ['task-history'] });
      queryClient.invalidateQueries({ queryKey: ['task-history', taskName] });
      queryClient.invalidateQueries({
        queryKey: ['plugins', pluginName, 'tasks'],
      });
      queryClient.removeQueries({
        queryKey: ['plugins', pluginName, 'tasks', taskName],
      });
    },
  });
}
