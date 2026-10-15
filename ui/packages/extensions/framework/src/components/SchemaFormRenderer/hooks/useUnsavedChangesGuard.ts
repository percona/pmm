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

import { useEffect } from 'react';
import { useFormContext } from 'react-hook-form';

/**
 * Computes the "dirty and not yet successfully submitted" guard predicate,
 * registers a beforeunload listener while guarded, and re-arms the guard
 * (clears isSubmitSuccessful) when the parent reports a server error after a
 * prior successful synchronous submit.
 *
 * Returns `isGuarded` so the caller can pass it to an in-app navigation blocker.
 * Must be called inside a `<FormProvider>`.
 */
export function useUnsavedChangesGuard(submitError?: string | null): boolean {
  const { formState, reset } = useFormContext<Record<string, unknown>>();
  const { isDirty, isSubmitSuccessful } = formState;
  const isGuarded = isDirty && !isSubmitSuccessful;

  // Re-arm: clear isSubmitSuccessful each time the parent's async mutation fails
  // after the synchronous onSubmit returned without throwing (RHF flips
  // isSubmitSuccessful=true before the async leg resolves). This fires whenever
  // isSubmitSuccessful transitions back to true while an error is present —
  // covering both the first failure and subsequent failures with the same error
  // string, since the dep on isSubmitSuccessful ensures re-execution.
  // Note: useForm always initialises isSubmitSuccessful=false, so this effect
  // cannot fire spuriously on mount.
  useEffect(() => {
    if (submitError && isSubmitSuccessful) {
      reset(undefined, {
        keepValues: true,
        keepDirty: true,
        keepIsSubmitted: false,
      });
    }
  }, [submitError, isSubmitSuccessful, reset]);

  // Native browser prompt for tab-close / window-close / refresh.
  // Registered while guarded and removed immediately when predicate goes false.
  useEffect(() => {
    if (!isGuarded) {
      return;
    }
    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = ''; // Safari + older Chromium still require returnValue
    };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [isGuarded]);

  return isGuarded;
}
