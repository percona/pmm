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

const FORBIDDEN_SEGMENTS = new Set([
  '__proto__',
  'constructor',
  'prototype',
  '__class__',
]);

function pathHasForbiddenSegment(path: string): boolean {
  return path
    .split('.')
    .some((segment) => segment === '' || FORBIDDEN_SEGMENTS.has(segment));
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return false;
  }
  const proto = Object.getPrototypeOf(value) as unknown;
  return proto === Object.prototype || proto === null;
}

/** Read a value at ``path``, supporting dotted nested keys on plain objects. */
export function getAtPath(
  values: Record<string, unknown>,
  path: string
): unknown {
  if (pathHasForbiddenSegment(path)) {
    return undefined;
  }
  if (Object.prototype.hasOwnProperty.call(values, path)) {
    return values[path];
  }
  if (!path.includes('.')) {
    return undefined;
  }
  const segments = path.split('.');
  let current: unknown = values;
  for (const segment of segments) {
    if (
      !isPlainObject(current) ||
      !Object.prototype.hasOwnProperty.call(current, segment)
    ) {
      return undefined;
    }
    current = current[segment];
  }
  return current;
}

/** Write ``value`` at ``path``, creating intermediate plain objects as needed.

Shallow-clones existing plain-object intermediates along the path so callers
that shallow-copy the root (for example ``coerceFormValues``) do not mutate
nested objects still referenced from the input.
*/
export function setAtPath(
  target: Record<string, unknown>,
  path: string,
  value: unknown
): void {
  if (pathHasForbiddenSegment(path)) {
    return;
  }
  const segments = path.split('.');
  let current = target;
  for (let index = 0; index < segments.length - 1; index += 1) {
    const segment = segments[index];
    const next = current[segment];
    if (isPlainObject(next)) {
      const cloned: Record<string, unknown> = { ...next };
      current[segment] = cloned;
      current = cloned;
    } else {
      const created: Record<string, unknown> = {};
      current[segment] = created;
      current = created;
    }
  }
  const leaf = segments[segments.length - 1];
  if (value === undefined) {
    delete current[leaf];
    return;
  }
  current[leaf] = value;
}
