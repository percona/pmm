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
  PomProcessRole,
  PomRunStatus,
  PomServiceStatus,
  PomUnavailableReason,
} from './types';

/** POM's own routes, relative to wherever the shell mounts the plugin. */
export const POM_ROUTE_OVERVIEW = '';
export const POM_ROUTE_TOPOLOGY = 'topology';
export const POM_ROUTE_RUNS = 'runs';

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

export const UNAVAILABLE_PHRASE: Record<PomUnavailableReason, string> = {
  service_not_observed:
    'Not observed — the service has no live executor, or it did not answer this run',
  metric_not_collected: 'Not collected — no collector produces this metric yet',
  no_version_catalog:
    'No version catalog yet — PMM has no PSMDB release data to compare against',
  not_applicable:
    'Not applicable — a standalone or a router has no replica-set oplog, and a single-member set has no peer to lag behind',
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
