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

import { useCallback, useMemo, useState } from 'react';
import { actionErrorMessage } from './actionErrorMessage';

export interface ActionErrorState {
  /** The last reported failure, or `null` once cleared / never failed. */
  error: unknown;
  /** {@link actionErrorMessage} of `error`, or `null` while there is none. */
  message: string | null;
  /** Record a failure. Pass the raw mutation error, not a rewritten string. */
  reportError: (error: unknown) => void;
  /** Drop the current failure (new attempt started, or the alert dismissed). */
  clearError: () => void;
}

/**
 * Hold one action failure at the level that renders it.
 *
 * Reach for this when the mutation's own `error` is not usable where the
 * message has to appear — the action was fired from a dialog that closes on
 * confirm, or from a child component that is unmounted by the refetch. Where
 * the mutation object is in scope at the render site, pass `mutation.error`
 * straight to {@link ActionErrorAlert} instead of duplicating it into state.
 */
export function useActionError(fallback?: string): ActionErrorState {
  const [error, setError] = useState<unknown>(null);

  // A nullish failure still has to render something — the point of the hook is
  // that a reported failure is never silent — so it becomes an empty Error and
  // falls through to the fallback message.
  const reportError = useCallback(
    (next: unknown) => setError(next ?? new Error('')),
    []
  );
  const clearError = useCallback(() => setError(null), []);

  const message = useMemo(
    () =>
      error === null || error === undefined
        ? null
        : actionErrorMessage(error, fallback),
    [error, fallback]
  );

  return { error, message, reportError, clearError };
}
