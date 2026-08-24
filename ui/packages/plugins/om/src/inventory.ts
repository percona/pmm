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
 * Turning OM's estate into rows, as plain functions.
 *
 * Kept out of the components and out of the hooks because this is the part with real
 * logic in it - the join, and the three-way answer to "is there a database here" - and
 * it is testable without rendering anything or mocking a fetch. The same reasoning
 * `toClusterRows` and `toServiceRows` already follow.
 */

import type {
  OmHostDatabaseState,
  OmHostRow,
  OmInventoryHost,
  OmInventoryService,
  OmRepoReachability,
  OmServiceInventoryRow,
  OmServiceRow,
  OmUnavailableReason,
} from './types';

/**
 * What the estate query is currently doing, from a rendering page's point of view.
 *
 * Three states rather than the boolean this was, because a missing row means three
 * different things and only one of them is a fact about the estate. `ready` is the
 * only state in which "OM has no row for this service" is something the page knows.
 */
export type OmEstateStatus = 'ready' | 'pending' | 'unavailable';

/**
 * Why a service has no estate row, given what the estate query is doing.
 *
 * The `pending` answer is the one worth having. A page that reported
 * `not_in_inventory` while the request was still in flight would state the estate's
 * contents before reading them, and it would do so on every first paint, because the
 * topology document comes back in a tenth of a second and the estate does not.
 */
export function missingRowReason(estate: OmEstateStatus): OmUnavailableReason {
  switch (estate) {
    case 'pending':
      return 'inventory_pending';
    case 'unavailable':
      return 'inventory_unavailable';
    default:
      return 'not_in_inventory';
  }
}

/**
 * Which of the three database states a host is in.
 *
 * The page's headline question is "which hosts have no database", and answering it
 * from `services.length` alone would be wrong in a way that matters. PMM's inventory
 * cannot tell a bare pmm-client host from an arbiter: same node type, same agents, no
 * services, `distro: null` on both. So a host with no *registered* service may still
 * be running a mongod - PMM simply cannot authenticate against an arbiter to register
 * one. Reporting that host as empty would invite someone to install a database over a
 * port already in use.
 *
 * Hence three states rather than two, and the middle one is the reason the probe
 * reports processes PMM never asked about.
 */
export function databaseState(host: OmInventoryHost): OmHostDatabaseState {
  if (host.services.length > 0) {
    return 'has_service';
  }
  if (host.unregistered_mongods.length > 0) {
    return 'unregistered_only';
  }
  return 'installable';
}

/**
 * Read the repository check out of a host's document.
 *
 * It lives in `observed` rather than in a named field because the probe's attributes
 * are deliberately not enumerated in the proto - a new one appears the day it is
 * collected. The cost is exactly this: reading it back needs a shape check rather than
 * a type, since nothing upstream guarantees the key is there or that it is an object.
 *
 * @returns the reachability record, or null when this host has never reported one.
 */
export function repoReachability(
  host: OmInventoryHost
): OmRepoReachability | null {
  const repo = host.observed?.repo;
  if (!repo || typeof repo !== 'object' || Array.isArray(repo)) {
    return null;
  }
  const record = repo as Record<string, unknown>;
  return {
    url: typeof record.url === 'string' ? record.url : null,
    reachable: record.reachable === true,
    status_code:
      typeof record.status_code === 'number' ? record.status_code : null,
    latency_ms:
      typeof record.latency_ms === 'number' ? record.latency_ms : null,
    proxy: typeof record.proxy === 'string' ? record.proxy : null,
    error: typeof record.error === 'string' ? record.error : null,
  };
}

/** Build the Hosts table's rows, with everything the page derives from each host. */
export function toHostRows(hosts: OmInventoryHost[] | undefined): OmHostRow[] {
  if (!hosts) {
    return [];
  }
  return hosts.map((host) => ({
    ...host,
    database_state: databaseState(host),
    service_count: host.services.length,
    repo: repoReachability(host),
  }));
}

/**
 * Join PMM's snapshot to OM's estate, one row per service.
 *
 * The join is a map lookup and nothing more, which is the payoff of keying the estate
 * on PMM's own service id: there is no matching function, no name or address
 * heuristic, and so nothing to get wrong. Everywhere else in this system that two
 * sources have to be lined up - a node to its executor, a mongod to a service - costs
 * a matching rule and a way to be wrong about it.
 *
 * Rows come from the snapshot, not from the estate. A service PMM registered since the
 * last sweep has no estate row yet and must still appear, with its probe columns
 * unavailable rather than the row missing; a service the estate holds that PMM no
 * longer has is stale and belongs on the Hosts page's delete action, not here.
 *
 * A snapshot service with no id joins to nothing rather than to the first estate row
 * without one. The snapshot types it nullable and the estate does not, so the two
 * nulls would otherwise meet in the map and match - which is the one way a keyed join
 * can still attach a probe's answers to the wrong service.
 */
export function joinServiceInventory(
  services: OmServiceRow[],
  inventory: OmInventoryService[] | undefined
): OmServiceInventoryRow[] {
  const byServiceId = new Map(
    (inventory ?? []).map((entry) => [entry.service_id, entry])
  );
  return services.map((service) => ({
    ...service,
    inventory:
      (service.service_id ? byServiceId.get(service.service_id) : null) ?? null,
  }));
}

/**
 * How long ago something happened, in seconds.
 *
 * Ages rather than timestamps are what the pages show: the estate is upserted, so
 * every attribute is only meaningful beside how old it is, and "3 days ago" is read
 * faster than a date that has to be subtracted from today.
 *
 * @returns the age in seconds, or null when the timestamp is absent or unparseable.
 */
export function ageSeconds(
  timestamp: string | null | undefined,
  now: number = Date.now()
): number | null {
  if (!timestamp) {
    return null;
  }
  const parsed = Date.parse(timestamp);
  if (Number.isNaN(parsed)) {
    return null;
  }
  // Clamped at zero rather than reported negative: a host clock a few seconds ahead of
  // the browser's is ordinary, and "in -4 seconds" reads as a bug in this page.
  return Math.max(0, Math.round((now - parsed) / 1000));
}

/**
 * Whether a row is currently failing its probe.
 *
 * Reads `failing_since` rather than `consecutive_failures > 0`, because the two answer
 * different questions: the counter is reset by a success, but so is the timestamp, and
 * only the timestamp says *since when*. A row that has failed once and not yet
 * succeeded is failing; the count is how badly.
 */
export function isFailing(host: {
  freshness: { failing_since?: string | null };
}): boolean {
  return host.freshness.failing_since != null;
}
