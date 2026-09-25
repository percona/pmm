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
 * Map react-hook-form `useWatch({ name: [...] })` output to field names.
 *
 * RHF returns an array when `name` is an array, but may return a bare scalar
 * when exactly one name is watched depending on version/config.
 */
export function watchValuesByName(
  names: string[],
  raw: unknown
): Record<string, unknown> {
  if (names.length === 0) {
    return {};
  }
  if (names.length === 1) {
    const name = names[0];
    if (Array.isArray(raw)) {
      return { [name]: raw[0] };
    }
    return { [name]: raw };
  }
  const values = Array.isArray(raw) ? raw : [];
  return Object.fromEntries(names.map((n, i) => [n, values[i]]));
}
