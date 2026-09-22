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
 * Value normalization for the free-solo (`allow_custom`) path of the
 * `RemoteChoiceSelector`.
 *
 * The value-shaped counterpart to `FreeSoloSelect`'s `freeSoloValue`: a
 * `RemoteChoices` field commits a string (a fetched option's `value` or a
 * free-typed value), never an inventory id. The committed react-hook-form value
 * is a `string | null`:
 *   - picking an option — or typing a value that exactly matches an option's
 *     label — commits that option's `value`;
 *   - a typed value with no matching option commits the trimmed string;
 *   - clearing the field commits `null`.
 */

import type { ChoiceOption } from '@sep/api';

/** The value MUI Autocomplete renders: an option object, a free string, or empty. */
export type ChoiceFreeSoloDisplayValue = ChoiceOption | string | null;

/** The value committed to react-hook-form. */
export type ChoiceFreeSoloCommittedValue = string | null;

/**
 * Resolve the react-hook-form value into the value MUI Autocomplete should
 * display.
 *
 *   - a stored `value` string resolves to its matching option once the options
 *     have loaded, otherwise shows the raw string (a free-typed value, or a
 *     value whose option has not loaded yet);
 *   - a number is coerced to its string form before resolving (a numeric-string
 *     `value` persisted in a stored body);
 *   - an option object (a back-compat / persisted shape) resolves to the
 *     matching option, falling back to the object itself;
 *   - `''` / `null` / `undefined` show as empty.
 */
export function toDisplayValue(
  stored: unknown,
  options: readonly ChoiceOption[]
): ChoiceFreeSoloDisplayValue {
  if (stored === null || stored === undefined || stored === '') {
    return null;
  }
  if (typeof stored === 'number') {
    const asString = String(stored);
    return options.find((o) => o.value === asString) ?? asString;
  }
  if (typeof stored === 'string') {
    return options.find((o) => o.value === stored) ?? stored;
  }
  if (
    typeof stored === 'object' &&
    'value' in (stored as Record<string, unknown>)
  ) {
    const value = (stored as { value: unknown }).value;
    return options.find((o) => o.value === value) ?? (stored as ChoiceOption);
  }
  return null;
}

/**
 * Normalize the value emitted by MUI Autocomplete's `onChange` /
 * `onInputChange` into the `string | null` committed to react-hook-form.
 *
 *   - clearing yields `null`;
 *   - an option object yields its `value`;
 *   - a typed string that exactly matches a non-disabled option's label
 *     resolves to that option's `value` (not the label); a match against a
 *     disabled option is kept verbatim, since the UI presents disabled options
 *     as non-selectable;
 *   - any other non-empty string is kept verbatim;
 *   - a whitespace-only / empty string yields `null`.
 */
export function normalizeChange(
  next: ChoiceFreeSoloDisplayValue,
  options: readonly ChoiceOption[]
): ChoiceFreeSoloCommittedValue {
  if (next === null) {
    return null;
  }
  if (typeof next === 'object') {
    return next.value;
  }
  const trimmed = next.trim();
  if (trimmed === '') {
    return null;
  }
  const labelMatch = options.find((o) => o.label === trimmed && !o.disabled);
  return labelMatch ? labelMatch.value : trimmed;
}
