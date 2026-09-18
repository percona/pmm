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
import type { PluginField } from '../types';
import {
  evaluatePredicate,
  getGateFieldNames,
} from '../utils/predicateEvaluator';
import { watchValuesByName } from '../utils/watchValuesByName';

export interface ConditionalFieldState {
  isHidden: boolean;
  isRequired: boolean;
}

/**
 * Evaluate a field's `requires` and `forbidden` gates against current form
 * values and return the resolved visibility + required state.
 *
 * When `isHidden` becomes true the hook unregisters the field from
 * react-hook-form so its stale value is excluded from submission.
 */
export function useConditionalField(field: PluginField): ConditionalFieldState {
  const { control, unregister } = useFormContext();

  const watchedNames = useMemo<string[]>(() => {
    const requires = field.requires ?? [];
    const forbidden = field.forbidden ?? [];
    return getGateFieldNames([...requires, ...forbidden]);
    // field schema is static for the form's lifetime; dep on field is safe.
  }, [field]);

  // disabled=true when there are no gate fields — avoids subscribing to the
  // whole form when useWatch receives an empty name array.
  const rawValues = useWatch({
    control,
    name: watchedNames,
    disabled: watchedNames.length === 0,
  }) as unknown[];

  const isHidden = useMemo(() => {
    if (!field.forbidden?.length) {
      return false;
    }
    const map = watchValuesByName(watchedNames, rawValues);
    return field.forbidden.some((gate) => evaluatePredicate(gate.when, map));
  }, [rawValues, field, watchedNames]);

  const isRequired = useMemo(() => {
    if (field.required) {
      return true;
    }
    if (!field.requires?.length) {
      return false;
    }
    const map = watchValuesByName(watchedNames, rawValues);
    return field.requires.some((gate) => evaluatePredicate(gate.when, map));
  }, [rawValues, field, watchedNames]);

  // When a field becomes hidden its value must not appear in the submission
  // payload. Unregistering removes it from RHF state immediately.
  // On re-show, ConditionalFieldSlot remounts FieldRenderer, which re-calls
  // register() and the field starts fresh — intentional: the hidden value
  // is stale and should not silently resubmit.
  useEffect(() => {
    if (isHidden) {
      unregister(field.name);
    }
  }, [isHidden, field.name, unregister]);

  return { isHidden, isRequired };
}
