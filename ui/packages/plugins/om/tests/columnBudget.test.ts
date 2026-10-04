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

import { describe, expect, it } from 'vitest';
import {
  UNIFORM_CANDIDATES,
  uniformColumnVisibility,
} from '../src/columnBudget';

/** A row carrying every candidate, so a case only exercises what it varies. */
const row = (over: Record<string, unknown> = {}) => ({
  env_name: 'production',
  cluster_name: 'rs-main',
  replication_set: 'rs-main',
  version: '7.0.24',
  installed_version: '7.0.24',
  vendor: 'percona',
  ...over,
});

const ALL_HIDDEN = Object.fromEntries(
  UNIFORM_CANDIDATES.map((key) => [key, false])
);

describe('uniformColumnVisibility', () => {
  // The finding this exists for: Environment was a dash in every row, and Cluster,
  // Replication set, Version and Vendor each repeated one value down the whole table.
  it('hides every column whose value never changes', () => {
    expect(uniformColumnVisibility([row(), row()])).toEqual(ALL_HIDDEN);
  });

  it('keeps a column that distinguishes rows', () => {
    const visibility = uniformColumnVisibility([
      row(),
      row({ version: '8.0.12' }),
    ]);

    expect(visibility.version).toBeUndefined();
    expect(visibility.env_name).toBe(false);
  });

  // The finding's own example: Environment was a dash in every row, which is a
  // column of nothing rather than a column of one repeated value.
  it('hides a column that is empty in every row', () => {
    const visibility = uniformColumnVisibility([
      row({ env_name: null }),
      row({ env_name: null }),
    ]);

    expect(visibility.env_name).toBe(false);
  });

  // With one row every column is uniform, so the rule would hide the whole table.
  it('hides nothing when there is only one row', () => {
    expect(uniformColumnVisibility([row()])).toEqual({});
  });

  it('hides nothing when there are no rows', () => {
    expect(uniformColumnVisibility([])).toEqual({});
  });

  // Identity and health are deliberately not candidates: a column going away is
  // itself a claim, and "every service is up" is not one a table should make by
  // rendering nothing.
  it('never considers identity or health columns', () => {
    expect(UNIFORM_CANDIDATES).not.toContain('service_name');
    expect(UNIFORM_CANDIDATES).not.toContain('status');
  });
});
