// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

const SAFE_SEGMENT = /^([A-Za-z_][A-Za-z0-9_]*)((?:\[\d+\])*)$/;
const ARRAY_INDEX = /\[(\d+)\]/g;
const FORBIDDEN_KEYS = new Set([
  '__proto__',
  'constructor',
  'prototype',
  '__class__',
]);

/**
 * Resolve a dotted path against a record, returning the resolved value or
 * `undefined` when any segment of the path cannot be walked.
 *
 * Supports dotted property access and `[N]` array indices (e.g.
 * `"data.items[0].name"`). Prototype-walking keys (`__proto__`, `constructor`,
 * `prototype`, `__class__`) are refused. Inherited properties are not
 * resolved — only own string-keyed properties.
 *
 * Returns the resolved value verbatim, including `null`, `false`, `0`, and
 * empty strings — callers must apply their own empty-value guard.
 */
export function resolvePath(record: unknown, path: string): unknown {
  if (record === null || record === undefined || !path) {
    return undefined;
  }
  let cursor: unknown = record;
  for (const rawSegment of path.split('.')) {
    if (cursor === null || cursor === undefined || typeof cursor !== 'object') {
      return undefined;
    }
    const match = SAFE_SEGMENT.exec(rawSegment);
    if (!match) {
      return undefined;
    }
    const base = match[1];
    if (FORBIDDEN_KEYS.has(base)) {
      return undefined;
    }
    if (!Object.prototype.hasOwnProperty.call(cursor as object, base)) {
      return undefined;
    }
    cursor = (cursor as Record<string, unknown>)[base];
    const indices = match[2];
    if (indices) {
      for (const indexMatch of indices.matchAll(ARRAY_INDEX)) {
        if (!Array.isArray(cursor)) {
          return undefined;
        }
        cursor = cursor[Number(indexMatch[1])];
      }
    }
  }
  return cursor;
}
