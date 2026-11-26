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
import type { OmClusterRow } from '../types';
import { countValues } from './FleetClustersTab.utils';

const rowsIn = (...environments: string[]) =>
  environments.map((env_name) => ({ env_name }) as OmClusterRow);

describe('countValues', () => {
  it('counts values named like object members from zero', () => {
    const counts = countValues(
      rowsIn('constructor', 'toString', 'toString'),
      (row) => row.env_name ?? undefined
    );

    expect(counts.constructor).toBe(1);
    expect(counts.toString).toBe(2);
  });
});
