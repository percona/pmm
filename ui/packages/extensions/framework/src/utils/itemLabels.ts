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

type ItemNameSource = {
  display_name?: string | null;
  item_display_name?: string | null;
  item_display_name_plural?: string | null;
};

function nonEmptyString(value: unknown): string | undefined {
  return typeof value === 'string' && value.length > 0 ? value : undefined;
}

/**
 * Singular item noun from the schema. Falls back to `display_name`, then
 * `'item'`, so JSX like `New {itemName}` never silently renders as `New `
 * when the field is missing (React drops `undefined` without showing it).
 */
export function resolveItemDisplayName(schema: ItemNameSource): string {
  return (
    nonEmptyString(schema.item_display_name) ??
    nonEmptyString(schema.display_name) ??
    'item'
  );
}

/**
 * Plural item noun from the schema. Falls back to `display_name`, then
 * `'items'`.
 */
export function resolveItemDisplayNamePlural(schema: ItemNameSource): string {
  return (
    nonEmptyString(schema.item_display_name_plural) ??
    nonEmptyString(schema.display_name) ??
    'items'
  );
}
