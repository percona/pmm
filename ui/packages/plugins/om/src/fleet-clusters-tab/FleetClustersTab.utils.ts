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

import type { MRT_FilterFn } from 'material-react-table';
import type { FilterOption } from '../components/MultiSelectFilter';
import { CLUSTER_TYPE_LABEL, PROCESS_ROLE_LABEL } from '../constants';
import { compareVersions } from '../topology';
import type { OmClusterRow, OmProcessRole } from '../types';
import { Messages } from './FleetClustersTab.messages';
import type { GroupingColumn } from './FleetClustersTab.types';

export const environmentLabel = (row: OmClusterRow) =>
  row.env_name ?? Messages.unnamedEnvironment;

export const typeLabel = (row: OmClusterRow) => CLUSTER_TYPE_LABEL[row.type];

/** `3 mongod · 1 Router`, in the order the API lists the roles. */
export const describeRoles = (
  roles: Partial<Record<OmProcessRole, number>>
): string =>
  Object.entries(roles)
    .map(
      ([role, count]) =>
        `${count} ${PROCESS_ROLE_LABEL[role as OmProcessRole] ?? role}`
    )
    .join(' · ');

/**
 * A cluster's value in each grouping column.
 */
export const GROUP_VALUE: Record<
  GroupingColumn,
  (row: OmClusterRow) => string
> = {
  env_name: (row) => row.env_name ?? '',
  health: (row) => row.health,
};

export const groupLabel = (row: OmClusterRow, grouping: GroupingColumn) =>
  grouping === 'health' ? row.health : environmentLabel(row);

/** A filter's picked values; a hand-written URL can carry one bare value. */
export const pickedValues = (value: unknown): string[] =>
  Array.isArray(value)
    ? value.map(String)
    : value == null || value === ''
      ? []
      : [String(value)];

/** Exact match on any picked value; MRT's multi-select default matches `prod` in `preprod`. */
export const anyOf =
  (
    valueOf: (row: OmClusterRow) => string | undefined
  ): MRT_FilterFn<OmClusterRow> =>
  (row, _columnId, picked: unknown) => {
    const value = valueOf(row.original);
    return value !== undefined && pickedValues(picked).includes(value);
  };

/** Element by element, oldest first, so a cluster still on an old build leads. */
export const compareVersionLists = (a: string[], b: string[]) => {
  for (let i = 0; i < Math.min(a.length, b.length); i++) {
    const order = compareVersions(a[i], b[i]);
    if (order !== 0) {
      return order;
    }
  }
  return a.length - b.length;
};

/** Lists a picked value the options lack, such as a stale link's, so it can be unpicked. */
export const withPicked = (options: FilterOption[], picked: string[]) => [
  ...options,
  ...picked
    .filter((value) => !options.some((option) => option.value === value))
    .map((value) => ({ value, label: value })),
];
