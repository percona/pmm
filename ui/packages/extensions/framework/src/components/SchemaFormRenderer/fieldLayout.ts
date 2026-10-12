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

/** What one pass over a form's fields yields for the layout to read back. */
interface FieldIndex {
  /** Names some field declares as its `parent`. */
  parentNames: ReadonlySet<string>;
  /** Every field by name, for resolving a parent pointer to its label. */
  byName: ReadonlyMap<string, PluginField>;
}

// Every slot in a form is handed the same fields array from context, so the
// index is derived once per form rather than once per slot — a component-local
// `useMemo` would still walk the whole form for each of its N fields.
const indexCache = new WeakMap<readonly PluginField[], FieldIndex>();

/**
 * Return the derived index for a form's fields, computing it at most once.
 *
 * Callers pass the whole form's fields rather than one section's. `parent` is
 * documented as same-section, but field names are payload keys and so unique
 * across the form, which makes the wider lookup equivalent and saves threading
 * section scope through every slot.
 */
export function fieldIndex(fields: readonly PluginField[]): FieldIndex {
  const cached = indexCache.get(fields);
  if (cached) {
    return cached;
  }
  const parentNames = new Set<string>();
  const byName = new Map<string, PluginField>();
  for (const field of fields) {
    byName.set(field.name, field);
    if (field.parent) {
      parentNames.add(field.parent);
    }
  }
  const index: FieldIndex = { parentNames, byName };
  indexCache.set(fields, index);
  return index;
}
