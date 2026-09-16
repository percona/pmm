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

import type { PluginField } from '../types';

/**
 * The value that reads as *unset* for a field's widget.
 *
 * Matches what the backend's ``_field_is_present`` treats as absent, so a field
 * carrying this value satisfies a `forbidden` gate rather than tripping it.
 * Deliberately ignores the schema's `default`: this is what a field is cleared
 * *to*, not what it starts *at*.
 */
export function emptyFieldValue(field: PluginField): unknown {
  switch (field.type) {
    case 'bool':
      return false;
    case 'multi_choice':
      return [];
    case 'file':
      return undefined;
    case 'remote_choice':
      // Mirror the selector's empty commit (`null`), not `''` — an empty string
      // passes some client checks yet fails the backend NonEmptyStr with
      // "String should have at least 1 character".
      return null;
    default:
      return '';
  }
}

/** The value a field starts at: its schema `default`, else {@link emptyFieldValue}. */
export function fieldDefault(field: PluginField): unknown {
  if (field.type === 'file') {
    return undefined;
  }
  return field.default ?? emptyFieldValue(field);
}
