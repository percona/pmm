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
 * Shared layout for the plain tables rendered inside an unfolded row.
 *
 * Scrolls inside the panel rather than moving the page, which is the first half of
 * P17's "nested tables should not scroll with the parent".
 *
 * The second half, "keep the identity column pinned", is deliberately not here. A
 * sticky cell has to paint an opaque background or the scrolled cells show through
 * it, and there is no correct colour to paint: the plugin sets no surface, so the
 * panel inherits the page's, and naming any token produced a visibly different block
 * down the identity column - on tables narrow enough never to scroll in the first
 * place. Matching the panel to the cell instead would have made the panel a second
 * table surface, which is what the same finding's next bullet asks us not to do, and
 * would have hardcoded a surface that P22 is about getting wrong.
 *
 * The pinning bought nothing today: the cluster services table is ten columns and the
 * scan receipt eight, and neither overflows its panel. The 24-column table that P17
 * was actually about is Services, and that has a detail drawer now. Pin a column when
 * one of these starts overflowing, against whatever surface P22 settles on.
 */
export const NESTED_TABLE_WRAPPER = {
  p: 2,
  overflowX: 'auto',
} as const;
