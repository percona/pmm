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

import { SCAN_ERROR_KIND } from './constants';
import { runDurationSeconds } from './format';
import type {
  OmHostDatabaseState,
  OmHostRow,
  OmInventoryFreshness,
  OmInventoryHost,
  OmInventoryRun,
  OmInventoryService,
  OmRepoReachability,
  OmScanErrorCode,
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

/**
 * Read the free space the last scan measured at the default data path, or at its
 * nearest existing ancestor on a host where it does not exist yet.
 *
 * @returns bytes free, or null when this host has never reported it.
 */
export function dataDirFreeBytes(host: OmInventoryHost): number | null {
  const free = host.observed?.data_dir_free_bytes;
  return typeof free === 'number' ? free : null;
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

/**
 * A `last_error_code` as one of the kinds this page knows, `unknown` otherwise.
 *
 * Absent and unrecognised fold together on purpose: a server older than the code
 * sends none, a newer one may send a code this build has no words for, and in both
 * cases the honest thing to show is the raw error with no advice.
 */
export function scanErrorCode(
  code: string | null | undefined
): OmScanErrorCode {
  // hasOwnProperty rather than Object.hasOwn: the PMM app compiles this plugin
  // against a library without ES2022's Object.hasOwn.
  return code != null &&
    Object.prototype.hasOwnProperty.call(SCAN_ERROR_KIND, code)
    ? (code as OmScanErrorCode)
    : 'unknown';
}

/** How long a short reason may run before it is cut, in characters. */
const SHORT_REASON_MAX = 80;

/** A failing row's failure, in the pieces the Nodes page states it with. */
export interface ScanFailure {
  code: OmScanErrorCode;
  /** The kind's human label: "Scan crashed", or "Scan failed" for unknown. */
  label: string;
  /**
   * The reason as it fits on a table row. The label when the code is known; for
   * `unknown` the label says nothing, so it is the raw error's first line instead.
   */
  shortReason: string;
  /** The whole raw error, untrimmed, or null when the server recorded none. */
  error: string | null;
  /** What to do about it, or null when there is no advice worth giving. */
  hint: string | null;
  /** The first failure after the last success, as the server sent it. */
  failingSince: string;
  /** Seconds since `failing_since`, or null if that will not parse. */
  failingForSeconds: number | null;
  consecutiveFailures: number;
  runId: string | null;
}

/**
 * Why a row's scans are failing, or null when they are not.
 *
 * Keyed on `failing_since` through `isFailing`, never on `last_success_at`: a node
 * whose scans have never succeeded has no success time but is the one most in need of
 * an explanation, and keying on the success time was what left its error off the page.
 */
export function describeScanFailure(
  freshness: OmInventoryFreshness,
  now: number = Date.now()
): ScanFailure | null {
  if (!isFailing({ freshness })) {
    return null;
  }
  const code = scanErrorCode(freshness.last_error_code);
  const kind = SCAN_ERROR_KIND[code];
  const error = freshness.last_error?.trim() || null;
  let shortReason = kind.label;
  if (code === 'unknown' && error) {
    const firstLine = error.split('\n', 1)[0].trim();
    shortReason =
      firstLine.length > SHORT_REASON_MAX
        ? `${firstLine.slice(0, SHORT_REASON_MAX - 1).trimEnd()}…`
        : firstLine;
  }
  return {
    code,
    label: kind.label,
    shortReason,
    error,
    hint: kind.hint,
    // Non-null: isFailing above is exactly `failing_since != null`.
    failingSince: freshness.failing_since as string,
    failingForSeconds: ageSeconds(freshness.failing_since, now),
    consecutiveFailures: freshness.consecutive_failures,
    runId: freshness.last_run_id || null,
  };
}

/**
 * The scan history's window, as a list rather than a switch: adding a quick
 * filter is adding a row here, nowhere else. `AutomationsScansTab` renders one
 * chip per entry, in this order.
 *
 * `minutes` is a rolling window - `now` minus that many minutes. `'today'` is not
 * rolling: it is local midnight, which is what a reader means by the word, and
 * that is a different calculation from every other row. `null` (only `all`) sends
 * no `since` at all.
 */
export interface OmRunPeriodDef {
  id: string;
  label: string;
  window: number | 'today' | null;
}

export const RUN_PERIODS: readonly OmRunPeriodDef[] = [
  { id: '15m', label: 'Last 15 minutes', window: 15 },
  { id: '30m', label: 'Last 30 minutes', window: 30 },
  { id: '1h', label: 'Last hour', window: 60 },
  { id: '4h', label: 'Last 4 hours', window: 4 * 60 },
  { id: '8h', label: 'Last 8 hours', window: 8 * 60 },
  { id: 'today', label: 'Today', window: 'today' },
  { id: 'week', label: 'Last week', window: 7 * 24 * 60 },
  { id: 'month', label: 'Last month', window: 30 * 24 * 60 },
  { id: 'all', label: 'All', window: null },
] as const;

export type OmRunPeriod = (typeof RUN_PERIODS)[number]['id'];

const RUN_PERIOD_BY_ID: Record<string, OmRunPeriodDef> = Object.fromEntries(
  RUN_PERIODS.map((def) => [def.id, def])
);

/** How many recent scans {@link expectedScanSeconds} takes the middle of. */
const EXPECTED_FROM_RUNS = 5;

/**
 * How long a scan over `scope` usually takes: the median of the most recent finished
 * scans over the same nodes, or null when there are none.
 *
 * The same scope, because a one-node scan and a full one take very different times.
 * The median, so one scan that sat behind a stuck node does not set the expectation.
 * A scan that failed outright is left out: it ended early or timed out, and either
 * way says nothing about how long one takes.
 */
export function expectedScanSeconds(
  runs: OmInventoryRun[],
  scope: string[]
): number | null {
  const key = [...scope].sort().join(',');
  const durations: number[] = [];
  for (const run of runs) {
    if (durations.length === EXPECTED_FROM_RUNS) {
      break;
    }
    const seconds = runDurationSeconds(run.start_time, run.end_time);
    if (
      seconds != null &&
      (run.status === 'RUN_STATUS_SUCCESS' ||
        run.status === 'RUN_STATUS_PARTIAL') &&
      [...run.scope].sort().join(',') === key
    ) {
      durations.push(seconds);
    }
  }
  if (durations.length === 0) {
    return null;
  }
  durations.sort((a, b) => a - b);
  const middle = Math.floor(durations.length / 2);
  return durations.length % 2 === 1
    ? durations[middle]
    : (durations[middle - 1] + durations[middle]) / 2;
}

/** How many runs to ask for when the window is unbounded (`all`). */
export const DEFAULT_RUN_LIMIT = 25;

/** How many to ask for inside a date window — the GET /runs ceiling. */
export const WINDOWED_RUN_LIMIT = 100;

/**
 * Inclusive `since` for a named window, or undefined for `all`.
 *
 * `now` is injectable so the ISO string is deterministic in tests.
 */
export function periodSince(
  period: OmRunPeriod,
  now: Date = new Date()
): string | undefined {
  const { window } = RUN_PERIOD_BY_ID[period];
  if (window === null) {
    return undefined;
  }
  if (window === 'today') {
    const midnight = new Date(now);
    midnight.setHours(0, 0, 0, 0);
    return midnight.toISOString();
  }
  return new Date(now.getTime() - window * 60 * 1000).toISOString();
}

/** Whether a window has a `since` bound at all — `all` is the only one that does not. */
export function isBoundedPeriod(period: OmRunPeriod): boolean {
  return RUN_PERIOD_BY_ID[period].window !== null;
}

export function isRunPeriod(value: string | null): value is OmRunPeriod {
  return value != null && value in RUN_PERIOD_BY_ID;
}
