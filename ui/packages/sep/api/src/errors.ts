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

import axios, {
  AxiosError,
  type AxiosRequestConfig,
  type AxiosResponse,
} from 'axios';

export type ApiErrorKind =
  | 'network'
  | 'timeout'
  | 'canceled'
  | 'http'
  | 'unknown';

export interface ApiErrorDetails {
  kind: ApiErrorKind;
  status?: number;
  code?: string;
  message: string;
  url?: string;
  method?: string;
  data?: unknown;
}

/**
 * Structured error surfaced to UI consumers. Wraps the original AxiosError
 * so detailed inspection is still possible, but exposes a stable shape that
 * components can rely on without knowing about axios internals.
 */
export class ApiError extends Error implements ApiErrorDetails {
  readonly kind: ApiErrorKind;
  readonly status?: number;
  readonly code?: string;
  readonly url?: string;
  readonly method?: string;
  readonly data?: unknown;
  readonly original?: unknown;

  constructor(details: ApiErrorDetails, original?: unknown) {
    super(details.message);
    this.name = 'ApiError';
    this.kind = details.kind;
    this.status = details.status;
    this.code = details.code;
    this.url = details.url;
    this.method = details.method;
    this.data = details.data;
    this.original = original;
  }
}

function messageFromPayload(payload: unknown, fallback: string): string {
  if (payload && typeof payload === 'object') {
    const obj = payload as Record<string, unknown>;
    const detail = obj.detail ?? obj.message ?? obj.error;
    if (typeof detail === 'string') {
      return detail;
    }
  }
  return fallback;
}

/** A single backend per-field validation failure, ready to map onto a form field. */
export interface FieldValidationError {
  /** Dot-joined field path (react-hook-form name). Empty string for form-level errors. */
  path: string;
  /** Human-readable message from the backend (`detail[].msg`). */
  message: string;
}

/**
 * Parse a 422 payload (`ApiError.data.detail[]`, a Pydantic ``errors()`` array)
 * into ``{ path, message }`` pairs.
 *
 * Each entry's ``loc`` tuple is joined into a dot path matching the form's
 * react-hook-form field names. FastAPI request-body validation prefixes ``loc``
 * with a ``"body"`` segment (e.g. ``["body", "limit"]``); manual
 * ``model_validate`` failures do not. The leading ``"body"`` is stripped so both
 * shapes resolve to the same field path. The ``"body"`` strip is positional, so
 * a top-level field literally named ``body`` is indistinguishable from FastAPI's
 * wrapper segment; plugin schemas do not use that name. Entries whose ``loc``
 * carries no field segment yield an empty ``path`` so callers can surface them
 * as form-level errors rather than dropping them.
 *
 * Returns an empty array when the payload is not a recognizable detail array.
 */
export function parseFieldErrors(error: unknown): FieldValidationError[] {
  const data = error instanceof ApiError ? error.data : error;
  const detail = (data as { detail?: unknown } | null | undefined)?.detail;
  if (!Array.isArray(detail)) {
    return [];
  }

  const result: FieldValidationError[] = [];
  for (const entry of detail) {
    if (!entry || typeof entry !== 'object') {
      continue;
    }
    const { loc, msg } = entry as { loc?: unknown; msg?: unknown };
    const message =
      typeof msg === 'string' && msg.length > 0 ? msg : 'Invalid value';
    let segments = Array.isArray(loc)
      ? loc.filter(
          (s): s is string | number =>
            typeof s === 'string' || typeof s === 'number'
        )
      : [];
    if (segments[0] === 'body') {
      segments = segments.slice(1);
    }
    result.push({ path: segments.map(String).join('.'), message });
  }
  return result;
}

export function normalizeAxiosError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }

  if (axios.isCancel(error)) {
    return new ApiError(
      { kind: 'canceled', message: 'Request canceled' },
      error
    );
  }

  if (error instanceof AxiosError) {
    const config = error.config as AxiosRequestConfig | undefined;
    const response = error.response as AxiosResponse | undefined;
    const url = config?.url;
    const method = config?.method?.toUpperCase();

    if (error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT') {
      return new ApiError(
        {
          kind: 'timeout',
          code: error.code,
          message: 'Request timed out',
          url,
          method,
        },
        error
      );
    }

    if (response) {
      const status = response.status;
      return new ApiError(
        {
          kind: 'http',
          status,
          code: error.code,
          message: messageFromPayload(response.data, `HTTP ${status}`),
          url,
          method,
          data: response.data,
        },
        error
      );
    }

    return new ApiError(
      {
        kind: 'network',
        code: error.code,
        message: error.message || 'Network error',
        url,
        method,
      },
      error
    );
  }

  const message = error instanceof Error ? error.message : 'Unknown error';
  return new ApiError({ kind: 'unknown', message }, error);
}
