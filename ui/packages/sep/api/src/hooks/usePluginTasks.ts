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

/// <reference path="../vite-env.d.ts" />
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { ApiError } from '../errors';

// Mock-fallback gate. Active in dev builds (`pnpm dev`) and in production
// builds explicitly opted-in via `VITE_MOCK_API=true` (e.g. the Playwright
// preview target). Vite statically replaces both expressions at build time,
// so the fallback branches are dead-code-eliminated in real production.
const MOCK_FALLBACKS_ENABLED =
  import.meta.env.DEV || import.meta.env.VITE_MOCK_API === 'true';

/**
 * Predicate used by the mock-fallback gate: returns ``true`` only when the
 * failure looks like the backend is unreachable, never for deterministic 4xx
 * responses. Exported for tests; not part of the public hook surface.
 *
 * 502 is intentionally treated as "backend unavailable" here even though
 * ``sepRetry`` short-circuits on it — the two predicates serve different
 * goals: ``sepRetry`` wants to stop hammering a known-bad gateway, while this
 * gate decides whether to substitute mock data in dev builds. A 502 from
 * ``/api/sep/*`` means the upstream Tasks-API is unreachable, which is
 * exactly the dev-without-backend scenario the mock fallback targets.
 */
export function isBackendUnavailable(error: unknown): boolean {
  if (error instanceof ApiError) {
    return (
      error.kind === 'network' ||
      (error.kind === 'http' && (error.status ?? 0) >= 500)
    );
  }
  return false;
}

type PaginatedPluginList<T> = {
  items: T[];
  total: number;
  offset: number;
  limit: number;
};

/** Accept legacy flat lists or paginated ``{ items, total, offset, limit }`` envelopes. */
export function unwrapPluginListResponse<T>(
  data: T[] | PaginatedPluginList<T>
): T[] {
  if (Array.isArray(data)) {
    return data;
  }
  if (data && typeof data === 'object' && Array.isArray(data.items)) {
    return data.items;
  }
  return [];
}

/**
 * Generic CRUD hooks for plugin tasks.
 *
 * The real API is always attempted first. When the backend is unavailable
 * (network error or 5xx), mock data is used as a fallback only when the
 * mock-fallback gate is on — that is, in dev builds, and in production
 * builds explicitly opted in via `VITE_MOCK_API=true` (the Playwright
 * preview target). Real production bundles never use the fallback.
 */

/**
 * Plugin task list endpoint shape during the multi-plugin migration:
 * - Legacy plugins return `T[]` directly.
 * - Migrated plugins (e.g. mysql_backups) return `PaginatedResponse<T>`.
 *
 * The hook unwraps both shapes to `T[]` so call sites stay stable.
 */
type PluginTasksResponse<T> = T[] | { items: T[] | null };

export function unwrapTasks<T>(
  data: PluginTasksResponse<T> | null | undefined
): T[] {
  if (Array.isArray(data)) {
    return data;
  }
  return data?.items ?? [];
}

export function usePluginTasks<T extends Record<string, unknown>>(
  pluginName: string,
  mockTasks?: T[],
  options?: { enabled?: boolean }
) {
  return useQuery<T[]>({
    queryKey: ['plugins', pluginName, 'tasks'],
    enabled: options?.enabled !== false,
    queryFn: async () => {
      try {
        const { data } = await apiClient.get<PluginTasksResponse<T>>(
          `/apps/${pluginName}/`
        );
        return unwrapTasks(data);
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockTasks &&
          isBackendUnavailable(error)
        ) {
          return mockTasks;
        }
        throw error;
      }
    },
    ...(MOCK_FALLBACKS_ENABLED && mockTasks && { placeholderData: mockTasks }),
  });
}

export function usePluginTask<T extends Record<string, unknown>>(
  pluginName: string,
  taskId: string | undefined,
  mockTasks?: T[],
  options?: { enabled?: boolean }
) {
  return useQuery<T | undefined>({
    queryKey: ['plugins', pluginName, 'tasks', taskId],
    enabled: options?.enabled !== false && !!taskId,
    queryFn: async () => {
      try {
        const { data } = await apiClient.get<T>(
          `/apps/${pluginName}/${encodeURIComponent(taskId!)}`
        );
        return data;
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockTasks &&
          isBackendUnavailable(error)
        ) {
          // Per-plugin detail/delete routes look up by `task_name`; mocks
          // resolve by the same key so dev fallback matches prod semantics.
          return mockTasks.find((t) => String(t.name ?? t.id) === taskId);
        }
        throw error;
      }
    },
  });
}

export function useCreatePluginTask<T extends Record<string, unknown>>(
  pluginName: string,
  mockTasks?: T[]
) {
  const queryClient = useQueryClient();

  return useMutation<T, Error, Record<string, unknown>>({
    mutationFn: async (values) => {
      try {
        const { data } = await apiClient.post<T>(
          `/apps/${pluginName}/`,
          values
        );
        return data;
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockTasks &&
          isBackendUnavailable(error)
        ) {
          return values as T;
        }
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['plugins', pluginName, 'tasks'],
      });
    },
  });
}

export function useUpdatePluginTask<T extends Record<string, unknown>>(
  pluginName: string,
  mockTasks?: T[]
) {
  const queryClient = useQueryClient();

  return useMutation<
    T,
    Error,
    { taskId: string; values: Record<string, unknown> }
  >({
    mutationFn: async ({ taskId, values }) => {
      try {
        const { data } = await apiClient.put<T>(
          `/apps/${pluginName}/${encodeURIComponent(taskId)}`,
          values
        );
        return data;
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockTasks &&
          isBackendUnavailable(error)
        ) {
          return { ...values, name: taskId } as unknown as T;
        }
        throw error;
      }
    },
    onSuccess: (_data, { taskId }) => {
      queryClient.invalidateQueries({
        queryKey: ['plugins', pluginName, 'tasks'],
      });
      queryClient.invalidateQueries({
        queryKey: ['plugins', pluginName, 'tasks', taskId],
      });
    },
  });
}

function entityQueryKey(pluginName: string, entityName: string) {
  return ['plugins', pluginName, 'entity', entityName] as const;
}

function entityQueriesRootKey(pluginName: string) {
  return ['plugins', pluginName, 'entity'] as const;
}

/**
 * Build a per-item URL path for a multi-entity plugin endpoint.
 *
 * ``id`` segments are always URL-encoded so callers cannot smuggle path
 * traversal (``"../foo"``) or sub-paths (``"a/b"``) into the request via a
 * misbehaving backend or attacker-controlled JSON.
 */
export function buildEntityItemPath(
  pluginName: string,
  entityName: string,
  id: string
): string {
  return `/apps/${pluginName}/${entityName}/${encodeURIComponent(id)}`;
}

/** List rows for one entity of a multi-entity plugin (GET ``/apps/{name}/{entity}/``). */
export function usePluginEntityList<T extends Record<string, unknown>>(
  pluginName: string,
  entityName: string,
  mockItems?: T[],
  options?: { enabled?: boolean }
) {
  return useQuery<T[]>({
    queryKey: entityQueryKey(pluginName, entityName),
    enabled: options?.enabled !== false,
    queryFn: async () => {
      try {
        const { data } = await apiClient.get<T[] | PaginatedPluginList<T>>(
          `/apps/${pluginName}/${entityName}/`
        );
        return unwrapPluginListResponse(data);
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockItems &&
          isBackendUnavailable(error)
        ) {
          return mockItems;
        }
        throw error;
      }
    },
    ...(MOCK_FALLBACKS_ENABLED && mockItems && { placeholderData: mockItems }),
  });
}

export function usePluginEntityDetail<T extends Record<string, unknown>>(
  pluginName: string,
  entityName: string,
  itemId: string | undefined,
  mockItems?: T[],
  options?: { enabled?: boolean }
) {
  return useQuery<T | undefined>({
    queryKey: [...entityQueryKey(pluginName, entityName), itemId],
    enabled: options?.enabled !== false && !!itemId,
    queryFn: async () => {
      try {
        const { data } = await apiClient.get<T>(
          buildEntityItemPath(pluginName, entityName, itemId!)
        );
        return data;
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockItems &&
          isBackendUnavailable(error)
        ) {
          return mockItems.find((t) => String(t.id) === itemId);
        }
        throw error;
      }
    },
  });
}

export function useCreatePluginEntity<T extends Record<string, unknown>>(
  pluginName: string,
  entityName: string,
  mockItems?: T[]
) {
  const queryClient = useQueryClient();

  return useMutation<T, Error, Record<string, unknown>>({
    mutationFn: async (values) => {
      try {
        const { data } = await apiClient.post<T>(
          `/apps/${pluginName}/${entityName}/`,
          values
        );
        return data;
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockItems &&
          isBackendUnavailable(error)
        ) {
          return values as T;
        }
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: entityQueriesRootKey(pluginName),
      });
    },
  });
}

export function useUpdatePluginEntity<T extends Record<string, unknown>>(
  pluginName: string,
  entityName: string,
  mockItems?: T[]
) {
  const queryClient = useQueryClient();

  return useMutation<T, Error, { id: string; values: Record<string, unknown> }>(
    {
      mutationFn: async ({ id, values }) => {
        try {
          const { data } = await apiClient.put<T>(
            buildEntityItemPath(pluginName, entityName, id),
            values
          );
          return data;
        } catch (error) {
          if (
            MOCK_FALLBACKS_ENABLED &&
            mockItems &&
            isBackendUnavailable(error)
          ) {
            return { ...values, id } as unknown as T;
          }
          throw error;
        }
      },
      onSuccess: () => {
        queryClient.invalidateQueries({
          queryKey: entityQueriesRootKey(pluginName),
        });
      },
    }
  );
}

export function useDeletePluginEntity(
  pluginName: string,
  entityName: string,
  mockItems?: unknown[]
) {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: async (id) => {
      try {
        await apiClient.delete(buildEntityItemPath(pluginName, entityName, id));
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockItems &&
          isBackendUnavailable(error)
        ) {
          return;
        }
        throw error;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: entityQueriesRootKey(pluginName),
      });
    },
  });
}

export function useDeletePluginTask<T extends Record<string, unknown>>(
  pluginName: string,
  mockTasks?: T[]
) {
  const queryClient = useQueryClient();

  return useMutation<void, Error, string>({
    mutationFn: async (taskId) => {
      try {
        await apiClient.delete(
          `/apps/${pluginName}/${encodeURIComponent(taskId)}`
        );
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockTasks &&
          isBackendUnavailable(error)
        ) {
          // Mock mode: pretend the delete succeeded so the UI flow can be
          // exercised offline, matching the create/list/detail hooks.
          return;
        }
        throw error;
      }
    },
    onSuccess: (_data, taskId) => {
      queryClient.invalidateQueries({
        queryKey: ['plugins', pluginName, 'tasks'],
      });
      queryClient.removeQueries({
        queryKey: ['plugins', pluginName, 'tasks', taskId],
      });
    },
  });
}
