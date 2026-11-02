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
 * `datetime-local` reads/writes as wall-clock with no timezone, and the form
 * submits that wall-clock unchanged: snippets read it in the executor host's
 * zone (PMM-15707). A zoned value coming in, such as a schema default, shows as
 * its UTC wall-clock, the same digits the backend passes to the snippet.
 */

/** Wall-clock string already suitable for a `datetime-local` input. */
const DATETIME_LOCAL_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?$/;

/**
 * Format a zoned ISO instant as a `datetime-local` value in UTC.
 * Invalid / empty input yields `''`.
 */
export function utcIsoToUtcInput(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return '';
  }
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

/**
 * Coerce any schema/API datetime into a `datetime-local` value.
 *
 * Bare wall-clock strings keep their digits (truncated to minute precision); ISO
 * with a timezone converts to UTC; anything else becomes `''`.
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
  return utcIsoToUtcInput(trimmed);
}
