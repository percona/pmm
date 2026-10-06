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

/** Shared layout for the plain tables rendered inside an unfolded row. */

/** Scrolls inside the panel, so a wide nested table does not move the page. */
export const NESTED_TABLE_WRAPPER = {
  p: 2,
  overflowX: 'auto',
} as const;

/**
 * The first cell of every row, head included, so what a row is stays readable while
 * its values scroll. The background is load-bearing: a transparent sticky cell shows
 * the scrolled cells sliding underneath it.
 */
export const STICKY_IDENTITY_CELL = {
  position: 'sticky',
  left: 0,
  zIndex: 1,
  bgcolor: 'background.paper',
} as const;
