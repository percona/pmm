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

/**
 * The schedules table's columns, in render order.
 *
 * One source of truth for both halves of the table: the panel renders these as
 * headers, and the row takes its edit-form `colSpan` from the count. They used
 * to be a header array and a hand-written `9` maintained separately, which was
 * survivable only while the set was fixed — it no longer is.
 *
 * `Chain` is dropped while nothing is chained (PMM-15454). These apps expose a
 * single chainable task, so no schedule can carry a chain and the column was an
 * unbroken run of em dashes advertising something the reader could not use. It
 * returns on its own the moment any schedule carries a chain.
 *
 * `Actions` is dropped for a session that may not mutate.
 */
export function scheduleColumnHeaders(
  showChain: boolean,
  showActions: boolean,
  itemLabel = 'Task'
): string[] {
  return [
    itemLabel,
    'Period',
    'Start Time',
    'Last Run',
    'Next Run',
    'Runs',
    ...(showChain ? ['Chain'] : []),
    'Enabled',
    ...(showActions ? ['Actions'] : []),
  ];
}
