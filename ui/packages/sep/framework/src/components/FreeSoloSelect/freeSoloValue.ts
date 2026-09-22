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
 * Value normalization shared by the free-solo (`allow_custom`) path of the
 * reference selectors (`SchemaSelector`, `TableSelector`, `ServiceSelector`,
 * `HostSelector`).
 *
 * A free-solo reference field collapses "pick from inventory" and "type a new
 * value" into one control. The committed react-hook-form value is an
 * `int | str` (mirroring the backend `SchemaRef` / `HostRef(allow_custom=True)`
 * contracts):
 *   - an inventory pick — or a typed value that exactly matches an inventory
 *     option's label — commits the inventory **id** (`number` for service /
 *     schema / table, `string` for host);
 *   - a typed value with no matching option commits the raw **string**;
 *   - clearing the field commits `null`.
 *
 * `coerceFormValues` then passes a scalar `number`/`string` straight through on
 * submit, so no extra serialisation step is needed.
 */

/** Minimal shape every reference option satisfies. */
export interface ReferenceOption {
  /** Inventory id — numeric for service/schema/table, string for host. */
  id: number | string;
  name: string;
}

/** The value MUI Autocomplete renders: an option object, a free string, or empty. */
export type FreeSoloDisplayValue<T extends ReferenceOption> = T | string | null;

/** The value committed to react-hook-form. */
export type FreeSoloCommittedValue = number | string | null;

/**
 * Resolve the react-hook-form value into the value MUI Autocomplete should
 * display.
 *
 *   - a `number` (inventory id) resolves to its matching option once the
 *     options have loaded; until then it shows empty rather than a bare numeric
 *     id (which would also mismatch `options` and trip an MUI warning);
 *   - a non-empty `string` that matches an option's `id` (string host ids, or
 *     a stringified numeric id) resolves to that option; otherwise it is shown
 *     as a free-typed value;
 *   - an option object (a back-compat / persisted shape) resolves to the
 *     matching option, falling back to the object itself;
 *   - `''` / `null` / `undefined` show as empty.
 */
export function toDisplayValue<T extends ReferenceOption>(
  stored: unknown,
  options: readonly T[]
): FreeSoloDisplayValue<T> {
  if (stored === null || stored === undefined || stored === '') {
    return null;
  }
  if (typeof stored === 'number') {
    return options.find((o) => o.id === stored) ?? null;
  }
  if (typeof stored === 'string') {
    const idMatch = options.find(
      (o) => o.id === stored || String(o.id) === stored
    );
    return idMatch ?? stored;
  }
  if (
    typeof stored === 'object' &&
    'id' in (stored as Record<string, unknown>)
  ) {
    const id = (stored as { id: unknown }).id;
    return options.find((o) => o.id === id) ?? (stored as T);
  }
  return null;
}

/**
 * Normalize the value emitted by MUI Autocomplete's `onChange` /
 * `onInputChange` into the `int | str | null` committed to react-hook-form.
 *
 *   - clearing yields `null`;
 *   - an option object yields its `id`;
 *   - a typed string that exactly matches an option's label resolves to that
 *     option's `id` (not the string);
 *   - any other non-empty string is kept verbatim;
 *   - a whitespace-only / empty string yields `null`.
 */
export function normalizeChange<T extends ReferenceOption>(
  next: FreeSoloDisplayValue<T>,
  options: readonly T[],
  getOptionLabel: (option: T) => string
): FreeSoloCommittedValue {
  if (next === null) {
    return null;
  }
  if (typeof next === 'object') {
    return next.id;
  }
  const trimmed = next.trim();
  if (trimmed === '') {
    return null;
  }
  const match = options.find((o) => getOptionLabel(o) === trimmed);
  return match ? match.id : trimmed;
}
