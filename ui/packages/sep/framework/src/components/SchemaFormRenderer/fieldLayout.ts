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

import type { PluginField } from './types';

/**
 * Sections lay their fields out on a two-column grid above `md`, which is the
 * difference between a form the common case can read at a glance and one that
 * scrolls. Single-column below that, so the layout still works in a narrow
 * pane.
 */
export const SECTION_GRID_SX = {
  display: 'grid',
  gridTemplateColumns: { xs: '1fr', md: 'repeat(2, minmax(0, 1fr))' },
  columnGap: 2,
} as const;

/**
 * Field types that read badly at half width — long free text, a code pane, or
 * a chip list that wraps as soon as it is narrowed.
 */
const FULL_ROW_TYPES: ReadonlySet<PluginField['type']> = new Set([
  'textarea',
  'yaml',
  'script_preview',
  'multi_choice',
  'file',
]);

/**
 * Whether a field claims the whole grid row rather than one column.
 *
 * `parentNames` are the fields some other field points at with `parent`; a
 * toggle with children takes a full row so its children land directly beneath
 * it rather than under whatever happened to share its row.
 */
export function isFullRowField(
  field: PluginField,
  parentNames?: ReadonlySet<string>
): boolean {
  // A nested child sits under its parent, so it has to start on a fresh row —
  // half of one would read as a sibling of whatever shares the row.
  if (field.parent) {
    return true;
  }
  if (parentNames?.has(field.name)) {
    return true;
  }
  return FULL_ROW_TYPES.has(field.type);
}

/**
 * The set of field names that some field in `fields` declares as its `parent`.
 *
 * Callers pass the whole form's fields rather than one section's. `parent` is
 * documented as same-section, but field names are payload keys and so unique
 * across the form, which makes the wider lookup equivalent and saves threading
 * section scope through every slot.
 */
export function collectParentNames(
  fields: readonly PluginField[]
): ReadonlySet<string> {
  const names = new Set<string>();
  for (const field of fields) {
    if (field.parent) {
      names.add(field.parent);
    }
  }
  return names;
}
