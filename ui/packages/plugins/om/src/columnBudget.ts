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

import type { OmServiceInventoryRow } from './types';

/**
 * The columns worth hiding when every row shares one value.
 *
 * Identity and health are deliberately absent: a column going away is itself a claim,
 * and "every service is up" is not one a table should make by rendering nothing.
 */
export const UNIFORM_CANDIDATES = [
  'env_name',
  'cluster_name',
  'replication_set',
  'version',
  'installed_version',
  'vendor',
] as const satisfies readonly (keyof OmServiceInventoryRow)[];

/**
 * Which of those columns carry one value down every row, and so answer nothing.
 *
 * Usually several do: one environment, one cluster, one version. The design review
 * counted six on a three-row table, each costing a column's width to say the same
 * word three times (P17).
 *
 * Nothing is hidden below two rows. With a single row *every* column is uniform, and
 * the rule would empty the table.
 */
export function uniformColumnVisibility(
  rows: readonly Partial<OmServiceInventoryRow>[]
): Record<string, boolean> {
  if (rows.length < 2) {
    return {};
  }
  const uniform: Record<string, boolean> = {};
  for (const key of UNIFORM_CANDIDATES) {
    const first = rows[0][key];
    if (rows.every((row) => row[key] === first)) {
      uniform[key] = false;
    }
  }
  return uniform;
}
