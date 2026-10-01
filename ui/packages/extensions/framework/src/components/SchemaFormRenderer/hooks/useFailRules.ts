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
import type { FormSection } from '../types';
import {
  evaluatePredicate,
  getGateFieldNames,
} from '../utils/predicateEvaluator';
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
  const { control } = useFormContext();

  const allRules = useMemo(
    () => sections.map((s) => s.fail_when ?? []),
    [sections]
  );

  const watchedNames = useMemo(() => {
    const names = new Set<string>();
    for (const rules of allRules) {
      for (const rule of rules) {
        // Watch all fields referenced in the predicate expression.
        for (const f of getGateFieldNames([{ when: rule.fail_when }])) {
          names.add(f);
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
  }, [rawValues, allRules, watchedNames]);
}
