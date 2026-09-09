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
 * Dev-only diagnostics for a malformed plugin schema.
 *
 * A schema arrives from the side-car at runtime, so a mistake in it cannot be
 * caught by this package's types or its tests — it surfaces as a form that
 * quietly renders wrong. These warnings make the common authoring mistakes
 * loud for whoever is writing the schema, and compile out of a production
 * bundle (Vite replaces `import.meta.env.DEV` statically).
 */

const IS_DEV = import.meta.env.DEV;

// One line per distinct problem, however many times the form re-renders.
const seen = new Set<string>();

export function warnSchema(message: string): void {
  if (!IS_DEV || seen.has(message)) {
    return;
  }
  seen.add(message);
  // eslint-disable-next-line no-console -- surface a malformed schema in dev
  console.warn(`[SchemaFormRenderer] ${message}`);
}

/** Test seam — the dedupe cache is module state and would leak between cases. */
export function resetSchemaWarnings(): void {
  seen.clear();
}
