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

/** Format a duration in seconds as a compact `2d 3h` / `4m 12s` string. */
export function formatDuration(seconds: number | null | undefined): string {
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
    // Two units is enough to read at a glance; `2d 3h` beats `2d 3h 7m 12s`.
    if (parts.length === 2) {
      break;
    }
  }
  return parts.join(' ');
}

/** Absolute local timestamp, or an em-dash placeholder when there is none. */
export function formatTimestamp(iso: string | null | undefined): string {
  if (!iso) {
    return '—';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }
  return date.toLocaleString();
}

/**
 * Age of a timestamp as `3m ago`, relative to `now`.
 *
 * `now` is injectable so tests do not depend on the clock.
 */
export function formatAge(
  iso: string | null | undefined,
  now: number = Date.now()
): string {
  if (!iso) {
    return '—';
  }
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) {
    return '—';
  }
  const seconds = Math.max(0, (now - then) / 1000);
  return `${formatDuration(seconds) || '0s'} ago`;
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
  if (!finishedAt) {
    return null;
  }
  const start = new Date(startedAt).getTime();
  const end = new Date(finishedAt).getTime();
  if (Number.isNaN(start) || Number.isNaN(end) || end < start) {
    return null;
  }
  return (end - start) / 1000;
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
  return formatDuration(seconds) || '0s';
}
