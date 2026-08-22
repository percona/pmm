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

import { ApiError, parseFieldErrors } from '@sep/api';

/** Used only when neither the response body nor the error carries a message. */
export const DEFAULT_ACTION_ERROR_FALLBACK =
  'The action failed. Please try again.';

/**
 * Derive the message to show for a failed action, preferring the server's own
 * reason over anything invented here.
 *
 * Two backend shapes carry that reason. A string `detail` — the 403 refusal,
 * 409, and 4xx/5xx generally — is already lifted into `ApiError.message` by
 * `@sep/api`. A 422 instead carries `detail` as a per-field array, which that
 * lift skips (leaving the literal `HTTP 422`), so the field entries are read
 * here and joined; a caller rendering a form uses `mapSubmitError` instead,
 * which places the same entries on their fields.
 *
 * The array branch is gated on the 422 status rather than on the payload shape,
 * matching `mapSubmitError`: other statuses can carry an array `detail` of
 * their own (a batch endpoint listing per-item failures), and those already
 * have a usable `ApiError.message`.
 *
 * `fallback` is reached only when neither path yields text — an `Error` with an
 * empty message, or a thrown non-error value.
 */
export function actionErrorMessage(
  error: unknown,
  fallback: string = DEFAULT_ACTION_ERROR_FALLBACK
): string {
  if (error === null || error === undefined) {
    return fallback;
  }

  if (error instanceof ApiError) {
    if (error.status === 422) {
      const fieldErrors = parseFieldErrors(error);
      if (fieldErrors.length > 0) {
        return fieldErrors
          .map(({ path, message }) => (path ? `${path}: ${message}` : message))
          .join('; ');
      }
      // Neither a per-field array nor a string `detail`: what is left is the
      // `HTTP 422` `@sep/api` synthesizes when the body carries no reason,
      // which tells the user nothing the caller's own wording does not.
      if (error.message === `HTTP ${error.status}`) {
        return fallback;
      }
    }
    return error.message || fallback;
  }

  if (error instanceof Error) {
    return error.message || fallback;
  }

  if (typeof error === 'string' && error.trim()) {
    return error;
  }

  return fallback;
}
