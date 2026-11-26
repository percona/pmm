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

import { useMemo, useState } from 'react';
import type { MRT_ExpandedState, MRT_Updater } from 'material-react-table';
import { NO_GROUPING } from '../components/GroupBySelect';
import type { OmClusterRow } from '../types';
import type { ClusterGrouping } from './FleetClustersTab.types';
import { GROUP_VALUE } from './FleetClustersTab.utils';

/**
 * Groups open and clusters folded by default. TanStack cannot say "open unless closed",
 * so the closed groups and the opened clusters are kept apart and merged.
 */
export function useGroupedExpansion(
  rows: OmClusterRow[],
  grouping: ClusterGrouping
) {
  const [closedGroups, setClosedGroups] = useState<Record<string, boolean>>({});
  const [openClusters, setOpenClusters] = useState<Record<string, boolean>>({});
  // TanStack names a top-level group row `<columnId>:<groupingValue>`.
  const groupIds = useMemo(
    () =>
      grouping === NO_GROUPING
        ? []
        : [
            ...new Set(
              rows.map((row) => `${grouping}:${GROUP_VALUE[grouping](row)}`)
            ),
          ],
    [rows, grouping]
  );
  const expanded = useMemo(() => {
    const state: Record<string, boolean> = { ...openClusters };
    for (const id of groupIds) {
      if (!closedGroups[id]) {
        state[id] = true;
      }
    }
    return state;
  }, [groupIds, closedGroups, openClusters]);

  const onExpandedChange = (updater: MRT_Updater<MRT_ExpandedState>) => {
    const next = typeof updater === 'function' ? updater(expanded) : updater;
    // `true` is "expand all": every group and every cluster.
    const isOpen = (id: string) => next === true || !!next[id];
    setClosedGroups(
      Object.fromEntries(
        groupIds.filter((id) => !isOpen(id)).map((id) => [id, true])
      )
    );
    setOpenClusters(
      Object.fromEntries(
        rows.filter((row) => isOpen(row.id)).map((row) => [row.id, true])
      )
    );
  };

  return { expanded, onExpandedChange };
}
