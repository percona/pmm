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
 * The topology document, served by pmm-managed at `/v1/om/topology`.
 *
 * Paired with `inventoryHooks.ts`, and kept apart from it for one reason worth stating:
 * the two sources are nothing alike. This document is PMM's own derivation from its
 * inventory and VictoriaMetrics, rebuilt per request in about a tenth of a second, and
 * it never touches a host. The estate over there is what SEP's probe found by running a
 * payload on the hosts themselves.
 *
 * That difference is why the triggers must not look alike. `POST
 * /topology/runs:collect` recomputes a document from data PMM already holds and answers
 * with a terminal status; `POST /inventory/runs:trigger` dispatches a Nomad job per host,
 * takes tens of seconds, and has to be polled. Keeping the hooks in separate files is
 * the cheapest way to keep them from being written alike.
 *
 * The transport both share is in `api.ts`; the pure derivations this document feeds are
 * in `topology.ts`.
 */

import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query';
import { isRunActive, request } from './api';
import type {
  OmTopologyResponse,
  OmTopologyRun,
  OmTopologyRunAccepted,
} from './types';

/** Rows fetched for the run history. The API caps `limit` at 100. */
export const OM_TOPOLOGY_RUNS_LIMIT = 25;

/** Poll interval while a collection run is in flight (ms). */
const RUN_POLL_MS = 3000;
// Idle cadence for the document and the run history. pmm-managed collects on its own
// schedule, so both change without anyone here asking; without a poll the page only
// ever shows what it happened to load with.
const SNAPSHOT_POLL_MS = 30000;

const TOPOLOGY_KEY = ['om', 'topology'] as const;
const RUNS_KEY = ['om', 'runs'] as const;

/**
 * The whole topology document from the latest terminal snapshot.
 *
 * One request, one document: the API assembles the tree per run, so there is no
 * per-cluster fetch and nothing to paginate. Keeping the previous page in place stops
 * the table collapsing to a spinner while a refetch is in flight.
 */
export function useOmTopology() {
  return useQuery<OmTopologyResponse>({
    queryKey: TOPOLOGY_KEY,
    placeholderData: keepPreviousData,
    queryFn: () => request<OmTopologyResponse>('/topology'),
    // pmm-managed rebuilds the document on its own timer, so a page left open goes
    // stale against a fleet that has moved on. Nothing else refetches it: the trigger's
    // invalidation only covers a sync the reader started themselves.
    refetchInterval: SNAPSHOT_POLL_MS,
  });
}

/**
 * Run history, newest first.
 *
 * Polls while a collection run is still going, which is also what makes the topology
 * refresh after a triggered run lands.
 */
export function useOmTopologyRuns(limit: number = OM_TOPOLOGY_RUNS_LIMIT) {
  return useQuery<OmTopologyRun[]>({
    queryKey: [...RUNS_KEY, limit],
    queryFn: async () => {
      const { runs } = await request<{ runs: OmTopologyRun[] }>(
        `/topology/runs?limit=${limit}`
      );
      // gRPC-Gateway always answers with an object, so the list arrives wrapped.
      return runs ?? [];
    },
    // Fast while a run is in flight, slow otherwise rather than off: runs are also
    // started by pmm-managed's own timer and by other readers, and a history that only
    // updates on reload cannot show them.
    //
    // `data[0]` is enough here, unlike on the inventory side. A collection pass takes a
    // global lock, so there is never a second one to miss.
    refetchInterval: (query) =>
      isRunActive(query.state.data?.[0]?.status)
        ? RUN_POLL_MS
        : SNAPSHOT_POLL_MS,
  });
}

/**
 * Rebuild the topology document now.
 *
 * pmm-managed does this in the request — inventory is a local database read and
 * VictoriaMetrics is on localhost — so the response already carries a terminal status
 * rather than a `202` to poll.
 */
export function useTriggerOmTopologyRun() {
  const queryClient = useQueryClient();
  return useMutation<OmTopologyRunAccepted, Error, void>({
    // No body. The method declares `body: "*"`, and grpc-gateway's generated decoder
    // treats io.EOF as an empty request rather than an error, so an empty
    // TriggerTopologyCollectionRequest is what binds either way.
    mutationFn: () =>
      request<OmTopologyRunAccepted>('/topology/runs:collect', {
        method: 'POST',
      }),
    // onSettled, not onSuccess. A 409 means pmm-managed's own timer holds the lock, so
    // a run is in flight and the document is about to change -- the one case where a
    // refetch is most obviously wanted, and the one the success-only path skipped. The
    // request is cheap and the answer is idempotent, so refetching after a failure that
    // is not a conflict costs nothing either.
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: RUNS_KEY });
      queryClient.invalidateQueries({ queryKey: TOPOLOGY_KEY });
    },
  });
}

/** Refresh everything the snapshot backs, once a run reaches a terminal status. */
export function useInvalidateOmTopologySnapshot() {
  const queryClient = useQueryClient();
  return () => {
    queryClient.invalidateQueries({ queryKey: TOPOLOGY_KEY });
    queryClient.invalidateQueries({ queryKey: RUNS_KEY });
  };
}
