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
 * Apply a `value_labels` map to a raw value.
 *
 * `ListColumn` and `DetailField` both carry an optional map from a stored value
 * to the text a renderer should show, so a stored enum member reaches the screen
 * as the word it stands for — a backup type arrives on the wire as `"M"` and is
 * shown as `Mydumper`. The map is the backend's to publish; this is the one
 * place the frontend consumes it, shared by the list cells and both detail-field
 * components so a value cannot be labelled in a table and raw on a detail page.
 *
 * A value the map does not mention renders unchanged, which is what keeps a
 * schema that labels only some members from blanking the rest, and an app that
 * publishes no map at all behaves exactly as it did before labels existed.
 */
export function applyValueLabel(
  value: unknown,
  valueLabels: Record<string, string> | undefined
): unknown {
  if (!valueLabels || value === null || value === undefined) {
    return value;
  }
  // Own properties only. A bare `valueLabels[key]` reads through the prototype,
  // so a value that happens to spell `constructor`, `toString` or `__proto__`
  // would resolve to an inherited function or object, defeat the `??` fallback,
  // and hand React something it cannot render — a crash for a cell whose only
  // sin is its text. Keyed by the value's string form because the wire carries
  // the map's keys as strings even where the column holds numbers or booleans.
  const key = String(value);
  return Object.prototype.hasOwnProperty.call(valueLabels, key)
    ? valueLabels[key]
    : value;
}
