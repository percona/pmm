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

export interface DetailAction {
  id: string;
  label: string;
  /** No API runs these yet; the menus show only the available ones. */
  available: boolean;
  /** For member actions: the roles it applies to, all when omitted. */
  roles?: MemberRole[];
}

const CLUSTER_ACTIONS: DetailAction[] = [
  { id: 'backup', label: 'Run backup', available: false },
  { id: 'restore', label: 'Restore from backup', available: false },
  { id: 'configure', label: 'Edit configuration', available: false },
  { id: 'update', label: 'Update MongoDB', available: false },
];

const MEMBER_ACTIONS: DetailAction[] = [
  { id: 'start', label: 'Start', available: false },
  { id: 'stop', label: 'Stop', available: false },
  { id: 'restart', label: 'Restart', available: false },
  { id: 'stepDown', label: 'Step down', roles: ['primary'], available: false },
];

export const AVAILABLE_CLUSTER_ACTIONS = CLUSTER_ACTIONS.filter(
  (action) => action.available
);

export const availableMemberActions = (role: MemberRole | null) =>
  MEMBER_ACTIONS.filter(
    (action) =>
      action.available &&
      (!action.roles || (role !== null && action.roles.includes(role)))
  );
