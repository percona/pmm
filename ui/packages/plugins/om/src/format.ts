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
 * Formatting helpers, on `date-fns` where `date-fns` has the answer.
 *
 * It is the monorepo's date library and `apps/pmm` already uses it, so parsing,
 * validity and differences come from there rather than from arithmetic kept here.
 *
 * `formatCompactDuration` is the exception, and deliberately not date-fns'
 * `formatCompactDuration`. Every caller is a table cell or a chip, where the string has to be
 * `2d 3h` rather than `2 days 3 hours`; date-fns emits words, and the short form needs
 * a custom `locale.formatDistance` - more code than the arithmetic it would replace. It
 * is named apart from date-fns' export of the same name so a file can import both.
 */

import { differenceInMilliseconds, format, isValid, parseISO } from 'date-fns';

/** Placeholder for a value that has no usable timestamp behind it. */
const NO_VALUE = '—';

/** How a timestamp is shown when it is shown absolutely. */
const TIMESTAMP_FORMAT = 'yyyy-MM-dd HH:mm:ss';

/**
 * Parse a wire timestamp, or null when there is nothing usable.
 *
 * One place for the two failure modes every helper here shares: no value at all, and a
 * value that will not parse. `parseISO` answers an Invalid Date rather than throwing,
 * so `isValid` is what separates them.
 */
function parse(iso: string | null | undefined): Date | null {
  if (!iso) {
    return null;
  }
  const date = parseISO(iso);
  return isValid(date) ? date : null;
}

/**
 * Format a duration in seconds as a compact `2d 3h` / `4m 12s` string.
 *
 * Two units, never more: these land in table cells beside other numbers, and
 * `2d 3h 7m 12s` reads as noise where `2d 3h` reads at a glance.
 */
export function formatCompactDuration(
  seconds: number | null | undefined
): string {
  if (seconds == null || !Number.isFinite(seconds) || seconds < 0) {
    return '';
  }
  const rounded = Math.floor(seconds);
  if (rounded < 60) {
    return `${rounded}s`;
  }
  const units: [number, string][] = [
    [86400, 'd'],
    [3600, 'h'],
    [60, 'm'],
    [1, 's'],
  ];
  const parts: string[] = [];
  let left = rounded;
  for (const [size, suffix] of units) {
    const value = Math.floor(left / size);
    if (value > 0) {
      parts.push(`${value}${suffix}`);
      left -= value * size;
    }
    if (parts.length === 2) {
      break;
    }
  }
  return parts.join(' ');
}

/** Absolute local timestamp, or an em-dash placeholder when there is none. */
export function formatTimestamp(iso: string | null | undefined): string {
  const date = parse(iso);
  return date ? format(date, TIMESTAMP_FORMAT) : NO_VALUE;
}

/**
 * Age of a timestamp as `3m ago`, relative to `now`.
 *
 * Deliberately not `formatDistanceToNowStrict`: that rounds to a single unit and
 * answers "3 minutes ago", and these are columns where `1h 12m` and `1h 58m` have to
 * differ. The difference comes from date-fns; the rendering is the compact form above.
 *
 * `now` is injectable so tests do not depend on the clock.
 */
export function formatAge(
  iso: string | null | undefined,
  now: number = Date.now()
): string {
  const date = parse(iso);
  if (!date) {
    return NO_VALUE;
  }
  // Clamped rather than signed: a host whose clock runs ahead should read as "just
  // collected", not as a negative age.
  const seconds = Math.max(0, differenceInMilliseconds(now, date) / 1000);
  return `${formatCompactDuration(seconds) || '0s'} ago`;
}

/**
 * Wall-clock length of a run in seconds, or null while it is still going.
 *
 * Split out from formatRunDuration so a table can sort on the number: sorting the
 * formatted string puts "9s" after "10m".
 */
export function runDurationSeconds(
  startedAt: string,
  finishedAt: string | null | undefined
): number | null {
  const start = parse(startedAt);
  const end = parse(finishedAt);
  if (!start || !end) {
    return null;
  }
  const elapsed = differenceInMilliseconds(end, start);
  return elapsed < 0 ? null : elapsed / 1000;
}

/** Wall-clock length of a run; empty while it is still going. */
export function formatRunDuration(
  startedAt: string,
  finishedAt: string | null | undefined
): string {
  const seconds = runDurationSeconds(startedAt, finishedAt);
  if (seconds === null) {
    return '';
  }
  return formatCompactDuration(seconds) || '0s';
}
