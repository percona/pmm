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
 * Extract a numeric id from a value that may be:
 *   - a safe-integer number (returned as-is),
 *   - a decimal-integer string (parsed; fractional, hex, whitespace-only and
 *     otherwise non-integer strings return `null`),
 *   - an option object with an `id` field (recursively extracted),
 *   - or anything else (returns `null`).
 *
 * Used by the cascading selectors to handle parent values that may come from
 * either an upstream selector (object form) or a persisted form default
 * (scalar form, possibly stringified).
 */
export function extractId(value: unknown): number | null {
  if (typeof value === 'number') {
    // Same bar as the string branch below: a fractional or unsafe-integer id
    // would otherwise pass here and enable a lookup no service can satisfy.
    return Number.isSafeInteger(value) ? value : null;
  }
  if (typeof value === 'string' && value !== '') {
    // Only decimal integers: `Number` would coerce a whitespace-only string to
    // `0` and accept `'1.5'` / `'0x10'`, each of which then reads as a
    // resolvable inventory id and fires a lookup for a service that cannot exist.
    const trimmed = value.trim();
    if (!/^-?\d+$/.test(trimmed)) {
      return null;
    }
    const n = Number(trimmed);
    return Number.isSafeInteger(n) ? n : null;
  }
  if (value && typeof value === 'object' && 'id' in value) {
    return extractId((value as { id: unknown }).id);
  }
  return null;
}
