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

// Project-prefixed class applied to MRT paper containers so theme.ts can scope
// overrides without using MUI's reserved "Mui*" prefix.
export const SEP_TABLE_CLASS = 'SepTable';

/**
 * Widest a form in a SEP app may draw its inputs, in pixels.
 *
 * PMM's page container caps content at `DEFAULT_PAGE_MAX_WIDTH` (1000px), but a
 * SEP page opts out of that cap so its *lists* can use the screen. A form has
 * no such need: an input stretched across 1250px of window is harder to scan,
 * not easier, and the single-column stretch was the most visible way these
 * screens read as not-PMM (PMM-15456). Narrower than the page cap on purpose —
 * this is measured against line length, not against the container.
 *
 * 750px, not a new number: it matches the reading-width cap PMM's Settings
 * page already applies to text, per @pmcf-percona on PMM-15456 — one reading
 * width for the app, decided once here, that Settings can adopt too.
 */
export const FORM_CONTENT_MAX_WIDTH = 750;
