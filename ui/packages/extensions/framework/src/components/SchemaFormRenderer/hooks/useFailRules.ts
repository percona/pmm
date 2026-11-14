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

import { useMemo } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';
import type { FailRule, FormSection, PluginField } from '../types';
import { getAtPath, setAtPath } from '../utils/fieldPath';
import {
  evaluatePredicate,
  getGateFieldNames,
} from '../utils/predicateEvaluator';
import { coerceFormValues } from '../utils/validationMapper';
import { watchValuesByName } from '../utils/watchValuesByName';

export interface FailViolation {
  message: string;
  /** Field names to highlight in the UI — rendering hint from the schema. */
  error_fields: string[];
}

function defaultFailMessage(): string {
  return 'This combination of values is not allowed.';
}

/**
 * Evaluate several lists of fail_when rules under one `useWatch` subscription,
 * returning the violations of each list positionally.
 *
 * Given `fields`, values are coerced as submit coerces them, and a name in
 * `unsubmitted` reads as absent, since the payload will not carry it.
 */
function useFailRuleLists(
  ruleLists: FailRule[][],
  fields?: PluginField[],
  unsubmitted?: ReadonlySet<string>,
  fixedValues?: Record<string, unknown>
): FailViolation[][] {
  const { control } = useFormContext();

  const watchedNames = useMemo(() => {
    const names = new Set<string>();
    for (const rules of ruleLists) {
      for (const rule of rules) {
        // Watch all fields referenced in the predicate expression.
        for (const f of getGateFieldNames([{ when: rule.fail_when }])) {
          names.add(f);
        }
      }
    }
    return [...names];
  }, [ruleLists]);

  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  return useMemo(() => {
    const map = watchValuesByName(watchedNames, rawValues);
    if (fields) {
      // coerceFormValues works on nested values; the map is keyed by full path.
      const nested: Record<string, unknown> = {};
      for (const name of watchedNames) {
        setAtPath(nested, name, map[name]);
      }
      const coerced = coerceFormValues(nested, fields);
      for (const name of watchedNames) {
        map[name] = getAtPath(coerced, name);
      }
    }
    for (const name of unsubmitted ?? []) {
      if (Object.prototype.hasOwnProperty.call(map, name)) {
        map[name] = undefined;
      }
    }
    for (const [name, value] of Object.entries(fixedValues ?? {})) {
      if (Object.prototype.hasOwnProperty.call(map, name)) {
        map[name] = value;
      }
    }
    return ruleLists.map((rules) => {
      const violations: FailViolation[] = [];
      for (const rule of rules) {
        if (evaluatePredicate(rule.fail_when, map)) {
          violations.push({
            message: rule.message ?? defaultFailMessage(),
            error_fields: rule.error_fields,
          });
        }
      }
      return violations;
    });
  }, [rawValues, ruleLists, watchedNames, fields, unsubmitted, fixedValues]);
}

/**
 * Evaluates fail_when rules for all sections against current form values.
 *
 * A fail_when rule fires when its predicate evaluates to true — the inverse of
 * cardinality_rules (which fire when a count constraint is violated). Mirrors
 * the BE ConditionalRulesModel.apply_conditional_rules fail_when evaluation.
 *
 * Returns a parallel array of violations per section — index i corresponds to
 * sections[i]. Uses a single useWatch subscription over all referenced fields.
 */
export function useFailRules(sections: FormSection[]): FailViolation[][] {
  const ruleLists = useMemo(
    () => sections.map((s) => s.fail_when ?? []),
    [sections]
  );
  return useFailRuleLists(ruleLists);
}

/**
 * Evaluates the plugin schema's root fail_when rules against the values the
 * form would submit.
 *
 * Values are read the way the backend sees the same payload: coerced by field
 * type, and absent for a field the payload will not carry — gated out, or in an
 * inactive one-of branch. `fixedValues` are what the caller adds to the payload
 * outside the form, and are read as submitted.
 */
export function useRootFailRules(
  rules: FailRule[] | undefined,
  fields: PluginField[],
  unsubmitted: ReadonlySet<string>,
  fixedValues?: Record<string, unknown>
): FailViolation[] {
  const ruleLists = useMemo(() => [rules ?? []], [rules]);
  return useFailRuleLists(ruleLists, fields, unsubmitted, fixedValues)[0];
}
