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
    health: 'Health status',
    env_name: 'Environment',
    type: 'Type',
    versions: 'Version',
    none: 'None',
  },
  groupBar: {
    groups: (count: number) => `${count} ${pluralize(count, 'group')}`,
    clusters: (shown: number, total: number) =>
      shown === total
        ? `${total} ${pluralize(total, 'cluster')}`
        : `${shown} of ${total} ${pluralize(total, 'cluster')}`,
    refreshed: (age: string) => `Refreshed ${age}`,
  },
  summary: {
    clusters: (count: number) => `${count} ${pluralize(count, 'cluster')}`,
    health: (count: number, label: string) => `${count} ${label.toLowerCase()}`,
    members: (count: number) => `${count} ${pluralize(count, 'member')}`,
  },
  healthCard: {
    title: 'Fleet health',
    unhealthy: (total: number) => `${pluralize(total, 'cluster')} unhealthy`,
    showUnhealthy: 'Show unhealthy clusters',
  },
  install: 'Install new cluster',
  installTooltip:
    'Pick the nodes to install on, on the Nodes page, then install from there.',
  columns: {
    environment: 'Environment',
    cluster: 'Cluster name',
    type: 'Type',
    health: 'Health',
    members: 'Members',
    services: 'Services',
    process: 'Process',
    versions: 'Version',
    maxReplicationLag: 'Max repl. lag',
    minOplogWindow: 'Oplog window',
    lastSeen: 'Last seen',
    maxCpu: 'Max CPU',
    minConnectionsFree: 'Min conn. free',
  },
  columnTooltips: {
    members: 'Members up, out of all the members in the cluster',
    lastSeen: 'When the least recently observed member was last observed',
    maxReplicationLag: 'The largest replication lag among the members',
    minOplogWindow: 'The smallest oplog window among the members',
    minConnectionsFree:
      'The lowest percentage of free connections among the members',
  },
  versions: {
    mixed: 'Mixed',
    mixedGroup: 'Mixed versions',
    unknownGroup: 'Unknown version',
    count: (count: number) => `${count} versions`,
  },
  unknownType: 'Unknown type',
  healthReason: {
    allDown: (total: number) =>
      total === 1 ? 'Its only member is down' : `All ${total} members down`,
    oneDown: (member: string) => `${member} down`,
    someDown: (down: number, total: number) =>
      `${down} of ${total} members down`,
    noMembers: 'No members in this snapshot',
    notReporting: (count: number, total: number) =>
      `${count} of ${total} ${pluralize(total, 'member')} not reporting`,
  },
  membersUnreported: '?',
  membersUnreportedTooltip: 'No member has reported in this snapshot',
  lastSeenStale: (limit: string) => `Not observed for over ${limit}`,
  thresholds: {
    lagAbove: (limit: string) => `Above ${limit}, the alert threshold`,
    oplogBelow: (limit: string) =>
      `Below ${limit}: a secondary down longer than this needs a full resync`,
  },
  updateAvailable: 'Update available',
  search: 'Filter clusters by name',
  clearSearch: 'Clear search',
  noMatch: 'No cluster matches the search and filters.',
  filters: {
    health: 'Health',
    type: 'Type',
    environment: 'Environment',
    version: 'Version',
    clear: 'Clear all',
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
