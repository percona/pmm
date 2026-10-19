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
import type { CardinalityRule, FormSection } from '../types';
import {
  evaluatePredicate,
  getGateFieldNames,
  isPresent,
} from '../utils/predicateEvaluator';
import { watchValuesByName } from '../utils/watchValuesByName';

export interface CardinalityViolation {
  message: string;
}

function defaultMessage(fields: string[], min: number, max: number): string {
  const n = fields.length;
  if (min === max) {
    return `Exactly ${min} of these ${n} fields must be filled: ${fields.join(', ')}.`;
  }
  if (max === Infinity) {
    return `At least ${min} of these ${n} fields must be filled: ${fields.join(', ')}.`;
  }
  if (min === 0) {
    return `At most ${max} of these ${n} fields may be filled: ${fields.join(', ')}.`;
  }
  return `Between ${min} and ${max} of these ${n} fields must be filled: ${fields.join(', ')}.`;
}

function checkRule(
  rule: CardinalityRule,
  map: Record<string, unknown>
): CardinalityViolation | null {
  if (rule.when && !evaluatePredicate(rule.when, map)) {
    return null;
  }
  const presentCount = rule.fields.filter((f) => isPresent(map[f])).length;
  const min = rule.min ?? 0;
  const max = rule.max ?? Infinity;
  if (presentCount < min || presentCount > max) {
    return { message: rule.message ?? defaultMessage(rule.fields, min, max) };
  }
  return null;
}

/**
 * Evaluates cardinality_rules for all sections against current form values.
 *
 * Returns a parallel array of violations per section — index i corresponds to
 * sections[i]. Uses a single useWatch subscription over all referenced fields
 * so there is only one React subscription per form regardless of rule count.
 */
export function useCardinalityRules(
  sections: FormSection[]
): CardinalityViolation[][] {
  const { control } = useFormContext();

  const allRules = useMemo(
    () => sections.map((s) => s.cardinality_rules ?? []),
    [sections]
  );

  const watchedNames = useMemo(() => {
    const names = new Set<string>();
    for (const rules of allRules) {
      for (const rule of rules) {
        for (const f of rule.fields) {
          names.add(f);
        }
        if (rule.when) {
          for (const f of getGateFieldNames([{ when: rule.when }])) {
            names.add(f);
          }
        }
      }
    }
    return [...names];
  }, [allRules]);

  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  return useMemo(() => {
    const map = watchValuesByName(watchedNames, rawValues);
    return allRules.map((rules) => {
      const violations: CardinalityViolation[] = [];
      for (const rule of rules) {
        const v = checkRule(rule, map);
        if (v) {
          violations.push(v);
        }
      }
      return violations;
    });
  }, [rawValues, allRules, watchedNames]);
}
