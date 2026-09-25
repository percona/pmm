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

import { extractId } from './extractId';

/** Minimal inventory option shape used by reference selectors. */
export interface ReferenceOption {
  id: number;
  name?: string;
}

/**
 * Return whether `value` is a fully hydrated inventory option object.
 *
 * Persisted `data['_form']` bodies store reference fields as scalar ids;
 * live create/edit forms store the richer option object until submit coercion.
 */
export function isHydratedReferenceOption(
  value: unknown
): value is ReferenceOption {
  return (
    value !== null &&
    typeof value === 'object' &&
    'id' in value &&
    'name' in value &&
    typeof (value as { name: unknown }).name === 'string'
  );
}

/**
 * Resolve a stored reference value into an inventory option for display.
 *
 * Accepts a scalar id (number or numeric string), a partially populated option
 * object, or an already hydrated option. Returns `null` when the id cannot be
 * matched in `options` yet (for example while services are still loading).
 */
export function resolveReferenceOption<T extends ReferenceOption>(
  stored: unknown,
  options: readonly T[]
): T | null {
  if (stored === null || stored === undefined || stored === '') {
    return null;
  }
  const id = extractId(stored);
  if (id !== null) {
    const match = options.find((option) => option.id === id);
    if (match) {
      return match;
    }
    if (isHydratedReferenceOption(stored)) {
      return stored as T;
    }
    return null;
  }
  return null;
}
