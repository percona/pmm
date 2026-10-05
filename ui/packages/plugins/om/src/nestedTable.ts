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
 * Shared layout for the plain tables that render inside an unfolded row.
 *
 * A nested table is wider than the row it opens in, and the design review found
 * the consequence (P17): with no scroll container of its own it scrolls the whole
 * page sideways, and the first column -- the only one naming what each row *is* --
 * is the first thing to leave the screen. A reader is then looking at numbers
 * belonging to a service they can no longer identify.
 *
 * So: scroll inside the panel rather than the page, and keep the identity column
 * where it is while the rest moves.
 */

/**
 * The wrapper around a nested table: its own horizontal scroll, so the page does
 * not move when the panel is too wide for its row.
 */
export const NESTED_TABLE_WRAPPER = {
  p: 2,
  overflowX: 'auto',
} as const;

/**
 * The first cell of every row, head included, so what a row is stays readable
 * while its values scroll.
 *
 * An explicit background is load-bearing: a sticky cell with a transparent one
 * shows the scrolled cells sliding underneath it. `background.paper` is the
 * surface the detail panel already sits on, so pinning is invisible until the
 * table actually overflows.
 */
export const STICKY_IDENTITY_CELL = {
  position: 'sticky',
  left: 0,
  zIndex: 1,
  bgcolor: 'background.paper',
} as const;
