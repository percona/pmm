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

/**
 * `datetime-local` reads/writes as local wall-clock with no timezone.
 * Backend datetime values are UTC ISO. Format the UTC instant in the browser's
 * local zone for display; parse the local input back through `Date` (which
 * interprets it as local) before serializing to UTC.
 */

/** Wall-clock string already suitable for a `datetime-local` input. */
const DATETIME_LOCAL_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?$/;

/**
 * Format a UTC ISO instant as a `datetime-local` value in the browser's zone.
 * Invalid / empty input yields `''`.
 */
export function utcIsoToLocalInput(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return '';
  }
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/**
 * Coerce any schema/API datetime into a `datetime-local` value.
 *
 * Bare local strings keep their wall-clock (truncated to minute precision); ISO
 * with a timezone converts to the browser's local zone; anything else becomes
 * `''`.
 */
export function toDatetimeLocalValue(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }
  if (typeof value !== 'string') {
    return '';
  }
  const trimmed = value.trim();
  if (trimmed === '') {
    return '';
  }
  if (DATETIME_LOCAL_RE.test(trimmed)) {
    return trimmed.slice(0, 16);
  }
  return utcIsoToLocalInput(trimmed);
}

/**
 * Parse a `datetime-local` wall-clock string as local time and return UTC ISO.
 * Empty / invalid → `undefined` so optional fields serialise as absent.
 */
export function localInputToUtcIso(local: string): string | undefined {
  const trimmed = local.trim();
  if (trimmed === '') {
    return undefined;
  }
  const d = new Date(trimmed);
  if (Number.isNaN(d.getTime())) {
    return undefined;
  }
  return d.toISOString();
}

/**
 * Coerce a form value back to UTC ISO for the backend.
 * Non-strings and empty values become `undefined`.
 */
export function fromDatetimeLocalValue(value: unknown): string | undefined {
  if (value === null || value === undefined) {
    return undefined;
  }
  if (typeof value !== 'string') {
    return undefined;
  }
  return localInputToUtcIso(value);
}
