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
 * Wire types for OM's read API, served by pmm-managed at `/v1/om`.
 *
 * Hand-written rather than generated: OM is not in PMM's checked-in OpenAPI client
 * yet, so this file is the frontend's copy of the document contract. The authority is
 * `api/om/v1/om.proto`; regenerate from it once OM joins the `SPECS` list in
 * `api/Makefile`.
 */

/** Health verdicts the worker derives. `unknown` is the common case, not an edge. */
/**
 * Whether the exporter reached the service on this run.
 *
 * These are proto enum names, not prose: the gateway serialises `pom.v1.ServiceStatus`
 * by name, so the wire carries `SERVICE_STATUS_UP`. Verbose, and the price of the
 * server rejecting a value it does not know rather than passing a typo through.
 */
export type OmServiceStatus =
  | 'SERVICE_STATUS_UNSPECIFIED'
  | 'SERVICE_STATUS_UP'
  | 'SERVICE_STATUS_DOWN';

/** What the process is in its cluster. */
export type OmProcessRole =
  | 'PROCESS_ROLE_UNSPECIFIED'
  | 'PROCESS_ROLE_MONGOD'
  | 'PROCESS_ROLE_MONGOS'
  | 'PROCESS_ROLE_CONFIGSVR'
  | 'PROCESS_ROLE_SHARDSVR';

/** How a host was matched to the executor client its probe ran on. */
export type OmExecutorResolution =
  | 'EXECUTOR_RESOLUTION_UNSPECIFIED'
  | 'EXECUTOR_RESOLUTION_NAME'
  | 'EXECUTOR_RESOLUTION_ADDRESS'
  | 'EXECUTOR_RESOLUTION_ORPHANED';

/**
 * How a configuration field may be overridden.
 *
 * `NESTED_ONLY` is the one to be careful with: the parent object refuses a whole-object
 * write while its children accept one, so a form must not read it as read-only.
 */
export type OmSettingReload =
  | 'SETTING_RELOAD_UNSPECIFIED'
  | 'SETTING_RELOAD_HOT'
  | 'SETTING_RELOAD_NESTED_ONLY'
  | 'SETTING_RELOAD_NOT_OVERRIDABLE';

/**
 * Why a field is null. The worker exports these as constants precisely because
 * the frontend matches on them.
 */
export type OmUnavailableReason =
  | 'service_not_observed'
  | 'metric_not_collected'
  | 'no_version_catalog'
  | 'not_applicable'
  | 'not_in_inventory'
  | 'probe_never_succeeded'
  | 'inventory_unavailable';

/**
 * One monitored MongoDB service.
 *
 * Two conventions for "no value" the table depends on, and they differ on purpose:
 *
 * - `cpu_usage_percent` / `connections_free_percent` are **-1 when not measured**, never
 *   absent and never 0 — zero CPU is a real reading, so the numeric sentinel is what
 *   keeps "idle" and "unknown" apart in a numeric column.
 * - `replication_lag_seconds` / `oplog_window_seconds` are **absent when they do not
 *   apply** — a router and a standalone have no replica-set oplog. That is a different
 *   statement from -1's "we could not measure it".
 *
 * The optional fields are `optional` scalars in `om.proto`, and protojson omits an unset
 * one rather than emitting null, so they arrive as a missing key. They are typed
 * `?: T | null` because `null` is still worth tolerating and every reader here already
 * uses `== null`, which catches both.
 */
export interface OmService {
  service_name: string;
  host?: string | null;
  endpoint?: string | null;
  service_id?: string | null;
  service_type?: string | null;
  version?: string | null;
  vendor?: string | null;
  edition?: string | null;
  replication_set?: string | null;
  state?: string | null;
  status: OmServiceStatus;
  cpu_usage_percent: number;
  connections_free_percent: number;
  process_role: OmProcessRole;
  replication_lag_seconds?: number | null;
  oplog_window_seconds?: number | null;

  /**
   * Probe-only, and null wherever SEP's `om_inventory` app has not run.
   *
   * `installed_version` is the binary on disk, which is not necessarily the server
   * that is running: divergence from `version` is the upgraded-but-not-restarted
   * case, and no metric anywhere carries it.
   */
  installed_version?: string | null;
  config_path?: string | null;
  argv?: string | null;
}

/** One cluster or replica set. `name` is null when its services carry no label. */
export interface OmCluster {
  name?: string | null;
  services: OmService[];
}

/** One monitoring environment. `env_name` is null when unset. */
export interface OmEnvironment {
  env_name?: string | null;
  clusters: OmCluster[];
}

/** Fleet-level counts, computed by the worker so the UI never re-derives them. */
export interface OmTopologySummary {
  environments: number;
  clusters: number;
  total_services: number;
  up_services: number;
  down_services: number;
  process_role_counts: Partial<Record<OmProcessRole, number>>;
}

/** Provenance every snapshot-backed response repeats. */
export interface OmTopologySnapshotEnvelope {
  generated_at: string;
  observed_at?: string | null;
  stale: boolean;
  schema_version: number;
  run_id: string;
}

/** The whole estate, as one document. */
export interface OmTopologyResponse {
  snapshot: OmTopologySnapshotEnvelope;
  origin_node?: string | null;
  source_queries: string[];
  summary: OmTopologySummary;
  environments: OmEnvironment[];
}

/**
 * One service flattened out of the tree, carrying the two grouping keys.
 *
 * The table renders a flat list so it can sort and filter across the whole estate;
 * the nesting is preserved as columns rather than as rows.
 */
export interface OmServiceRow extends OmService {
  env_name?: string | null;
  cluster_name?: string | null;
}

/**
 * One cluster rolled up out of its services, carrying its environment.
 *
 * The overview answers "what is out there, and is any of it unhappy" one level above
 * the service table: a row per cluster, grouped by environment. Everything here is
 * derived from the same document the topology table renders, so the two can never
 * disagree -- there is no second endpoint and no second snapshot.
 */
export interface OmClusterRow {
  env_name?: string | null;
  cluster_name?: string | null;
  /** The cluster's own services, so unfolding a row needs no second lookup. */
  services: OmService[];
  total_services: number;
  up_services: number;
  down_services: number;
  by_process_role: Partial<Record<OmProcessRole, number>>;
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
  max_replication_lag_seconds?: number | null;
  min_oplog_window_seconds?: number | null;
}

/**
 * One environment and everything under it, ready to render as its own table.
 *
 * The overview gives each environment a table of its own rather than one table with
 * environment group rows: environments are what estates are actually divided by, and
 * a per-environment table keeps that division visible even when one of them is long
 * enough to scroll past.
 */
export interface OmEnvironmentSection {
  env_name?: string | null;
  clusters: OmClusterRow[];
  total_services: number;
  up_services: number;
  down_services: number;
}

export type OmTopologyRunStatus =
  | 'RUN_STATUS_UNSPECIFIED'
  | 'RUN_STATUS_RUNNING'
  | 'RUN_STATUS_SUCCESS'
  | 'RUN_STATUS_PARTIAL'
  | 'RUN_STATUS_FAILED'
  /**
   * Refused before doing any work, because another refresh already held its hosts.
   *
   * Not a failure and not a success: it did nothing, deliberately. Recorded rather
   * than skipped silently so a ten-minute schedule cannot look like it fired and
   * found nothing.
   */
  | 'RUN_STATUS_SKIPPED';

/**
 * What one run mapped and probed.
 *
 * `resolved_services` versus `successful_probes` is the diagnostic pair: resolved says
 * the executor mapping worked, successful says the node answered.
 */
export interface OmTopologyRunCounts {
  total_services: number;
  resolved_services: number;
  orphaned_services: number;
  successful_probes: number;
}

export interface OmTopologyRunError {
  scope: string;
  service_name?: string | null;
  code: string;
  message: string;
}

export interface OmTopologyRun {
  run_id: string;
  status: OmTopologyRunStatus;
  start_time: string;
  end_time?: string | null;
  counts: OmTopologyRunCounts;
  errors: OmTopologyRunError[];
}

export interface OmTopologyRunAccepted {
  run_id: string;
  status: OmTopologyRunStatus;
  start_time: string;
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
export interface OmProbeCounts {
  total_services: number;
  resolved_services: number;
  orphaned_services: number;
  answered_services: number;
}

/**
 * One sweep of SEP's `om_inventory` app.
 *
 * Distinct from `OmTopologyRun`, which is pmm-managed rebuilding the topology document in a
 * tenth of a second from inventory and VictoriaMetrics. A sweep runs a payload on
 * every database host over Nomad and takes tens of seconds; what it collects is the
 * on-host facts no metric carries.
 */
export interface OmProbeRun {
  run_id: string;
  status: OmTopologyRunStatus;
  start_time: string;
  end_time?: string | null;
  counts: OmProbeCounts;
  facts_collected: number;
  /** Why the sweep itself failed, when it did. */
  error?: string | null;
}

export interface OmProbeAccepted {
  run_id: string;
  status: OmTopologyRunStatus;
  start_time: string;
}

/**
 * One mapped service, as a sweep saw it.
 *
 * The run's counters are this list's summary: "5 of 14 answered" cannot say which
 * five, on which hosts, or which host took a minute.
 */
export interface OmProbeNode {
  /** PMM's service UUID, or null where SEP's inventory holds none. */
  service_id?: string | null;
  service_name: string;
  /** The host the probe ran on; null when the service is orphaned. */
  executor_host?: string | null;
  /** `name` / `address` / `orphaned` — how that host was matched, or that it was not. */
  resolution: string;
  answered: boolean;
  /**
   * The host's wall-clock, dispatch to collected output.
   *
   * Repeated across the services one host serves, because one dispatch covers all of
   * them: there is no per-service time to report.
   */
  duration_seconds?: number | null;
  facts_collected: number;
  /** The host-level failure, when its probe failed. */
  error?: string | null;
}

/** One fact the probe read off a host. */
export interface OmProbeFact {
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
export interface OmProbeRunDetail extends OmProbeRun {
  nodes: OmProbeNode[];
  facts: OmProbeFact[];
}

/*
 * ---------------------------------------------------------------------------
 * The inventory: OM's estate, served by pmm-managed at /v1/om/inventory
 * ---------------------------------------------------------------------------
 *
 * A different source from everything above. The topology document is PMM's own
 * derivation from its inventory and VictoriaMetrics, rebuilt per request in about a
 * tenth of a second and never touching a host. The estate is what SEP's probe found by
 * running a payload on the hosts themselves, upserted row by row, and it answers for a
 * service or a host whether or not the last sweep reached it.
 *
 * Hand-kept against `api/om/v1/om.proto`, like the rest of this file: PMM's plugin
 * generates no client. The gateway runs `UseProtoNames` + `EmitUnpopulated`, so the
 * wire is snake_case with nulls spelled out.
 */

/**
 * Why nothing can run on a host, in the three ways it can be true.
 *
 * A single "can I dispatch here" boolean collapses three problems that need three
 * different people: a machine that was never onboarded, one whose agent is stopped,
 * and one that is up with a broken driver. `registered` is what separates the first
 * from the other two.
 */
export interface OmInventoryExecutor {
  registered: boolean;
  reachable: boolean;
  driver_healthy: boolean;
  /** The backend's own reason, when the driver is the problem. */
  detail?: string | null;
}

/**
 * A mongod running on a host that PMM has no service for.
 *
 * Arbiters, mostly: PMM registers no service for one, so nothing else in OM reports
 * the process at all. Also what stands in the way of installing anything over a port
 * already in use.
 */
export interface OmUnregisteredMongod {
  port?: number | null;
  config_path?: string | null;
  argv?: string | null;
  program?: string | null;
  pid?: number | null;
}

/**
 * How current a row is, and how it is failing.
 *
 * On hosts and services alike, because the estate is upserted rather than rebuilt: a
 * row keeps what it last reported, so every attribute is only meaningful beside the
 * time it was collected. `failing_since` is the *first* failure after the last
 * success, which is what makes "failing for three days" expressible rather than only
 * "failed a minute ago".
 */
export interface OmInventoryFreshness {
  first_seen_at?: string | null;
  last_attempt_at?: string | null;
  /** Null means it has never answered, which is not the same as answering nothing. */
  last_success_at?: string | null;
  /** Null means it is not failing. */
  failing_since?: string | null;
  consecutive_failures: number;
  last_error?: string | null;
}

/**
 * One MongoDB service OM has probed, or tried to.
 *
 * The named fields are the ones a table sorts by; `observed` carries the whole
 * document besides. The split is deliberate and is why a new probe attribute reaches
 * this page without a proto change: it appears in `observed` the day it is collected,
 * and is promoted to a field of its own only when something wants to sort by it.
 */
export interface OmInventoryService {
  service_id: string;
  node_id: string;
  name?: string | null;
  port?: number | null;
  role?: string | null;
  /**
   * What the package database on the host says.
   *
   * Not the same as the snapshot's `version`, which is what the running process
   * reports over the wire. They disagree exactly when a package has been upgraded and
   * the process not restarted, which is a state OM exists to find - so the two are
   * never merged into one column.
   */
  installed_version?: string | null;
  running_version?: string | null;
  config_path?: string | null;
  argv?: string | null;
  probe_status?: string | null;
  server_running?: boolean | null;
  uptime_seconds?: number | null;
  replication_set?: string | null;
  observed: Record<string, unknown>;
  freshness: OmInventoryFreshness;
}

/**
 * One host OM keeps a row for, whether or not a database runs on it.
 *
 * A host with no service is the point rather than an edge case: it is where a database
 * can be installed, and it has no service to be discovered through.
 */
export interface OmInventoryHost {
  node_id: string;
  name: string;
  address?: string | null;
  /** The executor client serving it, or null when none matched. */
  executor_host?: string | null;
  os?: string | null;
  kernel?: string | null;
  executor: OmInventoryExecutor;
  unregistered_mongods: OmUnregisteredMongod[];
  observed: Record<string, unknown>;
  freshness: OmInventoryFreshness;
  /** The services on it. Empty is a meaningful answer, not a gap. */
  services: OmInventoryService[];
}

/** Whether a host can fetch packages, and why not when it cannot. */
export interface OmRepoReachability {
  url?: string | null;
  reachable: boolean;
  status_code?: number | null;
  latency_ms?: number | null;
  /** Null is "direct", stated rather than omitted: a failure needs the context. */
  proxy?: string | null;
  error?: string | null;
}

/** The three answers to "is there a database on this host". */
export type OmHostDatabaseState =
  | 'has_service'
  | 'unregistered_only'
  | 'installable';

/**
 * One row of the Hosts table: a host with what the page derives from it.
 *
 * The derived fields exist because the page's headline question - which hosts have no
 * database - cannot be answered from `services.length` alone. PMM's inventory cannot
 * tell a bare pmm-client host from an arbiter, so a host with no service may still be
 * running a mongod that PMM cannot authenticate against.
 */
export interface OmHostRow extends OmInventoryHost {
  database_state: OmHostDatabaseState;
  service_count: number;
  repo: OmRepoReachability | null;
}

/**
 * One row of the Services table: the snapshot's view joined to the estate's.
 *
 * The join is free, which is the payoff of keying the estate on PMM's own service id:
 * there is no matching function and nothing to get wrong. `inventory` is null for a
 * service OM has no row for yet - a service PMM registered since the last sweep.
 */
export interface OmServiceInventoryRow extends OmServiceRow {
  inventory: OmInventoryService | null;
}

/** A refresh accepted by the app, from `POST /v1/om/inventory/runs`. */
export interface OmInventoryRunAccepted {
  run_id: string;
  /** Always `RUN_STATUS_RUNNING`: the refresh is accepted, not finished. */
  status: OmTopologyRunStatus;
  start_time?: string | null;
  /** The hosts it will cover. Empty means the whole estate. */
  scope: string[];
}

/**
 * What one refresh reached.
 *
 * Hosts and services, because a refresh attempts both. Counting only services makes a
 * host-only refresh read as "0 of 0", which is what a run that did nothing also looks
 * like - on the one kind of host OM most exists to describe.
 *
 * The `total`/`probeable` pair and the `resolved`/`answered` pair say the same kind of
 * thing one level apart: the first of each is what enumeration found, the second is
 * what could actually be reached. A single "failed" count would hide the difference
 * between a broken mapping and broken executors.
 */
export interface OmInventoryRunCounts {
  total_services: number;
  resolved_services: number;
  orphaned_services: number;
  answered_services: number;
  total_hosts: number;
  probeable_hosts: number;
  answered_hosts: number;
}

/** One refresh of the estate. */
export interface OmInventoryRun {
  run_id: string;
  status: OmTopologyRunStatus;
  start_time: string;
  /** Null while it is still going. */
  end_time?: string | null;
  counts: OmInventoryRunCounts;
  /**
   * The hosts it was limited to. Empty means the whole estate.
   *
   * Without it a scoped refresh reads as a full sweep that found one host, which
   * looks like a catastrophic failure rather than the single-host refresh it was.
   */
  scope: string[];
  error?: string | null;
}

/**
 * One entity a refresh attempted, and what came of it.
 *
 * Outcomes, not observations. What the probe found lives on the estate, upserted and
 * current; a receipt carrying the attributes too would be a second copy that goes
 * stale on the next refresh.
 */
export interface OmInventoryRunEntityService {
  service_id?: string | null;
  service_name?: string | null;
  answered: boolean;
  error?: string | null;
}

/**
 * One host a refresh attempted, and what came of it.
 *
 * Host-oriented, because a refresh attempts hosts. A flat service list - which this
 * was - cannot show a machine carrying a PMM client and no database, however many
 * times it is probed, and that machine is the case OM most exists to describe.
 *
 * One dispatch covers every service on a host, so the host owns the timing and the
 * failure; its services carry only what is theirs.
 */
export interface OmInventoryRunEntity {
  node_id: string;
  host_name?: string | null;
  executor_host?: string | null;
  /** How the host was matched to an executor client, or that it was not. */
  resolution: OmExecutorResolution;
  answered: boolean;
  duration_seconds?: number | null;
  error?: string | null;
  /** The services on it. Empty is a meaningful answer, not a gap. */
  services: OmInventoryRunEntityService[];
}

/** One refresh with the rows behind its counters, from `GET /runs/{id}`. */
export interface OmInventoryRunDetail {
  run: OmInventoryRun;
  entities: OmInventoryRunEntity[];
}

/** One configuration field, and where its value came from. */
export interface OmInventorySetting {
  key: string;
  value: unknown;
  default_value: unknown;
  type: string;
  reload: OmSettingReload;
  has_override: boolean;
  is_advanced: boolean;
  description?: string | null;
}
