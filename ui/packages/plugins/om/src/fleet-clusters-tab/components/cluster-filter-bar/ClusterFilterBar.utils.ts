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

import type { OmClusterRow } from '../../../types';
import {
  TYPE_FILTER_VALUE,
  UNNAMED_ENVIRONMENT_VALUE,
} from '../../FleetClustersTab.constants';
import type { FilterColumn } from '../../FleetClustersTab.types';

export const getEnvironmentValue = (row: OmClusterRow) =>
  row.env_name || UNNAMED_ENVIRONMENT_VALUE;

export const getEnvironments = (rows: OmClusterRow[]) =>
  // The unnamed environment last, as in Group by.
  [...new Set(rows.map(getEnvironmentValue))].sort(
    (a, b) =>
      Number(a === UNNAMED_ENVIRONMENT_VALUE) -
        Number(b === UNNAMED_ENVIRONMENT_VALUE) || a.localeCompare(b)
  );

/** Each filter's value, as its options and the URL spell it. */
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
