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

import { PROCESS_ROLE_LABEL } from '../constants';
import { pluralize } from '../format';

export const Messages = {
  fleet: 'MongoDB Fleet',
  load: {
    error: 'Could not load the cluster',
  },
  notFound: {
    title: 'Cluster not found',
    body: 'No monitored MongoDB cluster has this id. It may have been removed, or its services relabelled so it is now part of another cluster.',
    back: 'Back to Fleet',
  },
  switcher: {
    open: 'Switch cluster',
    search: 'Find a cluster',
    empty: 'No clusters match',
  },
  header: {
    members: (count: number) => `${count} ${pluralize(count, 'member')}`,
    shards: (count: number) => `${count} ${pluralize(count, 'shard')}`,
    replicaSet: 'Replica set name',
    environment: 'Environment',
    dashboard: 'Monitoring dashboard',
    clusterActions: 'Cluster actions',
  },
  connectionString: {
    label: 'Connection string',
    copy: 'Copy',
  },
  members: {
    title: 'Members',
    reachable: (up: number, total: number) => `${up} / ${total} reachable`,
    allHealthy: 'all healthy',
    down: (count: number) => `${count} down`,
    empty: 'This cluster has no services in the current snapshot.',
    columns: {
      hostname: 'Hostname',
      role: 'Role',
      health: 'Health',
      oplogWindow: 'Oplog window',
      replicationLag: 'Repl. lag',
    },
    groups: {
      routers: 'Routers',
      configServers: 'Config servers',
      ungrouped: 'Other members',
    },
    groupCount: (count: number) => `${count} ${pluralize(count, 'member')}`,
    actions: (host: string) => `Actions for ${host}`,
    openDashboards: 'Open in PMM dashboards',
  },
  roles: {
    primary: 'Primary',
    secondary: 'Secondary',
    arbiter: PROCESS_ROLE_LABEL.PROCESS_ROLE_ARBITER,
    router: PROCESS_ROLE_LABEL.PROCESS_ROLE_MONGOS,
  },
};
