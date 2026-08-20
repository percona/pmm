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
 * Stable key for resetting cascaded children when an upstream selector changes.
 *
 * Handles hydrated option objects, numeric ids, digit-only strings (treated as
 * ids), and free-typed non-id strings (``custom:``). Empty / missing parents
 * share ``id:none``.
 */
export function cascadeParentResetKey(parent: unknown): string {
  if (parent && typeof parent === 'object' && 'id' in parent) {
    const id = (parent as { id: unknown }).id;
    return `id:${id ?? 'none'}`;
  }
  if (typeof parent === 'number' && Number.isFinite(parent)) {
    return `id:${parent}`;
  }
  if (typeof parent === 'string' && parent.trim() !== '') {
    const text = parent.trim();
    return /^\d+$/.test(text) ? `id:${text}` : `custom:${text}`;
  }
  return 'id:none';
}
