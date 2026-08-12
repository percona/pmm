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

import { useState } from 'react';
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import {
  apiClient,
  DEFAULT_PLUGIN_LIST_LIMIT,
  normalizePluginListResponse,
  type PluginListResult,
  type PaginatedPluginList,
} from '@sep/api';
import { SNIPPETS_PLUGINS_API_BASE } from '@sep/framework';
import type {
  AtwBatchExecuteResponse,
  AtwBatchExecuteWrite,
  AtwCategoryListing,
  AtwConfig,
  AtwIncident,
  AtwIncidentExecution,
  AtwIncidentUpdate,
  AtwIncidentWrite,
  AtwMergedSchema,
  AtwPage,
  AtwPageParams,
  AtwSendJobWrite,
  AtwSendLog,
  AtwSendLogDetail,
  AtwSnippetSearchRow,
  AtwSnippetSummary,
} from './types';

const ATW_BASE = '/apps/atw';

const ATW_STALE_TIME_MS = 5 * 60 * 1000;

/** Poll interval while any incident execution is still running (ms). */
const EXECUTIONS_POLL_MS = 5000;

/** Non-terminal task statuses — poll the execution list while any row is here. */
const RUNNING_TASK_STATUSES: ReadonlySet<
  NonNullable<AtwIncidentExecution['task_status']>
> = new Set(['running', 'pending']);

/** Poll interval while a diagnostics send is still in flight (ms). */
const SEND_JOB_POLL_MS = 2000;

/** Non-terminal send statuses — poll while an attempt is still here. */
const ACTIVE_SEND_STATUSES: ReadonlySet<AtwSendLog['status']> = new Set([
  'pending',
  'running',
]);

/** Rows per page for both incident and execution lists. */
export const ATW_PAGE_SIZE = 20;

const incidentsKey = ['atw', 'incidents'] as const;

function incidentExecutionsKey(incidentId: string) {
  return ['atw', 'incidents', incidentId, 'executions'] as const;
}

function sendJobsKey(incidentId: string) {
  return ['atw', 'incidents', incidentId, 'send-jobs'] as const;
}

/** Return whether a send attempt has not yet reached a terminal status. */
export function isSendJobActive(job: AtwSendLog | undefined | null): boolean {
  return Boolean(job && ACTIVE_SEND_STATUSES.has(job.status));
}

/**
 * Read a send attempt's evidence under the shape the orchestrator writes.
 *
 * The column is free-form JSON, so the generated client types it opaquely; this
 * is the single place that view is applied.
 */
export function sendJobDetail(
  job: AtwSendLog | undefined | null
): AtwSendLogDetail {
  return (job?.detail ?? {}) as AtwSendLogDetail;
}

// ── Category browser ─────────────────────────────────────────────────────

export function useAtwCategories() {
  return useQuery<AtwCategoryListing[]>({
    queryKey: ['atw', 'categories'],
    queryFn: async () => {
      const { data } = await apiClient.get<AtwCategoryListing[]>(
        `${ATW_BASE}/`
      );
      return data;
    },
    staleTime: ATW_STALE_TIME_MS,
  });
}

// ── Snippet search ───────────────────────────────────────────────────────

/**
 * Rows fetched per search request.
 *
 * The endpoint is paginated, so a broad term can match more snippets than one
 * page holds; the picker reports that overflow rather than dropping it
 * silently. Kept at the shared app-list default for consistency with every
 * other app list — the snippets route itself would serve up to
 * `MAX_PAGINATION_LIMIT` (200), so raising this is a deliberate change, not a
 * ceiling to lift.
 */
export const ATW_SNIPPET_SEARCH_LIMIT = DEFAULT_PLUGIN_LIST_LIMIT;

/** Project a snippets-app list row onto the picker's snippet shape. */
export function toAtwSnippetSummary(
  row: AtwSnippetSearchRow
): AtwSnippetSummary {
  return {
    name: row.filename,
    title: row.title || row.filename,
    description: row.description,
  };
}

/**
 * Search every snippet by free text, independent of the ATW category taxonomy.
 *
 * Served by the snippets app's own list endpoint (the same one the Snippet
 * Manager list uses), so the picker reaches snippets the ATW category listing
 * does not expose — the `atw` metadata tag is a presentation filter on that
 * listing, never consulted by the execute path.
 *
 * `approval=approved` is load-bearing: executing an unapproved snippet is
 * rejected server-side, so offering one would only fail at execute time. The
 * query is disabled while the term is empty; callers debounce the term.
 *
 * A local hook rather than the Snippet Manager's `useSnippets`, since `@sep/atw`
 * does not depend on `@sep/snippets` and an app-to-app dependency is worse than
 * this duplication. If a third consumer appears, lift it into `@sep/framework`.
 */
export function useAtwSnippetSearch(search: string) {
  const term = search.trim();
  return useQuery<PluginListResult<AtwSnippetSummary>>({
    queryKey: ['atw', 'snippet-search', term],
    enabled: term.length > 0,
    staleTime: ATW_STALE_TIME_MS,
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { data } = await apiClient.get<
        AtwSnippetSearchRow[] | PaginatedPluginList<AtwSnippetSearchRow>
      >(`${SNIPPETS_PLUGINS_API_BASE}/`, {
        params: {
          search: term,
          approval: 'approved',
          offset: 0,
          limit: ATW_SNIPPET_SEARCH_LIMIT,
        },
      });
      const page = normalizePluginListResponse<AtwSnippetSearchRow>(data);
      return { ...page, items: page.items.map(toAtwSnippetSummary) };
    },
  });
}

// ── Incident CRUD ────────────────────────────────────────────────────────

/**
 * List incidents newest-first, one page at a time.
 *
 * Returns the whole envelope rather than just `items`: the caller needs `total`
 * to render pagination controls, and keeping the previous page in place while
 * the next one loads stops the list collapsing to a spinner on every page flip.
 */
export function useAtwIncidents(
  page: AtwPageParams = { offset: 0, limit: ATW_PAGE_SIZE }
) {
  return useQuery<AtwPage<AtwIncident>>({
    queryKey: [...incidentsKey, page],
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { data } = await apiClient.get<AtwPage<AtwIncident>>(
        `${ATW_BASE}/incidents/`,
        {
          params: { offset: page.offset, limit: page.limit },
        }
      );
      return data;
    },
  });
}

export function useAtwIncident(incidentId: string | undefined) {
  return useQuery<AtwIncident>({
    queryKey: ['atw', 'incidents', incidentId],
    enabled: Boolean(incidentId),
    queryFn: async () => {
      const { data } = await apiClient.get<AtwIncident>(
        `${ATW_BASE}/incidents/${incidentId}`
      );
      return data;
    },
  });
}

export function useCreateAtwIncident() {
  const queryClient = useQueryClient();
  return useMutation<AtwIncident, Error, AtwIncidentWrite>({
    mutationFn: async (body) => {
      const { data } = await apiClient.post<AtwIncident>(
        `${ATW_BASE}/incidents/`,
        body
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: incidentsKey });
    },
  });
}

export function useUpdateAtwIncident() {
  const queryClient = useQueryClient();
  return useMutation<
    AtwIncident,
    Error,
    { incidentId: string; body: AtwIncidentUpdate }
  >({
    mutationFn: async ({ incidentId, body }) => {
      const { data } = await apiClient.patch<AtwIncident>(
        `${ATW_BASE}/incidents/${incidentId}`,
        body
      );
      return data;
    },
    onSuccess: () => {
      invalidateAtwIncidentQueries(queryClient);
    },
  });
}

export function useDeleteAtwIncident() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, string>({
    mutationFn: async (incidentId) => {
      await apiClient.delete(`${ATW_BASE}/incidents/${incidentId}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: incidentsKey });
    },
  });
}

function invalidateAtwIncidentQueries(
  queryClient: ReturnType<typeof useQueryClient>
) {
  queryClient.invalidateQueries({ queryKey: incidentsKey });
}

function useAtwIncidentLifecycleAction(
  action: 'close' | 'reopen',
  onDone: (incidentId: string) => void
) {
  const queryClient = useQueryClient();
  return useMutation<AtwIncident, Error, string>({
    mutationFn: async (incidentId) => {
      const { data } = await apiClient.post<AtwIncident>(
        `${ATW_BASE}/incidents/${incidentId}/${action}/`
      );
      return data;
    },
    onSuccess: () => {
      invalidateAtwIncidentQueries(queryClient);
    },
    onSettled: (_data, _error, incidentId) => {
      onDone(incidentId);
    },
  });
}

/** Shared close/reopen mutations, error text, and in-flight ids for list and workspace. */
export function useAtwIncidentLifecycle() {
  const [pendingIncidentIds, setPendingIncidentIds] = useState<
    ReadonlySet<string>
  >(() => new Set());

  const clearPending = (incidentId: string) => {
    setPendingIncidentIds((prev) => {
      const next = new Set(prev);
      next.delete(incidentId);
      return next;
    });
  };

  const closeMutation = useAtwIncidentLifecycleAction('close', clearPending);
  const reopenMutation = useAtwIncidentLifecycleAction('reopen', clearPending);

  const reset = () => {
    closeMutation.reset();
    reopenMutation.reset();
  };

  return {
    close: (incidentId: string) => {
      reopenMutation.reset();
      setPendingIncidentIds((prev) => new Set(prev).add(incidentId));
      closeMutation.mutate(incidentId);
    },
    reopen: (incidentId: string) => {
      closeMutation.reset();
      setPendingIncidentIds((prev) => new Set(prev).add(incidentId));
      reopenMutation.mutate(incidentId);
    },
    reset,
    error: closeMutation.isError
      ? (closeMutation.error?.message ?? 'Failed to close incident')
      : reopenMutation.isError
        ? (reopenMutation.error?.message ?? 'Failed to reopen incident')
        : null,
    isPending: (incidentId: string) => pendingIncidentIds.has(incidentId),
  };
}

// ── Merged execution schema ──────────────────────────────────────────────

/**
 * Fetch the merged execution schema for the selected snippet filenames.
 *
 * The filenames ride as repeated `snippet_filename` query params (FastAPI's
 * list-param form), so `paramsSerializer` emits `?snippet_filename=a&…=b`
 * rather than axios' default bracket notation. The query is disabled while no
 * snippet is selected.
 */
export function useAtwMergedSchema(snippetFilenames: string[]) {
  const names = [...new Set(snippetFilenames)];
  return useQuery<AtwMergedSchema>({
    queryKey: ['atw', 'execution-schema', names],
    enabled: names.length > 0,
    staleTime: ATW_STALE_TIME_MS,
    queryFn: async () => {
      const { data } = await apiClient.get<AtwMergedSchema>(
        `${ATW_BASE}/execution-schema/`,
        {
          params: { snippet_filename: names },
          paramsSerializer: { indexes: null },
        }
      );
      return data;
    },
  });
}

// ── Batch execution ──────────────────────────────────────────────────────

export function useAtwBatchExecute(incidentId: string) {
  const queryClient = useQueryClient();
  return useMutation<AtwBatchExecuteResponse, Error, AtwBatchExecuteWrite>({
    mutationFn: async (body) => {
      const { data } = await apiClient.post<AtwBatchExecuteResponse>(
        `${ATW_BASE}/incidents/${incidentId}/executions/`,
        body
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: incidentExecutionsKey(incidentId),
      });
    },
  });
}

// ── Incident execution history ───────────────────────────────────────────

/**
 * List an incident's recorded executions, newest-first. Polls while any row on
 * the current page is still running, so batch progress reflects each task
 * independently — one failing task never stalls the others.
 *
 * Only the current page is watched: a run on a later page is not polled until
 * the user navigates to it. Newest-first ordering keeps freshly-started runs on
 * page 1, where they are.
 */
export function useAtwIncidentExecutions(
  incidentId: string | undefined,
  page: AtwPageParams = { offset: 0, limit: ATW_PAGE_SIZE }
) {
  return useQuery<AtwPage<AtwIncidentExecution>>({
    queryKey: incidentId
      ? [...incidentExecutionsKey(incidentId), page]
      : ['atw', 'incidents', 'executions', page],
    enabled: Boolean(incidentId),
    placeholderData: keepPreviousData,
    queryFn: async () => {
      const { data } = await apiClient.get<AtwPage<AtwIncidentExecution>>(
        `${ATW_BASE}/incidents/${incidentId}/executions/`,
        { params: { offset: page.offset, limit: page.limit } }
      );
      return data;
    },
    refetchInterval: (query) => {
      const rows = query.state.data?.items;
      if (!rows) {
        return false;
      }
      const anyRunning = rows.some(
        (row) =>
          row.task_status !== null &&
          row.task_status !== undefined &&
          RUNNING_TASK_STATUSES.has(row.task_status)
      );
      return anyRunning ? EXECUTIONS_POLL_MS : false;
    },
  });
}

// ── Diagnostics send ─────────────────────────────────────────────────────

/**
 * Probe whether a diagnostics receiver is configured, so the Send action can
 * carry the reasons it is unavailable. Returns no reasons on any error: a
 * transient config blip should not silently withhold the action, and the POST's
 * own 503 gate remains the real guard.
 */
export function useAtwConfig() {
  return useQuery<AtwConfig>({
    queryKey: ['atw', 'config'],
    queryFn: async () => {
      try {
        const { data } = await apiClient.get<AtwConfig>(`${ATW_BASE}/config/`);
        return data;
      } catch {
        return { send_disabled_reasons: [] };
      }
    },
    staleTime: Infinity,
    retry: false,
  });
}

export function useStartSendJob(incidentId: string) {
  const queryClient = useQueryClient();
  return useMutation<AtwSendLog, Error, AtwSendJobWrite>({
    mutationFn: async (body) => {
      const { data } = await apiClient.post<AtwSendLog>(
        `${ATW_BASE}/incidents/${incidentId}/send-jobs/`,
        body
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: sendJobsKey(incidentId) });
    },
  });
}

/** Poll one send attempt until it reaches a terminal status. */
export function useAtwSendJob(incidentId: string, jobId: string | null) {
  return useQuery<AtwSendLog>({
    queryKey: [...sendJobsKey(incidentId), jobId],
    enabled: Boolean(jobId),
    queryFn: async () => {
      const { data } = await apiClient.get<AtwSendLog>(
        `${ATW_BASE}/incidents/${incidentId}/send-jobs/${jobId}`
      );
      return data;
    },
    refetchInterval: (query) =>
      isSendJobActive(query.state.data) ? SEND_JOB_POLL_MS : false,
  });
}

/** List an incident's send attempts, newest-first, polling while any is active. */
export function useAtwSendJobs(incidentId: string | undefined) {
  return useQuery<AtwPage<AtwSendLog>>({
    queryKey: incidentId
      ? sendJobsKey(incidentId)
      : ['atw', 'incidents', 'send-jobs'],
    enabled: Boolean(incidentId),
    queryFn: async () => {
      const { data } = await apiClient.get<AtwPage<AtwSendLog>>(
        `${ATW_BASE}/incidents/${incidentId}/send-jobs/`
      );
      return data;
    },
    refetchInterval: (query) =>
      query.state.data?.items.some((row) => isSendJobActive(row))
        ? SEND_JOB_POLL_MS
        : false,
  });
}
