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

import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import {
  CLUSTERS_TABLE_NAME,
  HIDDEN_BY_DEFAULT,
} from './FleetClustersTab.constants';
import { useClusterColumnVisibility } from './useClusterColumnVisibility';

const STORAGE_KEY = `${CLUSTERS_TABLE_NAME}.columns`;

const visibility = () =>
  renderHook(() => useClusterColumnVisibility()).result.current
    .columnVisibility;

describe('useClusterColumnVisibility', () => {
  afterEach(() => localStorage.clear());

  it('keeps hidden a default column the stored choice does not mention', () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ max_cpu_percent: true, env_name: false })
    );

    expect(visibility()).toEqual({
      ...HIDDEN_BY_DEFAULT,
      max_cpu_percent: true,
      env_name: false,
    });
  });

  it('falls back to the defaults for a stored null', () => {
    localStorage.setItem(STORAGE_KEY, 'null');

    expect(visibility()).toEqual(HIDDEN_BY_DEFAULT);
  });
});
