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

import {
  ApiError,
  parseFieldErrors,
  type FieldValidationError,
} from '@sep/api';
import { flattenSectionFields } from '../SchemaFormRenderer';
import type { FormSection } from '../SchemaFormRenderer/types';

/** Error state shared by the SchemaDrivenPlugin create and edit forms. */
export interface SubmitErrorState {
  /** Persistent in-form banner text, or `null` to hide the banner. */
  submitError: string | null;
  /** Per-field errors applied by SchemaFormRenderer via `setError`. */
  fieldErrors: FieldValidationError[];
}

/** Empty state used to clear both banner and per-field errors before a submit. */
export const EMPTY_SUBMIT_ERROR: SubmitErrorState = {
  submitError: null,
  fieldErrors: [],
};

/**
 * Map a mutation error into the persistent banner + per-field error state.
 *
 * Only HTTP 422 responses produce a banner and per-field errors; every other
 * failure (network, timeout, 5xx, other HTTP errors) returns the empty state so
 * those paths keep their existing toast-only behavior.
 *
 * The banner lists every parsed message — labelling each by its form field when
 * the path resolves to a schema field — so errors that do not map onto a
 * rendered field (form-level, hidden conditional, or unknown) are still
 * surfaced rather than silently dropped. A 422 without a parseable detail array
 * still shows a generic banner.
 */
export function mapSubmitError(
  error: unknown,
  sections: FormSection[],
  fallbackMessage: string
): SubmitErrorState {
  if (!(error instanceof ApiError) || error.status !== 422) {
    return EMPTY_SUBMIT_ERROR;
  }

  const fieldErrors = parseFieldErrors(error);
  if (fieldErrors.length === 0) {
    return { submitError: fallbackMessage, fieldErrors: [] };
  }

  const labelByPath = new Map<string, string>();
  for (const field of flattenSectionFields(sections)) {
    labelByPath.set(field.name, field.label || field.name);
  }

  const lines = fieldErrors.map(({ path, message }) => {
    const label = path ? (labelByPath.get(path) ?? path) : '';
    return label ? `• ${label}: ${message}` : `• ${message}`;
  });

  return {
    submitError: [fallbackMessage, ...lines].join('\n'),
    fieldErrors,
  };
}
