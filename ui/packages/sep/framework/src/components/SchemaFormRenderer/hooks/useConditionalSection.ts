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
import type { FormSection } from '../types';
import { isOneOfGroup } from '../utils/flattenSectionFields';
import {
  evaluatePredicate,
  getGateFieldNames,
} from '../utils/predicateEvaluator';
import { watchValuesByName } from '../utils/watchValuesByName';

export interface ConditionalSectionState {
  isHidden: boolean;
}

/**
 * Evaluate a FormSection's ``forbidden`` gates against current form values.
 * When any gate fires the section becomes hidden and every child field is
 * unregistered from react-hook-form so stale values do not ship in the
 * submission payload.
 *
 * Cardinality (repeated) sections are out of scope for this hook — none
 * of the current schema-driven plugins gate a cardinality_rules-driven
 * section. Revisit if a future plugin needs the combination.
 */
export function useConditionalSection(
  section: FormSection
): ConditionalSectionState {
  const { control, unregister } = useFormContext();

  const watchedNames = useMemo<string[]>(() => {
    const forbidden = section.forbidden ?? [];
    return getGateFieldNames(forbidden);
    // section schema is static for the form's lifetime; dep on section is safe.
  }, [section]);

  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  const isHidden = useMemo(() => {
    if (!section.forbidden?.length) {
      return false;
    }
    const map = watchValuesByName(watchedNames, rawValues);
    return section.forbidden.some((gate) => evaluatePredicate(gate.when, map));
  }, [rawValues, section, watchedNames]);

  const sectionFieldNames = useMemo<string[]>(() => {
    const names = new Set<string>();
    for (const field of section.fields) {
      if (isOneOfGroup(field)) {
        names.add(field.name);
        names.add(field.discriminator);
        for (const branch of field.branches) {
          for (const leaf of branch.fields) {
            names.add(leaf.name);
          }
        }
        continue;
      }
      names.add(field.name);
    }
    return [...names];
  }, [section.fields]);

  // When the section becomes hidden every child field must drop out of
  // RHF state so its (possibly default) value does not ship. On re-show,
  // SectionRenderer re-mounts the children which re-call register() and
  // start fresh — intentional: stale user input from the prior session
  // would otherwise ship in the payload and fail backend cross-mode validation.
  useEffect(() => {
    if (isHidden) {
      for (const name of sectionFieldNames) {
        unregister(name);
      }
    }
  }, [isHidden, sectionFieldNames, unregister]);

  return { isHidden };
}
