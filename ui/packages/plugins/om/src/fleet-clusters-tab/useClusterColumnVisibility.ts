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

import { useState } from 'react';
import type { MRT_Updater, MRT_VisibilityState } from 'material-react-table';
import {
  CLUSTERS_TABLE_NAME,
  HIDDEN_BY_DEFAULT,
} from './FleetClustersTab.constants';

const STORAGE_KEY = `${CLUSTERS_TABLE_NAME}.columns`;

const read = (): MRT_VisibilityState => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    // Over the defaults, so a column hidden by default later stays hidden.
    return stored
      ? { ...HIDDEN_BY_DEFAULT, ...JSON.parse(stored) }
      : HIDDEN_BY_DEFAULT;
  } catch {
    return HIDDEN_BY_DEFAULT;
  }
};

/**
 * The chooser's choices, kept across visits. Owned here rather than by peak-ui's
 * Table, whose persisted state starts with every column shown (as Inventory's
 * `useIdColumnVisibility` does).
 */
export function useClusterColumnVisibility() {
  const [columnVisibility, setColumnVisibility] = useState(read);

  const onColumnVisibilityChange = (
    updater: MRT_Updater<MRT_VisibilityState>
  ) => {
    const next =
      typeof updater === 'function' ? updater(columnVisibility) : updater;
    setColumnVisibility(next);
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
    } catch {
      // Blocked storage only costs the choice outliving the visit.
    }
  };

  return { columnVisibility, onColumnVisibilityChange };
}
