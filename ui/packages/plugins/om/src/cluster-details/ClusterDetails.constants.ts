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

import type { MemberRole } from './ClusterDetails.types';

export const CLUSTER_DASHBOARD = {
  replicaSet: '/graph/d/mongodb-replicaset-summary/mongodb-replset-summary',
  sharded: '/graph/d/mongodb-cluster-summary/mongodb-sharded-cluster-summary',
};

export const ROLE_RANK: Record<MemberRole, number> = {
  primary: 0,
  secondary: 1,
  arbiter: 2,
  router: 3,
};

export const MEMBERS_TABLE_NAME = 'om-cluster-members';

export const MEMBERS_URL_STATE = {
  paramPrefix: 'members',
  sync: { pagination: false, filters: false, globalFilter: false },
};
