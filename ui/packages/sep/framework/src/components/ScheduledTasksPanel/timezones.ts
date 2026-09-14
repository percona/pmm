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

import type { PeriodicTaskResponse } from './hooks';

/**
 * Two different zones are in play on a schedules screen, and conflating them is
 * the mistake this module exists to prevent:
 *
 * - The zone a schedule *fires* in — {@link scheduleTimezone}. The backend owns
 *   it and reports it on every schedule.
 * - The zone timestamps are *rendered* in — {@link browserTimezone}.
 *   `formatAbsoluteTime` goes through `toLocaleString()`, so every absolute time
 *   on the screen is the reader's own zone whatever the schedule was written in.
 *
 * A cron schedule written in `Europe/Lisbon` and read from a browser in
 * `America/New_York` shows both, and neither number is wrong — but only if the
 * screen says which is which.
 */

/** Every IANA zone this runtime can offer the cron timezone picker. */
export const TIMEZONES: string[] = (() => {
  type IntlWithTz = typeof Intl & {
    supportedValuesOf?: (key: string) => string[];
  };
  const intl = Intl as IntlWithTz;
  const supported = (() => {
    if (typeof intl.supportedValuesOf === 'function') {
      try {
        return intl.supportedValuesOf('timeZone');
      } catch {
        return [];
      }
    }
    return [];
  })();
  // `supportedValuesOf` reports canonical zones only, so it carries `Etc/UTC`
  // but not the bare `UTC` link. `UTC` is the backend's default for a crontab
  // and the zone it reports for every interval schedule, so without this the
  // picker holds a value it cannot offer: editing any UTC cron schedule leaves
  // the Autocomplete with an off-list value.
  return supported.includes('UTC') ? supported : ['UTC', ...supported];
})();

/**
 * The reader's own zone — what every absolute timestamp on the screen is
 * rendered in. Reported as resolved, not constrained to {@link TIMEZONES}:
 * `toLocaleString()` uses the real zone whether or not the picker can offer it,
 * so narrowing here would name a zone the screen is not actually using.
 */
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
  } catch {
    return 'UTC';
  }
}

/**
 * The cron picker's default. Same zone as {@link browserTimezone} whenever it
 * is selectable, and UTC otherwise — an option the picker cannot list is an
 * option the user could never restore after changing it.
 */
export function defaultPickerTimezone(): string {
  const tz = browserTimezone();
  return TIMEZONES.includes(tz) ? tz : 'UTC';
}

/**
 * The zone a schedule actually fires in.
 *
 * Prefers the backend's own computed `timezone` (PMM-15480), which is the
 * crontab's zone for a cron schedule and `UTC` for an interval one — an
 * interval's cadence is an absolute `timedelta` with no wall-clock anchor, and
 * `start_time` is stored as UTC, so no other zone could apply to it.
 *
 * Falls back to the crontab's own zone, then UTC, so a build talking to a
 * side-car that predates that field still states something true.
 */
export function scheduleTimezone(task: PeriodicTaskResponse): string {
  return task.timezone || task.crontab?.timezone || 'UTC';
}

/** The zone an interval schedule fires in. Fixed by the backend; never asked. */
export const INTERVAL_TIMEZONE = 'UTC';

const pad = (n: number) => String(n).padStart(2, '0');

/**
 * Format a UTC ISO instant for a `datetime-local` input **without shifting it
 * into the reader's zone**.
 *
 * `datetime-local` carries no zone, so the value shown is whatever wall clock
 * is written into it. Writing the instant's UTC parts means the field reads
 * back the same numbers the backend stores — which is only honest as long as
 * the field is labelled UTC (PMM-15454). The reverse is
 * {@link utcInputToIso}.
 */
export function utcIsoToUtcInput(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return '';
  }
  return `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}T${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}`;
}

/**
 * Read a `datetime-local` value back as a UTC instant, treating what was typed
 * as UTC wall clock. Inverse of {@link utcIsoToUtcInput}; returns `null` for a
 * blank or unparseable field so the caller can send "no start time".
 */
export function utcInputToIso(value: string): string | null {
  if (!value) {
    return null;
  }
  // `Date.parse` reads a bare `YYYY-MM-DDTHH:mm` as *local* time; the trailing
  // `Z` is what pins it to UTC. Seconds are optional in the input's value, so
  // append them only when absent.
  const withSeconds = value.length === 16 ? `${value}:00` : value;
  const d = new Date(`${withSeconds}Z`);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}
