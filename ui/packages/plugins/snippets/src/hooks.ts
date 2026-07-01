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
import { apiClient, ApiError, type PluginSchema } from '@sep/api';
import {
  downloadBlob,
  SNIPPETS_PLUGINS_API_BASE,
  snippetPluginApprovalPath,
  snippetPluginDownloadPath,
  snippetPluginExecutePath,
  snippetPluginHistoryPath,
  snippetPluginSchemaPath,
  type PaginatedTaskHistory,
} from '@sep/framework';
import type {
  BatchApprovalErrorResponse,
  BatchApprovalResponse,
  RefreshResponse,
  SnippetBatchApproveRequest,
  SnippetExecutionRequest,
  SnippetExecutionResponse,
  SnippetResponse,
  SnippetsCapabilitiesResponse,
} from './types';

const SNIPPETS_BASE = SNIPPETS_PLUGINS_API_BASE;

function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) && value.every((item) => typeof item === 'string')
  );
}

function extractBatchApprovalError(
  data: unknown
): BatchApprovalErrorResponse | null {
  if (!data || typeof data !== 'object') {
    return null;
  }

  const detail = (data as { detail?: unknown }).detail;
  if (!detail || typeof detail !== 'object') {
    return null;
  }

  const candidate = detail as {
    missing_in_db?: unknown;
    missing_on_disk?: unknown;
  };
  if (
    !isStringArray(candidate.missing_in_db) ||
    !isStringArray(candidate.missing_on_disk)
  ) {
    return null;
  }

  return {
    missing_in_db: candidate.missing_in_db,
    missing_on_disk: candidate.missing_on_disk,
  };
}

/**
 * Fetch the list of snippet entities discovered by the backend.
 */
export function useSnippets() {
  return useQuery<SnippetResponse[]>({
    queryKey: ['snippets', 'list'],
    queryFn: async () => {
      const { data } = await apiClient.get<SnippetResponse[]>(
        `${SNIPPETS_BASE}/`
      );
      return data;
    },
  });
}

/**
 * Fetch the per-snippet form schema.
 *
 * Pass ``executionOnly=true`` on pages that mount ``SnippetExecutionAccordion``
 * so both share ``['snippets', filename, 'schema', { execution_only: true }]``
 * and React Query deduplicates the fetch.
 */
export function useSnippetSchema(
  filename: string | undefined,
  executionOnly = false
) {
  return useQuery<PluginSchema>({
    queryKey: executionOnly
      ? ['snippets', filename, 'schema', { execution_only: true }]
      : ['snippets', filename, 'schema'],
    queryFn: async () => {
      if (!filename) {
        throw new Error('Missing snippet filename');
      }
      const { data } = await apiClient.get<PluginSchema>(
        snippetPluginSchemaPath(filename),
        executionOnly ? { params: { execution_only: true } } : undefined
      );
      return data;
    },
    enabled: Boolean(filename),
    staleTime: executionOnly ? Infinity : 5 * 60 * 1000,
  });
}

/**
 * Fetch the execution history for a single snippet.
 *
 * Returns the upstream paginated tasks-history payload so the React detail
 * page can pass the result straight into the shared ``TaskHistoryTable``.
 */
export function useSnippetHistory(filename: string | undefined) {
  return useQuery<PaginatedTaskHistory>({
    queryKey: ['snippets', filename, 'history'],
    queryFn: async () => {
      if (!filename) {
        throw new Error('Missing snippet filename');
      }
      const { data } = await apiClient.get<PaginatedTaskHistory>(
        snippetPluginHistoryPath(filename)
      );
      return data;
    },
    enabled: Boolean(filename),
  });
}

/**
 * Mutation: download the raw snippet file (full body + YAML frontmatter).
 *
 * Routes through ``apiClient`` so the Bearer interceptor attaches the
 * in-memory access token, then turns the Blob response into a save
 * dialog via a temporary anchor click (mirrors ``useLogDownload``).
 */
export function useSnippetDownload(filename: string | undefined) {
  return useMutation<Blob, Error, void>({
    mutationFn: async () => {
      if (!filename) {
        throw new Error('Snippet filename is required for download.');
      }
      const { data } = await apiClient.get<Blob>(
        snippetPluginDownloadPath(filename),
        {
          responseType: 'blob',
        }
      );
      downloadBlob(data, filename);
      return data;
    },
  });
}

/**
 * Mutation: execute a snippet against the tasks API.
 *
 * On success, the per-snippet history list query is invalidated so the
 * detail page refetches and the new run row appears immediately.
 */
export function useSnippetExecution(filename: string | undefined) {
  const queryClient = useQueryClient();
  return useMutation<SnippetExecutionResponse, Error, SnippetExecutionRequest>({
    mutationFn: async (body) => {
      if (!filename) {
        throw new Error('Snippet filename is required for execution.');
      }
      const { data } = await apiClient.post<SnippetExecutionResponse>(
        snippetPluginExecutePath(filename),
        body
      );
      return data;
    },
    onSuccess: () => {
      if (filename) {
        queryClient.invalidateQueries({
          queryKey: ['snippets', filename, 'history'],
        });
      }
    },
  });
}
/**
 * Mutation: approve a single snippet (idempotent PUT).
 *
 * On success, the snippets list query is invalidated so the list view
 * reflects the new approval state immediately.
 */
export function useApproveSnippet(filename: string) {
  const queryClient = useQueryClient();
  return useMutation<
    SnippetResponse,
    Error,
    void,
    { previous?: SnippetResponse[] }
  >({
    mutationFn: async () => {
      const { data } = await apiClient.put<SnippetResponse>(
        snippetPluginApprovalPath(filename)
      );
      return data;
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ['snippets', 'list'] });
      const previous = queryClient.getQueryData<SnippetResponse[]>([
        'snippets',
        'list',
      ]);
      if (previous) {
        queryClient.setQueryData<SnippetResponse[]>(
          ['snippets', 'list'],
          previous.map((s) =>
            s.filename === filename ? { ...s, is_approved: true } : s
          )
        );
      }
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['snippets', 'list'], context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['snippets', 'list'] });
    },
  });
}

/**
 * Mutation: remove approval from a single snippet (idempotent DELETE).
 *
 * On success, the snippets list query is invalidated.
 */
export function useRemoveSnippetApproval(filename: string) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, void, { previous?: SnippetResponse[] }>({
    mutationFn: async () => {
      await apiClient.delete(snippetPluginApprovalPath(filename));
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: ['snippets', 'list'] });
      const previous = queryClient.getQueryData<SnippetResponse[]>([
        'snippets',
        'list',
      ]);
      if (previous) {
        queryClient.setQueryData<SnippetResponse[]>(
          ['snippets', 'list'],
          previous.map((s) =>
            s.filename === filename ? { ...s, is_approved: false } : s
          )
        );
      }
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['snippets', 'list'], context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['snippets', 'list'] });
    },
  });
}

/**
 * Discriminated error union for {@link useBatchApproveSnippets}.
 *
 * `structured: true` — the backend returned a 400 with a parseable
 * {@link BatchApprovalErrorResponse} payload (missing DB rows / disk files).
 * `structured: false` — any other failure (network error, 5xx, etc.); the
 * raw value is preserved in `raw` for logging.
 */
export type BatchApproveError =
  | (Error & { structured: true; detail: BatchApprovalErrorResponse })
  | (Error & { structured: false; raw: unknown });

/**
 * Mutation: batch-approve snippets via PATCH /approvals (idempotent).
 *
 * Returns a {@link BatchApprovalResponse} on success (200) or throws a
 * {@link BatchApproveError} so callers can narrow on `err.structured`
 * to render per-category badges (400) vs. a generic fallback message.
 */
export function useBatchApproveSnippets() {
  const queryClient = useQueryClient();
  return useMutation<
    BatchApprovalResponse,
    BatchApproveError,
    SnippetBatchApproveRequest
  >({
    mutationFn: async (body) => {
      try {
        const { data } = await apiClient.patch<BatchApprovalResponse>(
          `${SNIPPETS_BASE}/approvals`,
          body
        );
        return data;
      } catch (err: unknown) {
        const detail =
          err instanceof ApiError && err.status === 400
            ? extractBatchApprovalError(err.data)
            : null;
        if (detail) {
          const batchError: BatchApproveError = Object.assign(
            new Error('Batch approval failed'),
            {
              structured: true as const,
              detail,
            }
          );
          throw batchError;
        }
        const batchError: BatchApproveError = Object.assign(
          new Error('Batch approval failed'),
          {
            structured: false as const,
            raw: err,
          }
        );
        throw batchError;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['snippets', 'list'] });
    },
  });
}

/**
 * Fetch per-deployment capability flags for the snippets plugin.
 *
 * The capabilities response carries no privileged data, so this hook is
 * safe to call for any authenticated user. The 5-minute ``staleTime``
 * reflects that deployment flags do not change at runtime — refetching on
 * every tab focus would be wasteful.
 */
export function useSnippetsCapabilities() {
  return useQuery<SnippetsCapabilitiesResponse>({
    queryKey: ['snippets', 'capabilities'],
    queryFn: async () => {
      const { data } = await apiClient.get<SnippetsCapabilitiesResponse>(
        `${SNIPPETS_BASE}/capabilities`
      );
      return data;
    },
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Mutation: trigger a manual refresh of the snippets cache from disk.
 *
 * Admin + deployment-gated by ``manual_sync_enabled``; callers should
 * gate the affordance on both ``isAdmin`` and the capabilities flag
 * before invoking. On success, every cached snippet query is invalidated
 * (``['snippets']`` prefix) since refresh can add, update, or remove rows.
 */
export function useRefreshSnippets() {
  const queryClient = useQueryClient();
  return useMutation<RefreshResponse, ApiError, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<RefreshResponse>(
        `${SNIPPETS_BASE}/refresh`
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['snippets'] });
    },
  });
}
