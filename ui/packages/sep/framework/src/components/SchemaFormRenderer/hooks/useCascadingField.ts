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

import { useEffect, useRef } from 'react';
import { useFormContext, useWatch } from 'react-hook-form';

interface UseCascadingFieldArgs {
  /** Name of the field owning the cascade (the downstream field). */
  fieldName: string;
  /** Name of the upstream field to observe, or undefined if this field has no dependency. */
  dependsOn?: string;
}

interface UseCascadingFieldResult<T> {
  /** Current value of the upstream field, or undefined if no dependency is declared. */
  upstreamValue: T | undefined;
  /** True once the upstream value is set (or if there is no dependency). */
  ready: boolean;
}

/**
 * Observe an upstream field and reset this field when the upstream changes.
 *
 * Domain selectors (schema -> service, table -> schema) need to clear their
 * current value when the parent changes so stale selections cannot submit.
 * Consumers still own option fetching; the hook only tracks the dependency.
 */
export function useCascadingField<T = unknown>({
  fieldName,
  dependsOn,
}: UseCascadingFieldArgs): UseCascadingFieldResult<T> {
  const { control, setValue } = useFormContext();
  const upstreamValue = useWatch({
    control,
    name: dependsOn ?? '',
    disabled: !dependsOn,
  }) as T | undefined;
  const previousRef = useRef<T | undefined>(
    dependsOn ? upstreamValue : undefined
  );
  const didMountRef = useRef(false);

  useEffect(() => {
    if (!dependsOn) {
      return;
    }
    if (!didMountRef.current) {
      didMountRef.current = true;
      previousRef.current = upstreamValue;
      return;
    }
    if (previousRef.current !== upstreamValue) {
      setValue(fieldName, undefined, {
        shouldValidate: false,
        shouldDirty: false,
      });
      previousRef.current = upstreamValue;
    }
  }, [upstreamValue, fieldName, dependsOn, setValue]);

  return {
    upstreamValue: dependsOn ? upstreamValue : undefined,
    ready: dependsOn
      ? upstreamValue !== undefined && upstreamValue !== null
      : true,
  };
}
