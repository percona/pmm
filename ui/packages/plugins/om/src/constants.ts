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

import type { ChipProps } from '@mui/material/Chip';
import type {
  OmBootstrapRunStatus,
  OmBootstrapStepStatus,
  OmClusterHealth,
  OmHostDatabaseState,
  OmProcessRole,
  OmTopologyRunStatus,
  OmServiceStatus,
  OmUnavailableReason,
} from './types';

/**
 * Operations' own routes, relative to wherever the shell mounts the plugin.
 *
 * One route per *job*, with tabs inside for the flavours of it -- the rule the whole
 * structure follows (see om-design-review/structure-and-glossary-proposal.md). So the
 * cluster and service readings of one snapshot are two tabs on {@link OM_ROUTE_FLEET}
 * rather than two routes, and a scan's history sits beside an install's on
 * {@link OM_ROUTE_AUTOMATIONS} rather than on a page called Inventory, which collided
 * with PMM's own.
 */
export const OM_ROUTE_FLEET = '';
/** The machines, and the only page carrying actions that change one. */
export const OM_ROUTE_NODES = 'nodes';
/** The install wizard, given a node selection made on {@link OM_ROUTE_NODES}. */
export const OM_ROUTE_INSTALL = 'nodes/install';
export const OM_ROUTE_AUTOMATIONS = 'automations';
/** This app's own configuration. PMM's on/off switch stays in PMM's settings. */
export const OM_ROUTE_SETTINGS = 'settings';

/**
 * The routes that existed before the structure changed, kept only to redirect.
 *
 * Nothing has shipped, so these owe nobody a URL -- but the feature build is already
 * out with people clicking round it, and a dead link in a review is a bug report that
 * costs more to answer than the three lines it takes to avoid.
 */
export const OM_LEGACY_REDIRECTS: Record<string, string> = {
  services: `${OM_ROUTE_FLEET}?tab=services`,
  hosts: OM_ROUTE_NODES,
  'hosts/bootstrap': OM_ROUTE_INSTALL,
  inventory: `${OM_ROUTE_AUTOMATIONS}?tab=scans`,
};

export const SERVICE_STATUS_LABEL: Record<OmServiceStatus, string> = {
  SERVICE_STATUS_UNSPECIFIED: 'Unknown',
  SERVICE_STATUS_UP: 'Up',
  SERVICE_STATUS_DOWN: 'Down',
};

export const SERVICE_STATUS_COLOR: Record<OmServiceStatus, ChipProps['color']> =
  {
    SERVICE_STATUS_UNSPECIFIED: 'default',
    SERVICE_STATUS_UP: 'success',
    SERVICE_STATUS_DOWN: 'error',
  };

/** Display names for the process roles, which the raw values abbreviate heavily. */
export const PROCESS_ROLE_LABEL: Record<OmProcessRole, string> = {
  PROCESS_ROLE_UNSPECIFIED: 'Unknown',
  PROCESS_ROLE_MONGOD: 'mongod',
  PROCESS_ROLE_MONGOS: 'Router',
  PROCESS_ROLE_CONFIGSVR: 'Config server',
  PROCESS_ROLE_SHARDSVR: 'Shard',
};

/** How each cluster state reads on Overview: the word, then the colour behind it. */
export const CLUSTER_HEALTH_LABEL: Record<OmClusterHealth, string> = {
  healthy: 'Healthy',
  degraded: 'Degraded',
  down: 'Down',
  unknown: 'Unknown',
};

export const CLUSTER_HEALTH_COLOR: Record<OmClusterHealth, ChipProps['color']> =
  {
    healthy: 'success',
    degraded: 'warning',
    down: 'error',
    unknown: 'default',
  };

/**
 * How each of the three database states reads on the Hosts page.
 *
 * The middle one is the whole reason there are three. A host with no *registered*
 * service may still be running a mongod - PMM cannot authenticate against an arbiter,
 * so it registers no service for one - and calling that host empty would invite
 * someone to install a database over a port already in use.
 */
export const HOST_DATABASE_STATE_LABEL: Record<OmHostDatabaseState, string> = {
  has_service: 'Monitored',
  unregistered_only: 'Unregistered mongod',
  installable: 'No database',
};

export const HOST_DATABASE_STATE_COLOR: Record<
  OmHostDatabaseState,
  ChipProps['color']
> = {
  has_service: 'success',
  unregistered_only: 'warning',
  installable: 'default',
};

export const HOST_DATABASE_STATE_PHRASE: Record<OmHostDatabaseState, string> = {
  has_service: 'PMM has at least one registered MongoDB service on this node',
  unregistered_only:
    'No service PMM knows about, but a scan found a mongod running - an arbiter, most likely, since PMM cannot authenticate against one. Not an empty node.',
  installable:
    'No registered service and no mongod found. This is where a database can be installed.',
};

export const UNAVAILABLE_PHRASE: Record<OmUnavailableReason, string> = {
  service_not_observed:
    'Not observed — the service has no automation agent, or it did not answer this run',
  metric_not_collected: 'Not collected — no collector produces this metric yet',
  no_version_catalog:
    'No version catalog yet — PMM has no PSMDB release data to compare against',
  not_applicable:
    'Not applicable — a standalone or a router has no replica-set oplog, and a single-member set has no peer to lag behind',
  // The two the fleet view adds. Both mean "no scan has an answer", and they are kept
  // apart because they need different things done about them: one is a service
  // Operations has never been asked about, the other is a node a scan cannot reach.
  not_in_inventory:
    'Not scanned yet — Operations has no row for this service, so no scan has ever been sent for it. The next scan will create one.',
  probe_never_succeeded:
    'Never collected — Operations has a row for this service but no scan has ever succeeded against it. Its node may have no automation agent, or every attempt may have failed.',
  // Distinct from not_in_inventory on purpose. That one is a statement about the
  // fleet; this one is an admission that the fleet could not be read, and the two
  // must not look the same -- reporting "not scanned yet" for every row because one
  // request failed is a confident wrong answer.
  inventory_unavailable:
    'Scan results unavailable — Operations could not read them, so nothing is known about this service either way. The monitoring columns are unaffected.',
  // The third of the same family, and it exists for the same reason the second does.
  // The fleet document answers in a tenth of a second while the scan results are a
  // second request that may still be in flight; reporting "not scanned yet" during
  // that window states a fact about them before they have answered.
  inventory_pending:
    'Loading scan results — Operations has not answered yet, so whether it has a row for this service is not known. The monitoring columns come from PMM and are already current.',
};

/** Fallback for a reason code the frontend has not been taught. */
export const UNAVAILABLE_FALLBACK = 'Not available';

export const RUN_STATUS_LABEL: Record<OmTopologyRunStatus, string> = {
  RUN_STATUS_UNSPECIFIED: 'Unknown',
  RUN_STATUS_RUNNING: 'Running',
  RUN_STATUS_SUCCESS: 'Success',
  RUN_STATUS_PARTIAL: 'Partial',
  RUN_STATUS_FAILED: 'Failed',
  RUN_STATUS_SKIPPED: 'Skipped',
};

export const RUN_STATUS_COLOR: Record<OmTopologyRunStatus, ChipProps['color']> =
  {
    // A status this build has not been taught. Rendered rather than hidden: an unknown
    // value is a real answer from a newer server, not a missing one.
    RUN_STATUS_UNSPECIFIED: 'default',
    RUN_STATUS_RUNNING: 'info',
    RUN_STATUS_SUCCESS: 'success',
    RUN_STATUS_PARTIAL: 'warning',
    RUN_STATUS_FAILED: 'error',
    // Neither good nor bad: nothing happened, on purpose. Colouring it as a failure
    // would put a red row in the history every time the schedule met a manual refresh.
    RUN_STATUS_SKIPPED: 'default',
  };

/**
 * Human labels for OM's configuration fields.
 *
 * The raw keys are what the app calls them and what the API takes; these are what a
 * reader should see. Anything not named here falls back to its key, so a setting added
 * in PMM Extensions still renders rather than disappearing from the form.
 */
export const SETTING_LABEL: Record<string, string> = {
  // Not "ENABLED". The raw key rendered in capitals, and said nothing about what it
  // switches -- a reader could not tell it from PMM's own switch for the whole
  // feature, which is a different control on a different page.
  ENABLED: 'Automatic node scans',
  SCHEDULE__every: 'Scan every',
  SCHEDULE__period: 'Period',
  PROBE_DATABASE: 'Connect to MongoDB',
  REPO_URL: 'Repository check URL',
  REPO_TIMEOUT: 'Repository timeout',
  CONNECT_TIMEOUT: 'MongoDB connect timeout',
  TASK_TIMEOUT: 'Scan job timeout',
  POLL_INTERVAL: 'Job poll interval',
  MAX_CONCURRENT_PROBES: 'Concurrent scans',
  RUN_RETENTION: 'Scans kept',
  STALE_RUN_AFTER: 'Consider a scan stuck after',
};

/**
 * The unit a number is in, spelled out in the field's label.
 *
 * `STALE_RUN_AFTER` is the one that most needs it: it is a `timedelta` that arrives
 * over the wire as whole seconds, so "1800" beside a field called "wedged after" is
 * ambiguous in a way that matters.
 */
export const SETTING_UNIT: Record<string, string> = {
  REPO_TIMEOUT: 'seconds',
  CONNECT_TIMEOUT: 'seconds',
  TASK_TIMEOUT: 'seconds',
  POLL_INTERVAL: 'seconds',
  STALE_RUN_AFTER: 'seconds',
  MAX_CONCURRENT_PROBES: 'jobs at once',
  RUN_RETENTION: 'rows',
};

/** What each field costs or protects, for the reader deciding whether to touch it. */
export const SETTING_HELP: Record<string, string> = {
  ENABLED:
    'Scan every node on a schedule. Off stops the schedule only - you can still scan from the Nodes page, and nothing already collected is removed.',
  SCHEDULE__every:
    'How often every node is scanned. Each scan sends a job to every node.',
  PROBE_DATABASE:
    'On also asks each mongod for its replica-set state and running version, using the monitoring credentials PMM already holds for that service. Off collects process and OS facts only, which needs no credentials - and still yields the installed version.',
  REPO_URL:
    'The file each node fetches to prove it can install packages. Point it at a mirror on an air-gapped network, or every node reports as failing.',
  REPO_TIMEOUT:
    'Short on purpose: a repository slower than this is not usable by a package manager either.',
  CONNECT_TIMEOUT: 'Per-target connect and server-selection timeout.',
  TASK_TIMEOUT: 'How long to wait for one scan job before giving up on it.',
  POLL_INTERVAL: 'How often a running job is checked for completion.',
  MAX_CONCURRENT_PROBES:
    'Ceiling on scan jobs at once. Each scan runs as a job on its node, so this caps the load on PMM and on the nodes.',
  RUN_RETENTION: 'How many scan rows to keep before the oldest are removed.',
  STALE_RUN_AFTER:
    'How long a scan may stay running before its worker is presumed gone. Must exceed the slowest legitimate scan.',
};

/** One bootstrap step's status, as a short label. */
export const BOOTSTRAP_STEP_LABEL: Record<OmBootstrapStepStatus, string> = {
  pending: 'Pending',
  running: 'Running',
  succeeded: 'Succeeded',
  failed: 'Failed',
  skipped: 'Skipped',
};

/** Same palette convention as {@link RUN_STATUS_COLOR}. */
export const BOOTSTRAP_STEP_COLOR: Record<
  OmBootstrapStepStatus,
  ChipProps['color']
> = {
  pending: 'default',
  running: 'info',
  succeeded: 'success',
  failed: 'error',
  skipped: 'default',
};

/** A bootstrap run's overall status, as a short label. */
export const BOOTSTRAP_RUN_LABEL: Record<OmBootstrapRunStatus, string> = {
  running: 'Running',
  succeeded: 'Succeeded',
  failed: 'Failed',
  rolled_back: 'Rolled back',
};

/** Same palette convention as {@link RUN_STATUS_COLOR}. */
export const BOOTSTRAP_RUN_COLOR: Record<
  OmBootstrapRunStatus,
  ChipProps['color']
> = {
  running: 'info',
  succeeded: 'success',
  failed: 'error',
  rolled_back: 'warning',
};

/**
 * Which Settings tab a field belongs on.
 *
 * Grouped by who the field is for, not by what it configures. **General** is the two
 * switches a DBA reasons about. **Scanning** is the handful that change what they
 * then see -- a wrong repository URL makes every node report as failing, with nothing
 * on the Nodes page pointing back here. Everything else is the job runner's own
 * plumbing: real settings, but ones a DBA has no basis for choosing, so they sit
 * behind a tab that says so rather than beside the two that matter.
 *
 * Anything a future release adds lands in Advanced by default, which is the safe end
 * to be wrong at: an unknown setting shown late is a worse outcome than one shown
 * beside a warning.
 */
export const SETTING_GROUP: Record<string, 'general' | 'scanning'> = {
  ENABLED: 'general',
  SCHEDULE__every: 'general',
  SCHEDULE__period: 'general',
  PROBE_DATABASE: 'general',
  REPO_URL: 'scanning',
  REPO_TIMEOUT: 'scanning',
  CONNECT_TIMEOUT: 'scanning',
};

/** Why a fleet tab is empty before pmm-managed's first collection. */
export const FLEET_NOT_COLLECTED =
  "Operations has not read PMM's MongoDB services yet. It does within a minute, or press Refresh to do it now.";

/**
 * Where a reader goes to fix a node with no automation agent.
 *
 * The product documentation rather than a deep link into settings: both halves
 * of the fix (the feature flag and the public address) are server-side and need
 * a restart, and the client-side requirements are listed there too.
 */
export const NOMAD_DOC_URL =
  'https://docs.percona.com/percona-monitoring-and-management/3/reference/nomad.html';

/**
 * The pmm-agent version that first carries an automation agent.
 *
 * Mirrors `NomadAgentSupportVersion` in `version/features.go`; below it the server
 * creates no agent at all, silently.
 */
export const PMM_AGENT_AUTOMATION_MIN_VERSION = '3.2.0';
