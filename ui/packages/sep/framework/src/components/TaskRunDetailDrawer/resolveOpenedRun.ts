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

import type { TaskHistoryEntry } from '../../hooks/useTaskHistory';

/**
 * Re-read a clicked run out of the list it came from.
 *
 * {@link TaskRunDetailDrawer} renders an `entry` exactly as given, so a caller
 * that holds the row it was clicked with shows a stale run: one still in flight
 * when the drawer opened keeps counting elapsed time and never reaches its
 * terminal status, because the row object never changes even though the query
 * behind the table is polling.
 *
 * Pairing the clicked row with the live list fixes that without a second
 * request — the table's own query is already refetching while anything runs.
 *
 * The clicked row is returned unchanged when it has no id to match on, and when
 * it is no longer in `items` — a run that has paged out of the list should keep
 * the drawer showing what it last knew rather than collapsing to an empty view.
 */
export function resolveOpenedRun(
  opened: TaskHistoryEntry | null,
  items: TaskHistoryEntry[] | undefined
): TaskHistoryEntry | null {
  if (!opened || opened.id === null || opened.id === undefined) {
    return opened;
  }

  return items?.find((item) => item.id === opened.id) ?? opened;
}
