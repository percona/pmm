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

import type { PluginField } from '@sep/api';

/** Most entries a one-line summary spells out before it counts the rest. */
const MAX_SUMMARY_ENTRIES = 4;

/** Separator between entries, wide enough to read as a list on one line. */
const SUMMARY_SEPARATOR = ' · ';

/** What a section reads as when nothing in it carries a value. */
export const EMPTY_SECTION_SUMMARY = 'Using default values';

/**
 * Field types whose value does not summarise on one line: a file is a browser
 * handle rather than text, and a script preview is read-only output.
 */
const UNSUMMARISABLE_TYPES: ReadonlySet<PluginField['type']> = new Set([
  'file',
  'script_preview',
]);

/** The option label a choice value stands for, or the value itself. */
function choiceLabel(field: PluginField, value: unknown): string {
  if (field.type === 'choice' || field.type === 'multi_choice') {
    const match = field.choices.find((choice) => choice.value === value);
    if (match) {
      return match.label;
    }
  }
  return String(value);
}

/**
 * One `label: value` entry for a field, or null when it holds nothing worth
 * naming in a summary.
 *
 * An off switch is the resting state of nearly every boolean, so it is dropped
 * — unless the schema pre-filled it on, where off is the reader's own doing and
 * therefore the thing they most need to see without opening the section.
 */
export function summariseFieldValue(
  field: PluginField,
  value: unknown
): string | null {
  if (UNSUMMARISABLE_TYPES.has(field.type)) {
    return null;
  }
  if (value === undefined || value === null || value === '') {
    return null;
  }
  if (typeof value === 'boolean') {
    if (!value && !field.default) {
      return null;
    }
    return `${field.label}: ${value ? 'on' : 'off'}`;
  }
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return null;
    }
    const items = value.map((item) => choiceLabel(field, item)).join(', ');
    return `${field.label}: ${items}`;
  }
  return `${field.label}: ${choiceLabel(field, value)}`;
}

/**
 * One line naming what a collapsed section currently holds, so a reader can
 * check a form without opening every shell in it.
 *
 * `values` is positional against `fields` — react-hook-form's own
 * `getValues(names)` shape — so a caller never has to rebuild a name map.
 */
export function summariseSectionValues(
  fields: PluginField[],
  values: readonly unknown[]
): string {
  const entries: string[] = [];
  fields.forEach((field, index) => {
    const entry = summariseFieldValue(field, values[index]);
    if (entry !== null) {
      entries.push(entry);
    }
  });
  if (entries.length === 0) {
    return EMPTY_SECTION_SUMMARY;
  }
  if (entries.length <= MAX_SUMMARY_ENTRIES) {
    return entries.join(SUMMARY_SEPARATOR);
  }
  const shown = entries.slice(0, MAX_SUMMARY_ENTRIES).join(SUMMARY_SEPARATOR);
  return `${shown}${SUMMARY_SEPARATOR}+${entries.length - MAX_SUMMARY_ENTRIES} more`;
}
