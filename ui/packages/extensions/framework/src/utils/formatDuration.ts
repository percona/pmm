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

const SECONDS_PER_MINUTE = 60;
const SECONDS_PER_HOUR = 3600;

/**
 * Format a duration in seconds for display next to a task run.
 *
 * Buckets are chosen so the number stays readable at every scale a backup
 * reaches: sub-minute runs keep a decimal (a 2.4s check reads as work, `2s`
 * reads as a rounding artefact), minutes drop it, and anything past an hour
 * drops seconds entirely — `1h 47m` rather than the `107m 12s` an unbucketed
 * minutes format would print for the same run.
 *
 * Absent input renders as an em-dash: the wire leaves `duration` null for a
 * run that has not finished, and a `0s` there would read as an instant run
 * rather than a missing value.
 *
 * A negative input is clamped to zero. `started_at` comes from the server and
 * `Date.now()` from the browser, so a client whose clock runs behind can
 * subtract its way past the start of a run that is genuinely under way.
 */
export function formatDuration(seconds: number | null | undefined): string {
  if (seconds === null || seconds === undefined || Number.isNaN(seconds)) {
    return '—';
  }

  const safe = Math.max(0, seconds);
  // Bucket on the value as it will be rendered, not the raw one: 59.95s
  // renders as `60.0s`, which belongs in the minutes bucket as `1m 0s` rather
  // than being shown as a minute's worth of seconds.
  const rounded = Math.round(safe * 10) / 10;

  if (rounded < SECONDS_PER_MINUTE) {
    return `${rounded.toFixed(1)}s`;
  }

  const total = Math.round(rounded);

  if (total < SECONDS_PER_HOUR) {
    return `${Math.floor(total / SECONDS_PER_MINUTE)}m ${total % SECONDS_PER_MINUTE}s`;
  }

  const hours = Math.floor(total / SECONDS_PER_HOUR);
  const minutes = Math.floor((total % SECONDS_PER_HOUR) / SECONDS_PER_MINUTE);

  return `${hours}h ${minutes}m`;
}
