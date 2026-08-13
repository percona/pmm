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

/**
 * SEP's `pom_discovery` app — the on-host probe sweeps, read straight from SEP.
 *
 * This is the one part of POM that does not come from pmm-managed, and it cannot:
 * the sweep runs a payload on each database host over Nomad, which is SEP's half of
 * the split. pmm-managed pulls the *facts* a sweep produced (`GET /facts`, merged
 * into the topology document as `installed_version`, `config_path` and `argv`) but
 * does not proxy the run history, so the page asks the app itself.
 *
 * Two consequences, both of which the Discovery route is wired for:
 *
 * - **A SEP bearer is required.** `apiClient` mints and attaches it, and the route is
 *   mounted inside `SepPage` so the exchange has happened before this renders. The
 *   rest of POM stays on pmm-managed and needs none of it.
 * - **SEP has to be up.** A page of probe sweeps is meaningless without the app that
 *   records them, so an unreachable SEP is reported here rather than hidden.
 *
 * The wire shape is snake_case and stays that way: `apiClient` runs no case
 * converter, unlike PMM's own axios instance.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ApiError, apiClient } from '@sep/api';
import type { PomProbeAccepted, PomProbeRun, PomProbeRunDetail } from './types';

/** The app's mount under `@sep/api`'s `/api` base URL. */
const PROBE_BASE = '/apps/pom_discovery';

/** Rows fetched for the sweep history. The API caps `limit` at 100. */
export const POM_PROBE_RUNS_LIMIT = 25;

/** Poll interval while a sweep is in flight (ms). A sweep takes tens of seconds. */
const PROBE_POLL_MS = 5000;
// Idle cadence. Sweeps are mostly started by the app's own Celery beat, not from this
// page, so a history that stops refetching once nothing is in flight never learns about
// them -- the newest sweep simply never appears until a reload.
const PROBE_IDLE_POLL_MS = 30000;

const probeRunsKey = ['pom', 'probe', 'runs'] as const;

/** True while a sweep has not reached a terminal status. */
export function isProbeRunActive(status: string | undefined): boolean {
  return status === 'running';
}

/**
 * Sweep history, newest first.
 *
 * Polls while the newest sweep is still going, which is what makes the trigger
 * button's progress visible without any extra wiring.
 */
export function usePomProbeRuns(limit: number = POM_PROBE_RUNS_LIMIT) {
  return useQuery<PomProbeRun[]>({
    queryKey: [...probeRunsKey, limit],
    queryFn: async () => {
      // The app answers with a bare list, not an envelope.
      const { data } = await apiClient.get<PomProbeRun[]>(
        `${PROBE_BASE}/runs`,
        { params: { limit } }
      );
      return data ?? [];
    },
    refetchInterval: (query) =>
      isProbeRunActive(query.state.data?.[0]?.status)
        ? PROBE_POLL_MS
        : PROBE_IDLE_POLL_MS,
  });
}

/**
 * One sweep in full: its per-service records and every fact it collected.
 *
 * Fetched only when a run is unfolded — `enabled` keeps it off the wire until then —
 * because this is the payload the list endpoint deliberately does not carry. A
 * finished sweep never changes, so it is cached indefinitely; only a running one is
 * worth re-reading, and the list's own poll is what surfaces its terminal status.
 */
export function usePomProbeRun(
  runId: string | undefined,
  enabled = true,
  active = false
) {
  return useQuery<PomProbeRunDetail>({
    queryKey: [...probeRunsKey, runId],
    enabled: Boolean(runId) && enabled,
    // A finished sweep never changes, so it is cached indefinitely. One still in
    // flight does: the app writes its per-service records only when the sweep ends
    // (service.py, `finished.nodes = outcome.nodes`), so until then this endpoint
    // answers with an empty list. Caching that forever is what left a completed
    // sweep reporting "no per-service detail" until the page was reloaded.
    staleTime: active ? 0 : Infinity,
    // Poll while the payload itself still reports a non-terminal status. That is what
    // replaces a mid-sweep answer with the finished one; keying it off the cached
    // detail rather than the caller's copy means it stops on the first terminal read,
    // whatever the list happens to know.
    refetchInterval: (query) =>
      isProbeRunActive(query.state.data?.status) ? PROBE_POLL_MS : false,
    queryFn: async () => {
      const { data } = await apiClient.get<PomProbeRunDetail>(
        `${PROBE_BASE}/runs/${runId}`
      );
      return data;
    },
  });
}

/**
 * Queue a probe sweep.
 *
 * Answers `202` and returns immediately: the sweep dispatches Nomad jobs and takes
 * tens of seconds, so it is never performed on the request. `409` means one is
 * already in flight — the expected answer to a double click, not a failure — and the
 * status survives onto `ApiError` so the button can tell the two apart.
 */
export function useTriggerPomProbe() {
  const queryClient = useQueryClient();
  return useMutation<PomProbeAccepted, Error, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<PomProbeAccepted>(
        `${PROBE_BASE}/runs`,
        {}
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: probeRunsKey });
    },
  });
}

/** Whether an error is the app saying a sweep is already running. */
export function isProbeConflict(error: unknown): boolean {
  return error instanceof ApiError && error.status === 409;
}
