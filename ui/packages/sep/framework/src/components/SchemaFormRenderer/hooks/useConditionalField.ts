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

import { useEffect, useMemo } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import type { FieldGate, PluginField, Predicate } from '../types';
import { useFormFields } from '../formFieldsContext';
import { emptyFieldValue } from '../utils/fieldDefault';
import { warnSchema } from '../utils/schemaWarnings';
import {
  evaluatePredicate,
  getGateFieldNames,
} from '../utils/predicateEvaluator';
import { watchValuesByName } from '../utils/watchValuesByName';

export interface ConditionalFieldState {
  isHidden: boolean;
  isRequired: boolean;
  /**
   * The field declares a `parent` that is currently off. It renders, indented
   * beneath that parent, but takes no input — see {@link PluginField.parent}.
   */
  isDisabled: boolean;
}

/**
 * Whether a gate is exactly "the parent toggle is off".
 *
 * A parented field keeps the backend's own `forbidden` gate on its parent —
 * `Ui(parent=...)` is presentation-only, so the runtime rule has to live
 * somewhere — and that gate would otherwise hide the very field the parent
 * pointer asks to show greyed out. Matching it structurally lets the renderer
 * consume it as the disable condition while every other gate on the field goes
 * on hiding it.
 */
function isParentOffGate(when: Predicate, parent: string): boolean {
  const entries = Object.entries(when);
  if (entries.length !== 1) {
    return false;
  }
  const [op, operand] = entries[0];
  if (op === 'falsy') {
    return operand === parent;
  }
  if (op !== 'not') {
    return false;
  }
  const innerEntries = Object.entries(operand as Predicate);
  if (innerEntries.length !== 1) {
    return false;
  }
  const [innerOp, innerOperand] = innerEntries[0];
  return innerOp === 'truthy' && innerOperand === parent;
}

/**
 * A field's `forbidden` gates minus the one its `parent` pointer accounts for.
 *
 * The parent-off gate is the backend's runtime rule for the same relationship
 * the pointer describes; the renderer consumes it as the disable condition, so
 * applying it again as a hide would make the nested child vanish instead of
 * greying out. Everything else on the field still hides it.
 */
export function hidingGates(field: PluginField): FieldGate[] {
  const gates = field.forbidden ?? [];
  const parent = field.parent;
  if (!parent) {
    return gates;
  }
  return gates.filter((gate) => !isParentOffGate(gate.when, parent));
}

/**
 * Resolve one field's visibility, required-ness and disabled state.
 *
 * Pure: keeping a hidden or disabled field's value out of the submission
 * payload belongs to {@link useFieldPayloadCleanup}, which the form body calls
 * once for every field in the schema. That split is not cosmetic — a field
 * inside a collapsed section is never mounted, so a hook that only runs while
 * the slot is on screen cannot be the thing that guards the payload.
 */
export function useConditionalField(field: PluginField): ConditionalFieldState {
  const { control } = useFormContext();
  const formFields = useFormFields();
  const parent = field.parent;

  // The renderer trusts `parent` and derives the disable state from the named
  // field's truthiness alone; it cannot validate the schema. Say so out loud
  // in dev, because each of these mistakes renders a plausible-looking form
  // that behaves wrongly rather than failing.
  useEffect(() => {
    if (!parent) {
      return;
    }
    const target = formFields.find((f) => f.name === parent);
    if (!target) {
      warnSchema(
        `field '${field.name}' names parent '${parent}', which is not a field ` +
          `in this form. It will stay disabled forever.`
      );
      return;
    }
    if (target.type !== 'bool') {
      warnSchema(
        `field '${field.name}' names parent '${parent}', which is a ` +
          `'${target.type}' field. Only a bool can be a parent toggle.`
      );
    }
    if (target.parent) {
      warnSchema(
        `field '${field.name}' names parent '${parent}', which is itself ` +
          `parented to '${target.parent}'. Chained parents are not supported ` +
          `and a cycle leaves both toggles permanently inert.`
      );
    }
    if (!(field.forbidden ?? []).some((g) => isParentOffGate(g.when, parent))) {
      warnSchema(
        `field '${field.name}' names parent '${parent}' but carries no ` +
          `forbidden gate on that parent being falsy. The renderer disables ` +
          `the field, but nothing stops the backend accepting a value for it.`
      );
    }
  }, [field, parent, formFields]);

  // Gates the parent pointer already accounts for drop out of the hide set;
  // what remains still hides the field the way it always has.
  const forbidden = useMemo<FieldGate[]>(() => hidingGates(field), [field]);

  const watchedNames = useMemo<string[]>(() => {
    const requires = field.requires ?? [];
    const names = getGateFieldNames([...requires, ...forbidden]);
    if (parent && !names.includes(parent)) {
      names.push(parent);
    }
    return names;
    // field schema is static for the form's lifetime; dep on field is safe.
  }, [field, forbidden, parent]);

  // disabled=true when there are no gate fields — avoids subscribing to the
  // whole form when useWatch receives an empty name array.
  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  const isHidden = useMemo(() => {
    if (!forbidden.length) {
      return false;
    }
    const map = watchValuesByName(watchedNames, rawValues);
    return forbidden.some((gate) => evaluatePredicate(gate.when, map));
  }, [rawValues, forbidden, watchedNames]);

  const isDisabled = useMemo(() => {
    if (!parent) {
      return false;
    }
    const map = watchValuesByName(watchedNames, rawValues);
    return !evaluatePredicate({ truthy: parent }, map);
  }, [rawValues, parent, watchedNames]);

  const isRequired = useMemo(() => {
    // A field the user cannot reach must never block submission.
    if (isDisabled) {
      return false;
    }
    if (field.required) {
      return true;
    }
    if (!field.requires?.length) {
      return false;
    }
    const map = watchValuesByName(watchedNames, rawValues);
    return field.requires.some((gate) => evaluatePredicate(gate.when, map));
  }, [rawValues, field, watchedNames, isDisabled]);

  return { isHidden, isRequired, isDisabled };
}

/**
 * Keep gated-out values out of the submission payload, for every field in the
 * schema at once.
 *
 * Owned by the form body rather than by each field's slot, because a slot
 * inside a collapsed section or a collapsed group is never mounted:
 * `buildFormDefaults` still seeds its value, so a hook that ran only while the
 * slot was on screen would let a gated-out field's default ship whenever the
 * user submitted without expanding the section.
 *
 * Hidden wins over disabled: a hidden field is unregistered, which drops the
 * key entirely, so clearing it afterwards would only put it back.
 */
export function useFieldPayloadCleanup(fields: PluginField[]): void {
  const { control, unregister, setValue } = useFormContext();

  const watchedNames = useMemo<string[]>(() => {
    const names = new Set<string>();
    for (const field of fields) {
      for (const name of getGateFieldNames(hidingGates(field))) {
        names.add(name);
      }
      if (field.parent) {
        names.add(field.parent);
      }
    }
    return [...names];
  }, [fields]);

  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  const { hide, clear } = useMemo(() => {
    const map = watchValuesByName(watchedNames, rawValues);
    const hidden: string[] = [];
    const cleared: PluginField[] = [];
    for (const field of fields) {
      if (
        hidingGates(field).some((gate) => evaluatePredicate(gate.when, map))
      ) {
        hidden.push(field.name);
        continue;
      }
      if (field.parent && !evaluatePredicate({ truthy: field.parent }, map)) {
        cleared.push(field);
      }
    }
    return { hide: hidden, clear: cleared };
  }, [fields, watchedNames, rawValues]);

  // `useWatch` returns a fresh array every render, so the memo above does too.
  // Key the effect on what it would actually do instead: `setValue` notifies
  // subscribers even when the value is unchanged, so an effect that re-ran on
  // every render would spin.
  const signature = `${hide.join('|')}#${clear.map((f) => f.name).join('|')}`;

  useEffect(() => {
    for (const name of hide) {
      unregister(name);
    }
    for (const field of clear) {
      setValue(field.name, emptyFieldValue(field), {
        shouldDirty: false,
        shouldValidate: false,
      });
    }
    // `hide` / `clear` are re-derived every render; `signature` is what
    // actually changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [signature, unregister, setValue]);
}
