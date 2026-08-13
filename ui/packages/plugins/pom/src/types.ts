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
 * Wire types for POM's read API, served by pmm-managed at `/v1/pom`.
 *
 * Hand-written rather than generated: POM is not in PMM's checked-in OpenAPI client
 * yet, so this file is the frontend's copy of the document contract. The authority is
 * `api/pom/v1/pom.proto`; regenerate from it once POM joins the `SPECS` list in
 * `api/Makefile`.
 */

/** Health verdicts the worker derives. `unknown` is the common case, not an edge. */
/** Whether the exporter reached the service on this run. */
export type PomServiceStatus = 'UP' | 'DOWN';

/** What the process is in its cluster. */
export type PomProcessRole = 'mongod' | 'mongos' | 'configsvr' | 'shardsvr';

/**
 * Why a field is null. The worker exports these as constants precisely because
 * the frontend matches on them.
 */
export type PomUnavailableReason =
  | 'service_not_observed'
  | 'metric_not_collected'
  | 'no_version_catalog'
  | 'not_applicable';

/**
 * One monitored MongoDB service.
 *
 * Two sentinel conventions the table depends on, and they differ on purpose:
 *
 * - `cpu_usage_percent` / `connections_free_percent` are **-1 when not measured**, never
 *   null and never 0 — zero CPU is a real reading, so the numeric sentinel is what
 *   keeps "idle" and "unknown" apart in a numeric column.
 * - `replication_lag_seconds` / `oplog_window_seconds` are **null when they do not apply**
 *   — a router and a standalone have no replica-set oplog. Null means "not a thing
 *   here", which is not the same as -1's "we could not measure it".
 */
export interface PomService {
  service_name: string;
  host: string | null;
  endpoint: string | null;
  service_id: string | null;
  service_type: string | null;
  version: string | null;
  vendor: string | null;
  edition: string | null;
  replication_set: string | null;
  state: string | null;
  status: PomServiceStatus;
  cpu_usage_percent: number;
  connections_free_percent: number;
  process_role: PomProcessRole;
  replication_lag_seconds: number | null;
  oplog_window_seconds: number | null;

  /**
   * Probe-only, and null wherever SEP's `pom_discovery` app has not run.
   *
   * `installed_version` is the binary on disk, which is not necessarily the server
   * that is running: divergence from `version` is the upgraded-but-not-restarted
   * case, and no metric anywhere carries it.
   */
  installed_version: string | null;
  config_path: string | null;
  argv: string | null;
}

/** One cluster or replica set. `name` is null when its services carry no label. */
export interface PomCluster {
  name: string | null;
  services: PomService[];
}

/** One monitoring environment. `env_name` is null when unset. */
export interface PomEnvironment {
  env_name: string | null;
  clusters: PomCluster[];
}

/** Fleet-level counts, computed by the worker so the UI never re-derives them. */
export interface PomTopologySummary {
  environments: number;
  clusters: number;
  services_total: number;
  services_up: number;
  services_down: number;
  by_process_role: Partial<Record<PomProcessRole, number>>;
}

/** Provenance every snapshot-backed response repeats. */
export interface PomSnapshotEnvelope {
  generated_at: string;
  observed_at: string | null;
  stale: boolean;
  schema_version: number;
  run_id: string;
}

/** The whole estate, as one document. */
export interface PomTopologyResponse {
  snapshot: PomSnapshotEnvelope;
  origin_node: string | null;
  source_queries: string[];
  summary: PomTopologySummary;
  environments: PomEnvironment[];
}

/**
 * One service flattened out of the tree, carrying the two grouping keys.
 *
 * The table renders a flat list so it can sort and filter across the whole estate;
 * the nesting is preserved as columns rather than as rows.
 */
export interface PomServiceRow extends PomService {
  env_name: string | null;
  cluster_name: string | null;
}

/**
 * One cluster rolled up out of its services, carrying its environment.
 *
 * The overview answers "what is out there, and is any of it unhappy" one level above
 * the service table: a row per cluster, grouped by environment. Everything here is
 * derived from the same document the topology table renders, so the two can never
 * disagree -- there is no second endpoint and no second snapshot.
 */
export interface PomClusterRow {
  env_name: string | null;
  cluster_name: string | null;
  /** The cluster's own services, so unfolding a row needs no second lookup. */
  services: PomService[];
  services_total: number;
  services_up: number;
  services_down: number;
  by_process_role: Partial<Record<PomProcessRole, number>>;
  /** Replica-set member states, counted. Empty for routers and standalones. */
  by_state: Record<string, number>;
  /** Distinct running versions. More than one is a mixed-version cluster. */
  versions: string[];
  /**
   * The worst lag in the cluster, and the tightest oplog window.
   *
   * Both are null when no service in the cluster reports one -- a sharded cluster's
   * routers have neither, and an unobserved cluster has nothing at all.
   */
  max_replication_lag_seconds: number | null;
  min_oplog_window_seconds: number | null;
}

/**
 * One environment and everything under it, ready to render as its own table.
 *
 * The overview gives each environment a table of its own rather than one table with
 * environment group rows: environments are what estates are actually divided by, and
 * a per-environment table keeps that division visible even when one of them is long
 * enough to scroll past.
 */
export interface PomEnvironmentSection {
  env_name: string | null;
  clusters: PomClusterRow[];
  services_total: number;
  services_up: number;
  services_down: number;
}

export type PomRunStatus = 'running' | 'success' | 'partial' | 'failed';

/**
 * What one run mapped and probed.
 *
 * `services_resolved` versus `probes_ok` is the diagnostic pair: resolved says
 * the executor mapping worked, probes_ok says the node answered.
 */
export interface PomRunCounts {
  services_total: number;
  services_resolved: number;
  services_orphaned: number;
  probes_ok: number;
}

export interface PomRunError {
  scope: string;
  service_name: string | null;
  code: string;
  message: string;
}

export interface PomRun {
  run_id: string;
  status: PomRunStatus;
  started_at: string;
  finished_at: string | null;
  counts: PomRunCounts;
  errors: PomRunError[];
}

export interface PomRunAccepted {
  run_id: string;
  status: PomRunStatus;
  started_at: string;
}

/**
 * What one probe sweep reached.
 *
 * `resolved` versus `answered` is the diagnostic split, and it is the pair worth
 * reading first: resolved says the service mapped to a live executor host, answered
 * says that node ran the payload. `resolved=9, answered=0` is a healthy mapping and
 * broken executors, which a single failure count would hide. Orphaned services are
 * not an error — they are services with no executor to probe.
 */
export interface PomProbeCounts {
  services_total: number;
  services_resolved: number;
  services_orphaned: number;
  services_answered: number;
}

/**
 * One sweep of SEP's `pom_discovery` app.
 *
 * Distinct from `PomRun`, which is pmm-managed rebuilding the topology document in a
 * tenth of a second from inventory and VictoriaMetrics. A sweep runs a payload on
 * every database host over Nomad and takes tens of seconds; what it collects is the
 * on-host facts no metric carries.
 */
export interface PomProbeRun {
  run_id: string;
  status: PomRunStatus;
  started_at: string;
  finished_at: string | null;
  counts: PomProbeCounts;
  facts_collected: number;
  /** Why the sweep itself failed, when it did. */
  error: string | null;
}

export interface PomProbeAccepted {
  run_id: string;
  status: PomRunStatus;
  started_at: string;
}

/**
 * One mapped service, as a sweep saw it.
 *
 * The run's counters are this list's summary: "5 of 14 answered" cannot say which
 * five, on which hosts, or which host took a minute.
 */
export interface PomProbeNode {
  /** PMM's service UUID, or null where SEP's inventory holds none. */
  service_id: string | null;
  service_name: string;
  /** The host the probe ran on; null when the service is orphaned. */
  executor_host: string | null;
  /** `name` / `address` / `orphaned` — how that host was matched, or that it was not. */
  resolution: string;
  answered: boolean;
  /**
   * The host's wall-clock, dispatch to collected output.
   *
   * Repeated across the services one host serves, because one dispatch covers all of
   * them: there is no per-service time to report.
   */
  duration_seconds: number | null;
  facts_collected: number;
  /** The host-level failure, when its probe failed. */
  error: string | null;
}

/** One fact the probe read off a host. */
export interface PomProbeFact {
  service_id: string;
  field: string;
  value: unknown;
  observed_at: string;
}

/**
 * One sweep in full, from `GET /runs/{id}`.
 *
 * The list endpoint omits `nodes` and `facts` — a sweep's facts run to a few hundred
 * records, and a 25-run history carrying all of them would be an order of magnitude
 * larger to serve a page that shows one run at a time.
 */
export interface PomProbeRunDetail extends PomProbeRun {
  nodes: PomProbeNode[];
  facts: PomProbeFact[];
}
