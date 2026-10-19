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

import { toDatetimeLocalValue } from '../../../utils/datetimeLocal';
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

function hasSchemaDefault(field: PluginField): boolean {
  return field.default !== undefined && field.default !== null;
}

function isTextLike(
  field: PluginField
): field is Extract<
  PluginField,
  { type: 'string' } | { type: 'textarea' } | { type: 'yaml' }
> {
  return (
    field.type === 'string' ||
    field.type === 'textarea' ||
    field.type === 'yaml'
  );
}

/**
 * The value a field starts at.
 *
 * Order:
 * 1. schema `default` when set
 * 2. for required integer/float with no default — `ge`, else `1`, so a required
 *    Minutes field is not empty on first paint
 * 3. otherwise {@link emptyFieldValue}
 *
 * Placeholders stay grey ghost text only. Promoting them into the starting
 * value would submit example copy (e.g. "e.g. 2026-01-01…") and bypass
 * required checks; real defaults belong in schema `default` (PMM-15518).
 */
export function fieldDefault(field: PluginField): unknown {
  if (field.type === 'file') {
    return undefined;
  }
  if (hasSchemaDefault(field)) {
    // `datetime-local` cannot display a UTC ISO string; convert here so the
    // picker mounts with a real wall-clock value (PMM-15510).
    if (field.type === 'datetime') {
      return toDatetimeLocalValue(field.default);
    }
    return field.default;
  }
  if (field.required && (field.type === 'integer' || field.type === 'float')) {
    return typeof field.ge === 'number' ? field.ge : 1;
  }
  return emptyFieldValue(field);
}

/**
 * Grey placeholder text for the widget.
 *
 * Always the schema's `placeholder` for text-like fields. Format hints and
 * examples stay ghost text; they are never submitted.
 */
export function fieldPlaceholder(field: PluginField): string | undefined {
  if (!isTextLike(field) || !field.placeholder) {
    return undefined;
  }
  return field.placeholder;
}
