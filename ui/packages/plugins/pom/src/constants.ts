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
  PomHostDatabaseState,
  PomProcessRole,
  PomRunStatus,
  PomServiceStatus,
  PomUnavailableReason,
} from './types';

/** POM's own routes, relative to wherever the shell mounts the plugin. */
export const POM_ROUTE_OVERVIEW = '';
/**
 * Renamed from `topology`: the page now joins PMM's snapshot to POM's estate, and
 * "Services" is what it lists either way. Free while nothing has shipped.
 */
export const POM_ROUTE_SERVICES = 'services';
export const POM_ROUTE_HOSTS = 'hosts';
/** Renamed from `runs`, so the route matches the nav label. */
export const POM_ROUTE_DISCOVERY = 'discovery';

export const SERVICE_STATUS_LABEL: Record<PomServiceStatus, string> = {
  UP: 'Up',
  DOWN: 'Down',
};

export const SERVICE_STATUS_COLOR: Record<
  PomServiceStatus,
  ChipProps['color']
> = {
  UP: 'success',
  DOWN: 'error',
};

/** Display names for the process roles, which the raw values abbreviate heavily. */
export const PROCESS_ROLE_LABEL: Record<PomProcessRole, string> = {
  mongod: 'mongod',
  mongos: 'Router',
  configsvr: 'Config server',
  shardsvr: 'Shard',
};

/**
 * How each of the three database states reads on the Hosts page.
 *
 * The middle one is the whole reason there are three. A host with no *registered*
 * service may still be running a mongod - PMM cannot authenticate against an arbiter,
 * so it registers no service for one - and calling that host empty would invite
 * someone to install a database over a port already in use.
 */
export const HOST_DATABASE_STATE_LABEL: Record<PomHostDatabaseState, string> = {
  has_service: 'Monitored',
  unregistered_only: 'Unregistered mongod',
  installable: 'No database',
};

export const HOST_DATABASE_STATE_COLOR: Record<
  PomHostDatabaseState,
  ChipProps['color']
> = {
  has_service: 'success',
  unregistered_only: 'warning',
  installable: 'default',
};

export const HOST_DATABASE_STATE_PHRASE: Record<PomHostDatabaseState, string> =
  {
    has_service: 'PMM has at least one registered MongoDB service on this host',
    unregistered_only:
      'No service PMM knows about, but the probe found a mongod running - an arbiter, most likely, since PMM cannot authenticate against one. Not an empty host.',
    installable:
      'No registered service and no mongod found. This is where a database can be installed.',
  };

export const UNAVAILABLE_PHRASE: Record<PomUnavailableReason, string> = {
  service_not_observed:
    'Not observed — the service has no live executor, or it did not answer this run',
  metric_not_collected: 'Not collected — no collector produces this metric yet',
  no_version_catalog:
    'No version catalog yet — PMM has no PSMDB release data to compare against',
  not_applicable:
    'Not applicable — a standalone or a router has no replica-set oplog, and a single-member set has no peer to lag behind',
  // The two the estate adds. Both mean "the probe has no answer", and they are kept
  // apart because they need different things done about them: one is a service POM
  // has never been asked about, the other is a host the probe cannot reach.
  not_in_inventory:
    'Not in the inventory yet — POM has no row for this service, so no probe has ever been dispatched for it. The next sweep will create one.',
  probe_never_succeeded:
    'Never collected — POM has a row for this service but no probe has ever succeeded against it. Its host may have no executor, or every attempt may have failed.',
};

/** Fallback for a reason code the frontend has not been taught. */
export const UNAVAILABLE_FALLBACK = 'Not available';

export const RUN_STATUS_LABEL: Record<PomRunStatus, string> = {
  running: 'Running',
  success: 'Success',
  partial: 'Partial',
  failed: 'Failed',
};

export const RUN_STATUS_COLOR: Record<PomRunStatus, ChipProps['color']> = {
  running: 'info',
  success: 'success',
  partial: 'warning',
  failed: 'error',
};
