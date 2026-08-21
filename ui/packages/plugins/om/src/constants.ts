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
  OmHostDatabaseState,
  OmProcessRole,
  OmTopologyRunStatus,
  OmServiceStatus,
  OmUnavailableReason,
} from './types';

/** OM's own routes, relative to wherever the shell mounts the plugin. */
export const OM_ROUTE_OVERVIEW = '';
/**
 * Renamed from `topology`: the page now joins PMM's snapshot to OM's estate, and
 * "Services" is what it lists either way. Free while nothing has shipped.
 */
export const OM_ROUTE_SERVICES = 'services';
export const OM_ROUTE_HOSTS = 'hosts';
/** Renamed from `runs`, so the route matches the nav label. */
export const OM_ROUTE_INVENTORY = 'inventory';

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
  has_service: 'PMM has at least one registered MongoDB service on this host',
  unregistered_only:
    'No service PMM knows about, but the probe found a mongod running - an arbiter, most likely, since PMM cannot authenticate against one. Not an empty host.',
  installable:
    'No registered service and no mongod found. This is where a database can be installed.',
};

export const UNAVAILABLE_PHRASE: Record<OmUnavailableReason, string> = {
  service_not_observed:
    'Not observed — the service has no live executor, or it did not answer this run',
  metric_not_collected: 'Not collected — no collector produces this metric yet',
  no_version_catalog:
    'No version catalog yet — PMM has no PSMDB release data to compare against',
  not_applicable:
    'Not applicable — a standalone or a router has no replica-set oplog, and a single-member set has no peer to lag behind',
  // The two the estate adds. Both mean "the probe has no answer", and they are kept
  // apart because they need different things done about them: one is a service OM
  // has never been asked about, the other is a host the probe cannot reach.
  not_in_inventory:
    'Not in the inventory yet — OM has no row for this service, so no probe has ever been dispatched for it. The next sweep will create one.',
  probe_never_succeeded:
    'Never collected — OM has a row for this service but no probe has ever succeeded against it. Its host may have no executor, or every attempt may have failed.',
  // Distinct from not_in_inventory on purpose. That one is a statement about the
  // estate; this one is an admission that the estate could not be read, and the two
  // must not look the same -- reporting "not in the inventory" for every row because
  // one request failed is a confident wrong answer.
  inventory_unavailable:
    'Inventory unavailable — OM could not read the estate, so nothing is known about this service either way. The topology columns are unaffected.',
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
 * in SEP still renders rather than disappearing from the form.
 */
export const SETTING_LABEL: Record<string, string> = {
  SCHEDULE__every: 'Sweep every',
  SCHEDULE__period: 'Period',
  PROBE_DATABASE: 'Connect to MongoDB',
  REPO_URL: 'Repository check URL',
  REPO_TIMEOUT: 'Repository timeout',
  CONNECT_TIMEOUT: 'MongoDB connect timeout',
  TASK_TIMEOUT: 'Probe job timeout',
  POLL_INTERVAL: 'Job poll interval',
  MAX_CONCURRENT_PROBES: 'Concurrent probes',
  RUN_RETENTION: 'Refreshes kept',
  STALE_RUN_AFTER: 'Consider a refresh wedged after',
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
  MAX_CONCURRENT_PROBES: 'jobs in flight',
  RUN_RETENTION: 'rows',
};

/** What each field costs or protects, for the reader deciding whether to touch it. */
export const SETTING_HELP: Record<string, string> = {
  SCHEDULE__every:
    'How often the whole estate is swept. Each sweep dispatches a job to every host.',
  PROBE_DATABASE:
    'Off collects process and OS facts only, which needs no credentials - and still yields the installed version.',
  REPO_URL:
    'The file each host fetches to prove it can install packages. Point it at a mirror on an air-gapped estate, or every host reports as broken.',
  REPO_TIMEOUT:
    'Short on purpose: a repository slower than this is not usable by a package manager either.',
  CONNECT_TIMEOUT: 'Per-target connect and server-selection timeout.',
  TASK_TIMEOUT:
    'How long to wait for one dispatched probe job before giving up on it.',
  POLL_INTERVAL: 'How often a dispatched job is checked for completion.',
  MAX_CONCURRENT_PROBES:
    'Ceiling on probe jobs at once. Every dispatch is a Nomad job, so this is cluster capacity.',
  RUN_RETENTION: 'How many refresh rows to keep before the oldest are pruned.',
  STALE_RUN_AFTER:
    'How long a refresh may stay running before its worker is presumed gone. Must exceed the slowest legitimate sweep.',
};
