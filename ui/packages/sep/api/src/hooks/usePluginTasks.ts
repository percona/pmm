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
import type { components as TasksComponents } from '../generated/tasks';

export type TaskHistoryStatus =
  TasksComponents['schemas']['TaskHistoryStatusEnum'];

/**
 * Task-history statuses that count as "still executing".
 *
 * Canonical source of truth for the poll-while-running convention. It lives in
 * the ``api`` package because ``api`` is the lowest layer — ``framework`` and
 * the apps depend on it, never the other way round — so both the list-page
 * poll predicate here and the framework's ``useTaskHistory`` reuse the same
 * set instead of drifting apart. ``framework`` re-exports these for callers
 * that import from ``@sep/framework``.
 */
export const RUNNING_STATUSES: ReadonlySet<TaskHistoryStatus> = new Set([
  'running',
  'pending',
]);

export function isRunningStatus(status: TaskHistoryStatus): boolean {
  return RUNNING_STATUSES.has(status);
}

/** Default poll cadence (ms) while a running task is visible; mirrors ``useTaskHistory``. */
const DEFAULT_TASK_POLLING_INTERVAL_MS = 5000;

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

export type PaginatedPluginList<T> = {
  items: T[];
  total: number;
  offset: number;
  limit: number;
};

export type PluginListPagination = {
  total: number;
  offset: number;
  limit: number;
};

export type PluginListResult<T> = {
  items: T[];
  /** ``null`` when the backend returned a bare array (``NO_PAGINATION``). */
  pagination: PluginListPagination | null;
  /**
   * Set when ``fetchAllPages`` stopped at the page cap before reaching
   * ``total``. Absent (or false) when the fetch completed or was not used.
   */
  truncated?: boolean;
};

function isPaginatedPluginListEnvelope<T>(
  data: unknown
): data is PaginatedPluginList<T> {
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    return false;
  }
  const candidate = data as PaginatedPluginList<T>;
  return (
    Array.isArray(candidate.items) &&
    typeof candidate.total === 'number' &&
    typeof candidate.offset === 'number' &&
    typeof candidate.limit === 'number'
  );
}

/**
 * Normalize a plugin list response to items plus optional pagination metadata.
 *
 * Bare arrays (``NO_PAGINATION``) yield ``pagination: null``; full
 * ``{ items, total, offset, limit }`` envelopes preserve all four fields.
 * Partial ``{ items }`` envelopes (legacy migration shape) unwrap items only.
 */
export function normalizePluginListResponse<T>(
  data: T[] | PaginatedPluginList<T> | { items: T[] | null } | null | undefined
): PluginListResult<T> {
  if (Array.isArray(data)) {
    return { items: data, pagination: null };
  }
  if (isPaginatedPluginListEnvelope<T>(data)) {
    return {
      items: data.items,
      pagination: {
        total: data.total,
        offset: data.offset,
        limit: data.limit,
      },
    };
  }
  if (data && typeof data === 'object' && 'items' in data) {
    const items = (data as { items: T[] | null }).items;
    return { items: items ?? [], pagination: null };
  }
  return { items: [], pagination: null };
}

export const DEFAULT_PLUGIN_LIST_OFFSET = 0;
/**
 * Default page size for plugin list requests. Keep list UI page-size options at
 * this value: some plugins (``DEFAULT_PAGINATION_LIMIT``) reject ``limit`` > 50
 * with HTTP 422.
 */
export const DEFAULT_PLUGIN_LIST_LIMIT = 50;

/** Soft cap on pages walked by ``fetchAllPages`` (~2500 rows at the default limit). */
export const MAX_FETCH_ALL_PAGES = 50;

export type PluginListQueryOptions = {
  enabled?: boolean;
  offset?: number;
  limit?: number;
  /**
   * Walk paginated list pages for schedule joins (e.g. SchemaListView schedule
   * columns). Caps at ``MAX_FETCH_ALL_PAGES`` × ``DEFAULT_PLUGIN_LIST_LIMIT``
   * (2500 items); if ``total`` is larger the result sets ``truncated: true``
   * and logs a warning. Issues up to that many sequential GETs.
   */
  fetchAllPages?: boolean;
  /** Disable poll-while-running (stories, tests). */
  disablePolling?: boolean;
  /** Override poll cadence (ms) while a running task is visible. */
  pollingIntervalMs?: number;
};

function mockItemsToResult<T>(mockItems: T[]): PluginListResult<T> {
  return { items: mockItems, pagination: null };
}

function tasksQueryKey(
  pluginName: string,
  options: Pick<PluginListQueryOptions, 'offset' | 'limit' | 'fetchAllPages'>
) {
  return [
    'plugins',
    pluginName,
    'tasks',
    {
      offset: options.offset ?? DEFAULT_PLUGIN_LIST_OFFSET,
      limit: options.limit ?? DEFAULT_PLUGIN_LIST_LIMIT,
      fetchAllPages: options.fetchAllPages ?? false,
    },
  ] as const;
}

/**
 * Plugin list endpoint shape during the multi-plugin migration:
 * - Legacy plugins return `T[]` directly.
 * - Migrated plugins (e.g. mysql_backups) return `PaginatedResponse<T>`.
 */
type PluginListResponse<T> =
  | T[]
  | PaginatedPluginList<T>
  | { items: T[] | null };

async function fetchAllPluginListPages<T extends Record<string, unknown>>(
  path: string
): Promise<PluginListResult<T>> {
  const out: T[] = [];
  let offset = 0;
  let lastTotal: number | null = null;
  const limit = DEFAULT_PLUGIN_LIST_LIMIT;

  for (let iter = 0; iter < MAX_FETCH_ALL_PAGES; iter++) {
    const { data } = await apiClient.get<PluginListResponse<T>>(path, {
      params: { offset, limit },
    });
    const page = normalizePluginListResponse(data);

    if (page.pagination === null) {
      return { items: page.items, pagination: null };
    }

    lastTotal = page.pagination.total;
    out.push(...page.items);
    offset += page.items.length;
    if (offset >= page.pagination.total || page.items.length === 0) {
      return { items: out, pagination: null };
    }
  }

  // eslint-disable-next-line no-console -- surface silent schedule-join truncation
  console.warn(
    `[api] fetchAllPages truncated ${path} at ${out.length} of ${lastTotal ?? '?'} items ` +
      `(cap ${MAX_FETCH_ALL_PAGES}×${limit})`
  );
  return { items: out, pagination: null, truncated: true };
}

async function fetchPluginList<T extends Record<string, unknown>>(
  path: string,
  options: Pick<
    PluginListQueryOptions,
    'offset' | 'limit' | 'fetchAllPages'
  > = {}
): Promise<PluginListResult<T>> {
  if (options.fetchAllPages) {
    return fetchAllPluginListPages<T>(path);
  }
  const offset = options.offset ?? DEFAULT_PLUGIN_LIST_OFFSET;
  const limit = options.limit ?? DEFAULT_PLUGIN_LIST_LIMIT;
  const { data } = await apiClient.get<PluginListResponse<T>>(path, {
    params: { offset, limit },
  });
  return normalizePluginListResponse(data);
}

/** Fetch plugin task list rows, optionally across all pages. */
export async function fetchPluginTasksList<T extends Record<string, unknown>>(
  pluginName: string,
  options: Pick<
    PluginListQueryOptions,
    'offset' | 'limit' | 'fetchAllPages'
  > = {}
): Promise<PluginListResult<T>> {
  return fetchPluginList<T>(`/apps/${pluginName}/`, options);
}

/** Fetch rows for one entity of a multi-entity plugin, optionally across all pages. */
export async function fetchPluginEntityList<T extends Record<string, unknown>>(
  pluginName: string,
  entityName: string,
  options: Pick<
    PluginListQueryOptions,
    'offset' | 'limit' | 'fetchAllPages'
  > = {}
): Promise<PluginListResult<T>> {
  return fetchPluginList<T>(`/apps/${pluginName}/${entityName}/`, options);
}

/**
 * Poll while any fetched row is still running; otherwise stay idle.
 *
 * Exported for tests; not part of the public hook surface.
 */
export function taskListRefetchInterval<T extends Record<string, unknown>>(
  data: T[] | undefined,
  pollingMs: number,
  disabled: boolean
): number | false {
  if (disabled || !data) {
    return false;
  }
  const hasRunning = data.some((row) =>
    RUNNING_STATUSES.has(row.status as TaskHistoryStatus)
  );
  return hasRunning ? pollingMs : false;
}

export function usePluginTasks<T extends Record<string, unknown>>(
  pluginName: string,
  mockTasks?: T[],
  options?: PluginListQueryOptions
) {
  const offset = options?.offset ?? DEFAULT_PLUGIN_LIST_OFFSET;
  const limit = options?.limit ?? DEFAULT_PLUGIN_LIST_LIMIT;
  const fetchAllPages = options?.fetchAllPages ?? false;
  const disablePolling = options?.disablePolling ?? false;
  const pollingIntervalMs =
    options?.pollingIntervalMs ?? DEFAULT_TASK_POLLING_INTERVAL_MS;

  return useQuery<PluginListResult<T>>({
    queryKey: tasksQueryKey(pluginName, { offset, limit, fetchAllPages }),
    enabled: options?.enabled !== false,
    queryFn: async () => {
      try {
        return await fetchPluginTasksList<T>(pluginName, {
          offset,
          limit,
          fetchAllPages,
        });
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockTasks &&
          isBackendUnavailable(error)
        ) {
          return mockItemsToResult(mockTasks);
        }
        throw error;
      }
    },
    // Keep the previous page while offset/limit changes, but not across a
    // plugin switch that leaves the list mounted (would flash the wrong rows).
    placeholderData: (previousData, previousQuery) =>
      previousQuery?.queryKey[1] === pluginName ? previousData : undefined,
    // Keep the list live while something is running; an idle list issues no
    // repeat requests. Polling reflects the task's actual current state rather
    // than its state at page load.
    refetchInterval: (query) =>
      taskListRefetchInterval(
        query.state.data?.items,
        pollingIntervalMs,
        disablePolling
      ),
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

function entityQueriesPrefix(pluginName: string, entityName: string) {
  return ['plugins', pluginName, 'entity', entityName] as const;
}

function entityListQueryKey(
  pluginName: string,
  entityName: string,
  options: Pick<PluginListQueryOptions, 'offset' | 'limit' | 'fetchAllPages'>
) {
  return [
    ...entityQueriesPrefix(pluginName, entityName),
    {
      offset: options.offset ?? DEFAULT_PLUGIN_LIST_OFFSET,
      limit: options.limit ?? DEFAULT_PLUGIN_LIST_LIMIT,
      fetchAllPages: options.fetchAllPages ?? false,
    },
  ] as const;
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
  options?: PluginListQueryOptions
) {
  const offset = options?.offset ?? DEFAULT_PLUGIN_LIST_OFFSET;
  const limit = options?.limit ?? DEFAULT_PLUGIN_LIST_LIMIT;
  const fetchAllPages = options?.fetchAllPages ?? false;

  return useQuery<PluginListResult<T>>({
    queryKey: entityListQueryKey(pluginName, entityName, {
      offset,
      limit,
      fetchAllPages,
    }),
    enabled: options?.enabled !== false,
    queryFn: async () => {
      try {
        return await fetchPluginEntityList<T>(pluginName, entityName, {
          offset,
          limit,
          fetchAllPages,
        });
      } catch (error) {
        if (
          MOCK_FALLBACKS_ENABLED &&
          mockItems &&
          isBackendUnavailable(error)
        ) {
          return mockItemsToResult(mockItems);
        }
        throw error;
      }
    },
    // Keep the previous page while offset/limit changes, but not across an
    // entity-tab (or plugin) switch that leaves PluginListPage mounted.
    placeholderData: (previousData, previousQuery) =>
      previousQuery?.queryKey[1] === pluginName &&
      previousQuery?.queryKey[3] === entityName
        ? previousData
        : undefined,
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
    queryKey: [...entityQueriesPrefix(pluginName, entityName), itemId],
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
