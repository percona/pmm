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
import { apiClient } from '@sep/api';
import type {
  AtwBatchExecuteResponse,
  AtwBatchExecuteWrite,
  AtwCategoryListing,
  AtwIncident,
  AtwIncidentExecution,
  AtwIncidentUpdate,
  AtwIncidentWrite,
  AtwMergedSchema,
} from './types';

const ATW_BASE = '/apps/atw';

const ATW_STALE_TIME_MS = 5 * 60 * 1000;

/** Poll interval while any incident execution is still running (ms). */
const EXECUTIONS_POLL_MS = 5000;

/** Non-terminal task statuses — poll the execution list while any row is here. */
const RUNNING_TASK_STATUSES = new Set(['running', 'pending']);

interface PaginatedResponse<T> {
  items: T[];
  total: number;
  offset: number;
  limit: number;
}

const incidentsKey = ['atw', 'incidents'] as const;

function incidentExecutionsKey(incidentId: string) {
  return ['atw', 'incidents', incidentId, 'executions'] as const;
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

// ── Incident CRUD ────────────────────────────────────────────────────────

export function useAtwIncidents() {
  return useQuery<AtwIncident[]>({
    queryKey: incidentsKey,
    queryFn: async () => {
      const { data } = await apiClient.get<PaginatedResponse<AtwIncident>>(
        `${ATW_BASE}/incidents/`
      );
      return data.items;
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
    onSuccess: (incident) => {
      queryClient.invalidateQueries({ queryKey: incidentsKey });
      queryClient.invalidateQueries({
        queryKey: ['atw', 'incidents', incident.id],
      });
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
 * List an incident's recorded executions, newest-first. Polls while any row is
 * still running so batch progress reflects each task independently — one
 * failing task never stalls the others.
 */
export function useAtwIncidentExecutions(incidentId: string | undefined) {
  return useQuery<AtwIncidentExecution[]>({
    queryKey: incidentId
      ? incidentExecutionsKey(incidentId)
      : ['atw', 'incidents', 'executions'],
    enabled: Boolean(incidentId),
    queryFn: async () => {
      const { data } = await apiClient.get<
        PaginatedResponse<AtwIncidentExecution>
      >(`${ATW_BASE}/incidents/${incidentId}/executions/`);
      return data.items;
    },
    refetchInterval: (query) => {
      const rows = query.state.data;
      if (!rows) {
        return false;
      }
      const anyRunning = rows.some((row) =>
        RUNNING_TASK_STATUSES.has(row.task_status ?? '')
      );
      return anyRunning ? EXECUTIONS_POLL_MS : false;
    },
  });
}
