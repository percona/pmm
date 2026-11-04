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
import type { FailRule, FormSection } from '../types';
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
 * Evaluate several lists of fail_when rules under one `useWatch` subscription,
 * returning the violations of each list positionally.
 *
 * A name in `unsubmitted` reads as absent, since the payload will not carry it.
 */
function useFailRuleLists(
  ruleLists: FailRule[][],
  unsubmitted?: ReadonlySet<string>
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
    for (const name of unsubmitted ?? []) {
      if (Object.prototype.hasOwnProperty.call(map, name)) {
        map[name] = undefined;
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
  }, [rawValues, ruleLists, watchedNames, unsubmitted]);
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
 * A root rule may name fields in any section, so a field the payload will not
 * carry — gated out, or in an inactive one-of branch — reads as absent, the way
 * the backend sees the same payload.
 */
export function useRootFailRules(
  rules: FailRule[] | undefined,
  unsubmitted: ReadonlySet<string>
): FailViolation[] {
  const ruleLists = useMemo(() => [rules ?? []], [rules]);
  return useFailRuleLists(ruleLists, unsubmitted)[0];
}
