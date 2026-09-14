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
 * Capitalise only the first character so sentence-start labels match the
 * OpenAPI contract for `item_display_name` / `item_display_name_plural`
 * (stored mid-sentence; the UI capitalises when the word opens a label).
 *
 * Nullish / non-string input returns `''` so a missing schema field cannot
 * throw at render or snackbar time.
 */
export function capitalizeItemLabel(name: string | null | undefined): string {
  if (typeof name !== 'string' || name.length === 0) {
    return '';
  }
  return name.charAt(0).toUpperCase() + name.slice(1);
}

type ItemNameSource = {
  display_name: string;
  item_display_name?: string | null;
  item_display_name_plural?: string | null;
};

/**
 * Singular item noun from the schema. Falls back to `display_name` when
 * `item_display_name` is missing (same default the backend documents).
 */
export function resolveItemDisplayName(schema: ItemNameSource): string {
  const name = schema.item_display_name;
  return typeof name === 'string' && name.length > 0 ? name : schema.display_name;
}

/**
 * Plural item noun from the schema. Falls back to `display_name` when
 * `item_display_name_plural` is missing.
 */
export function resolveItemDisplayNamePlural(schema: ItemNameSource): string {
  const name = schema.item_display_name_plural;
  return typeof name === 'string' && name.length > 0 ? name : schema.display_name;
}
