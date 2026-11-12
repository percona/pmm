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

import { pluralize } from '../format';

export const Messages = {
  unnamedEnvironment: 'Unassigned environment',
  groupBy: {
    env_name: 'Environment',
    health: 'Health',
    none: 'None',
  },
  columns: {
    environment: 'Environment',
    cluster: 'Cluster',
    type: 'Type',
    health: 'Health',
    members: 'Members',
    services: 'Services',
    process: 'Process',
    versions: 'Versions',
    maxReplicationLag: 'Max repl. lag',
    minOplogWindow: 'Min oplog window',
    lastSeen: 'Last seen',
    maxCpu: 'Max CPU',
    minConnectionsFree: 'Min conn. free',
  },
  columnTooltips: {
    members: 'Members up, out of all the members in the cluster',
    lastSeen: 'When the least recently observed member was last observed',
    minConnectionsFree:
      'The lowest percentage of free connections among the members',
  },
  versions: {
    mixed: 'Mixed',
    count: (count: number) => `${count} versions`,
  },
  search: 'Search clusters',
  noMatch: 'No cluster matches the search and filters.',
  filters: {
    health: 'Health',
    type: 'Type',
    environment: 'Environment',
    clear: 'Clear filters',
    matchCount: (shown: number, total: number) =>
      `${shown} of ${total} ${pluralize(total, 'cluster')}`,
  },
  groupToggle: (expanded: boolean, label: string) =>
    `${expanded ? 'Collapse' : 'Expand'} ${label}`,
  members: {
    empty: 'This cluster has no services in the current snapshot.',
    columns: {
      service: 'Service',
      node: 'Node',
      status: 'Status',
      memberState: 'Member state',
      process: 'Process',
      version: 'Version',
      replicationLag: 'Repl. lag',
      oplogWindow: 'Oplog window',
    },
  },
  load: {
    error: 'Could not load the fleet',
    scanError: 'Could not read scan results',
    scanErrorDetail:
      "Until Operations can read them again, the Services tab shows no scan status, or the status from its last successful read. The clusters and their health come from PMM's own data and are unaffected.",
  },
  empty: {
    title: 'No MongoDB clusters yet',
    intro:
      "This page shows every MongoDB cluster PMM monitors, and the health of each one's members.",
    noServices:
      'It is empty because PMM has no MongoDB services registered yet - add one, and it appears here on the next refresh.',
    docLink: 'Read about Operations for MongoDB',
  },
};
