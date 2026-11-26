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
import { NO_GROUPING } from '../components/GroupBySelect';
import type { FilterOption } from '../components/MultiSelectFilter';
import {
  CLUSTER_HEALTH_LABEL,
  CLUSTER_TYPE_LABEL,
  PROCESS_ROLE_LABEL,
} from '../constants';
import { clusterHealthRank, compareVersions } from '../topology';
import type { OmClusterRow, OmProcessRole } from '../types';
import {
  MIXED_VERSIONS,
  TYPE_FILTER_VALUE,
  TYPE_ORDER,
  UNKNOWN_VERSION,
  UNNAMED_ENVIRONMENT_VALUE,
} from './FleetClustersTab.constants';
import { Messages } from './FleetClustersTab.messages';
import type {
  ClusterGrouping,
  FilterColumn,
  GroupingColumn,
} from './FleetClustersTab.types';

export const environmentLabel = (row: OmClusterRow) =>
  row.env_name || Messages.unnamedEnvironment;

export const typeLabel = (row: OmClusterRow) => CLUSTER_TYPE_LABEL[row.type];

export const getEnvironmentValue = (row: OmClusterRow) =>
  row.env_name || UNNAMED_ENVIRONMENT_VALUE;

/** Environment filter values in Group by's order: by name, the unnamed one last. */
export const compareEnvironmentValues = (a: string, b: string) =>
  Number(a === UNNAMED_ENVIRONMENT_VALUE) -
    Number(b === UNNAMED_ENVIRONMENT_VALUE) || a.localeCompare(b);

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
  health: (row) => row.health,
  env_name: (row) => row.env_name ?? '',
  type: (row) => row.type,
  versions: ({ versions }) =>
    versions.length === 1
      ? versions[0]
      : versions.length
        ? MIXED_VERSIONS
        : UNKNOWN_VERSION,
};

export const groupLabel = (row: OmClusterRow, grouping: GroupingColumn) => {
  switch (grouping) {
    case 'health':
      return CLUSTER_HEALTH_LABEL[row.health];
    case 'env_name':
      return environmentLabel(row);
    case 'type':
      return typeLabel(row) ?? Messages.unknownType;
    case 'versions': {
      const value = GROUP_VALUE.versions(row);
      if (value === MIXED_VERSIONS) {
        return Messages.versions.mixedGroup;
      }
      return value === UNKNOWN_VERSION ? Messages.versions.unknownGroup : value;
    }
  }
};

const typeRank = (row: OmClusterRow) => {
  const rank = TYPE_ORDER.indexOf(row.type);
  return rank < 0 ? TYPE_ORDER.length : rank;
};

/** Mixed after every single version, unknown after that. */
const versionRank = (row: OmClusterRow) =>
  [MIXED_VERSIONS, UNKNOWN_VERSION].indexOf(GROUP_VALUE.versions(row));

const GROUP_ORDER: Record<
  GroupingColumn,
  (a: OmClusterRow, b: OmClusterRow) => number
> = {
  health: (a, b) => clusterHealthRank(a.health) - clusterHealthRank(b.health),
  env_name: (a, b) =>
    compareEnvironmentValues(getEnvironmentValue(a), getEnvironmentValue(b)),
  type: (a, b) => typeRank(a) - typeRank(b),
  // Newest first.
  versions: (a, b) =>
    versionRank(a) - versionRank(b) ||
    compareVersions(GROUP_VALUE.versions(b), GROUP_VALUE.versions(a)),
};

/**
 * The rows in the order their groups should appear. TanStack keeps groups in the order
 * their first row arrives, since the default sort gives a group row nothing to compare.
 */
export const inGroupOrder = (rows: OmClusterRow[], grouping: ClusterGrouping) =>
  grouping === NO_GROUPING ? rows : [...rows].sort(GROUP_ORDER[grouping]);

/** By environment when there is more than one, else by health. */
export const defaultGrouping = (rows: OmClusterRow[]): ClusterGrouping =>
  new Set(rows.map((row) => row.env_name ?? '')).size > 1
    ? 'env_name'
    : 'health';

/** A filter's picked values; a hand-written URL can carry one bare value. */
export const pickedValues = (value: unknown): string[] =>
  Array.isArray(value)
    ? value.map(String)
    : value == null || value === ''
      ? []
      : [String(value)];

/** Each filter's value, as its options, its counts and the URL spell it. */
const FILTER_VALUE: Record<
  FilterColumn,
  (row: OmClusterRow) => string | string[] | undefined
> = {
  health: (row) => row.health,
  type: (row) => TYPE_FILTER_VALUE[row.type],
  env_name: getEnvironmentValue,
  versions: (row) => row.versions,
};

export const getFilterValue = (id: FilterColumn, row: OmClusterRow) =>
  FILTER_VALUE[id](row);

/**
 * Exact match of any picked value against the row's filter value, or any of its values
 * for Version; MRT's multi-select default matches `prod` in `preprod`.
 */
export const filterBy =
  (id: FilterColumn): MRT_FilterFn<OmClusterRow> =>
  (row, _columnId, picked: unknown) => {
    const values = [getFilterValue(id, row.original) ?? []].flat();
    return pickedValues(picked).some((value) => values.includes(value));
  };

/** How many rows carry each value; a row counts once per value it has. */
export const countValues = (
  rows: OmClusterRow[],
  valuesOf: (row: OmClusterRow) => string | string[] | undefined
) => {
  // No prototype, so a value named `constructor` starts from zero.
  const counts: Record<string, number> = Object.create(null);
  for (const row of rows) {
    for (const value of [valuesOf(row) ?? []].flat()) {
      counts[value] = (counts[value] ?? 0) + 1;
    }
  }
  return counts;
};

/** Every version in the fleet, newest first. */
export const versionOptions = (rows: OmClusterRow[]): FilterOption[] =>
  [...new Set(rows.flatMap((row) => row.versions))]
    .sort((a, b) => compareVersions(b, a))
    .map((value) => ({ value, label: value }));

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
