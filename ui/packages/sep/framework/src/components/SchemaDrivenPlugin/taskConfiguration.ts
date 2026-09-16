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

import type {
  FormSection,
  OneOfGroup,
  PluginField,
  SectionField,
} from '@sep/api';
import { evaluatePredicate } from '../SchemaFormRenderer/utils/predicateEvaluator';
import { fieldDefault } from '../SchemaFormRenderer/utils/fieldDefault';
import { getAtPath, setAtPath } from '../SchemaFormRenderer/utils/fieldPath';
import {
  collectOneOfGroups,
  flattenSectionFields,
  isOneOfGroup,
} from '../SchemaFormRenderer/utils/flattenSectionFields';

/**
 * Turn a task's stored create-form body into the settings a reader needs to see.
 *
 * What a reader wants is the answer to "what did someone actually set here",
 * and the create form already knows: it declares the labels, it declares the
 * defaults, and it declares which fields apply to the chosen type. This module
 * is the join — stored form values against the form schema — and it selects the
 * settings that differ from their default, under the labels the form used to
 * ask for them. The generated document stays available behind a disclosure for
 * the cases where the exact emitted config is the question.
 *
 * Deliberately not re-exported from the package index, unlike `getStoredForm`
 * next door: that one is part of the plugin-shell contract (a host decides
 * whether to offer Edit by asking whether a stored form exists), whereas this
 * is detail-page rendering.
 */

/** One configured setting, ready to render. */
export interface ConfiguredSetting {
  /** Form field name; stable key for React. */
  name: string;
  /** The form's own label for the field. */
  label: string;
  /** The form field's type, so a renderer can format the value by kind. */
  type: SectionField['type'];
  /** The stored value. */
  value: unknown;
  /** The field's display-label map, when it declares one. */
  valueLabels?: Record<string, string>;
}

/** Settings that differ from default, grouped as the form grouped them. */
export interface ConfiguredSection {
  title: string;
  settings: ConfiguredSetting[];
}

/**
 * Whether a stored value counts as "left at the default", and so not worth
 * listing as something this task configured.
 *
 * A blank value — absent, null, or empty string — always counts as default,
 * whatever the field declares. Three cases converge there and all three want
 * the same answer: the backend omits a null default from the wire, so a blank
 * optional field would otherwise read as a deliberate choice on every task; a
 * key missing from the stored body is a field the form never asked for; and a
 * field someone actively cleared has no value to show, so listing it would
 * print a label above an empty cell. Beyond that the comparison is structural,
 * which matters for the multi-choice fields whose value is an array.
 */
export function isDefaultValue(value: unknown, fieldDefault: unknown): boolean {
  const blank = (v: unknown) => v === undefined || v === null || v === '';
  if (blank(value)) {
    return true;
  }
  if (blank(fieldDefault)) {
    return false;
  }
  if (Array.isArray(value) && Array.isArray(fieldDefault)) {
    // Order-insensitive: the only array-valued field is `multi_choice`, whose
    // value is a set of selected members. Deselecting an option and picking it
    // again reorders the list without changing what is selected, and an
    // order-sensitive compare would report that as a configured setting.
    // Compared on the stringified members for the same reason the scalar branch
    // is loose — a choice backed by an int enum can arrive either way.
    if (value.length !== fieldDefault.length) {
      return false;
    }
    const remaining = fieldDefault.map(String);
    for (const item of value.map(String)) {
      const at = remaining.indexOf(item);
      if (at === -1) {
        return false;
      }
      // Spliced rather than membership-tested, so duplicates have to match in
      // count as well as in kind.
      remaining.splice(at, 1);
    }
    return true;
  }
  if (typeof value === 'object' || typeof fieldDefault === 'object') {
    return JSON.stringify(value) === JSON.stringify(fieldDefault);
  }
  // Loose on purpose at the last step: the wire carries an integer field's
  // default as a number and its stored value as a number, but a choice field
  // backed by an int enum can arrive as either, and `3 === '3'` being false
  // would list an untouched field as configured.
  return String(value) === String(fieldDefault);
}

/**
 * Whether any of a field's or section's `forbidden` gates fires for these
 * values — the same rule the form renderer uses to decide what to show, so the
 * configuration view hides exactly what the form would have hidden.
 */
function isGatedOut(
  gates: { when: Record<string, unknown> }[] | undefined,
  values: Record<string, unknown>
): boolean {
  return (gates ?? []).some((gate) => evaluatePredicate(gate.when, values));
}

/**
 * The taken branch's fields for a `one_of` group, else the item itself.
 *
 * A one-of group stores its chosen branch under the discriminator; the other
 * branches' fields are not part of this task's configuration even when the
 * stored body carries a leftover value for one of them.
 */
function selectedOneOfFields(
  item: SectionField,
  values: Record<string, unknown>
): PluginField[] {
  if (item.type !== 'one_of') {
    return [item];
  }
  // Through `getAtPath`, not a plain index: a discriminator is a dotted path
  // when the group is nested on the write model (`source.mode`), which is the
  // shape the schema types call out, and the form renderer resolves it the same
  // way. A plain read would miss it, fall through to the group default, and
  // silently drop every field of a branch someone did configure.
  const chosen = getAtPath(values, item.discriminator) ?? item.default;
  const branch = item.branches.find((b) => b.value === chosen);
  return branch ? branch.fields : [];
}

/** The branch a one-of group starts on, as the form renderer seeds it. */
function oneOfDefault(group: OneOfGroup): string {
  return group.default ?? group.branches[0]?.value ?? '';
}

/**
 * The branch a one-of group was switched to, as a setting of its own, or
 * `null` when the group was left on the branch it starts on.
 */
function selectedBranchSetting(
  group: OneOfGroup,
  values: Record<string, unknown>
): ConfiguredSetting | null {
  const chosen = getAtPath(values, group.discriminator);
  if (isDefaultValue(chosen, oneOfDefault(group))) {
    return null;
  }
  return {
    name: group.discriminator,
    label: group.label,
    type: group.type,
    value: chosen,
    valueLabels: Object.fromEntries(
      group.branches.map((branch) => [branch.value, branch.label])
    ),
  };
}

/**
 * The stored body with every default the form renderer seeds filled in, so
 * gates and discriminators read the values the form evaluated them against.
 */
function withFormDefaults(
  sections: FormSection[],
  storedForm: Record<string, unknown>
): Record<string, unknown> {
  const values = { ...storedForm };
  for (const field of flattenSectionFields(sections)) {
    setAtPath(
      values,
      field.name,
      getAtPath(values, field.name) ?? fieldDefault(field)
    );
  }
  for (const group of collectOneOfGroups(sections)) {
    setAtPath(
      values,
      group.discriminator,
      getAtPath(values, group.discriminator) ?? oneOfDefault(group)
    );
  }
  return values;
}

/**
 * Select the settings a task actually configured, grouped by form section.
 *
 * Sections and fields gated out for this task's values are dropped whole — a
 * Mydumper backup lists no XtraBackup settings — as are fields left at their
 * schema default and any name in `excludeNames`.
 *
 * `excludeNames` exists so the caller can avoid saying the same thing twice on
 * one page: the fields the detail page's own header and information card
 * already show do not need repeating three inches lower.
 *
 * Sections that end up empty are omitted, so a task that configured nothing
 * beyond the defaults produces an empty array rather than a run of empty
 * headings.
 */
export function selectConfiguredSettings(
  sections: FormSection[],
  storedForm: Record<string, unknown>,
  excludeNames: ReadonlySet<string> = new Set()
): ConfiguredSection[] {
  const values = withFormDefaults(sections, storedForm);
  const result: ConfiguredSection[] = [];

  for (const section of sections) {
    if (isGatedOut(section.forbidden, values)) {
      continue;
    }

    const settings: ConfiguredSetting[] = [];
    for (const item of section.fields) {
      const branchSetting = isOneOfGroup(item)
        ? selectedBranchSetting(item, values)
        : null;
      if (branchSetting && !excludeNames.has(branchSetting.name)) {
        settings.push(branchSetting);
      }
      for (const field of selectedOneOfFields(item, values)) {
        if (excludeNames.has(field.name)) {
          continue;
        }
        if (isGatedOut(field.forbidden, values)) {
          continue;
        }
        // Dotted for the same reason as the discriminator above — a one-of
        // branch's leaves carry nested paths — and `getAtPath` also refuses the
        // prototype-walking segments a raw index would happily follow. A field
        // the stored body never carried reads as `undefined`, which
        // `isDefaultValue` already treats as untouched.
        const value = getAtPath(storedForm, field.name);
        if (isDefaultValue(value, fieldDefault(field))) {
          continue;
        }
        settings.push({
          name: field.name,
          label: field.label,
          type: field.type,
          value,
          valueLabels: fieldValueLabels(field),
        });
      }
    }

    if (settings.length > 0) {
      result.push({ title: section.title, settings });
    }
  }

  return result;
}

/**
 * A choice field's own options, read as a display-label map.
 *
 * `value_labels` is declared on list columns and detail fields, not on form
 * fields — a form field carries its display text in its `choices` instead — so
 * the labels for a configured enum come from there.
 */
function fieldValueLabels(
  field: PluginField
): Record<string, string> | undefined {
  if (field.type !== 'choice' && field.type !== 'multi_choice') {
    return undefined;
  }
  const entries = field.choices.map(
    (choice) => [choice.value, choice.label] as const
  );
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
}
