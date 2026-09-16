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
 * The form-state shape every date-time control in this framework uses: the
 * `YYYY-MM-DDTHH:mm` wall clock an `<input type="datetime-local">` produced
 * before these fields moved onto Peak UI.
 *
 * Keeping the shape is not inertia. Both callers mean a *wall clock with no
 * zone*: a schema `datetime` field submits the string verbatim, and a
 * schedule's start time is UTC wall clock that
 * `ScheduledTasksPanel/timezones.ts` reads back as UTC (PMM-15454). Handing
 * either an adapter `Date` would attach the reader's zone to a value that has
 * none.
 */
const WALL_CLOCK = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/;

const pad = (n: number) => String(n).padStart(2, '0');
const padYear = (n: number) => String(n).padStart(4, '0');

/**
 * Read a wall-clock string as the `Date` the picker should display.
 *
 * The parts are fed to the local-time `Date` constructor rather than
 * `Date.parse`, which reads a bare `YYYY-MM-DDTHH:mm` as local but the same
 * string with a `Z` as UTC — a distinction this value does not have. Because
 * the picker formats in local time too, the digits round-trip unchanged
 * whatever zone the reader is in.
 *
 * **One exception, and it is not fixable here.** A wall clock inside the
 * reader's own spring-forward gap has no local instant — 02:30 does not exist
 * on 2026-03-08 in `America/New_York` — so `Date` normalizes it forward and the
 * picker shows 03:30. Rendering it correctly would need a picker that formats
 * in a fixed zone, and `AdapterDateFns` reports `isTimezoneCompatible = false`;
 * the host app (PMM's `App.tsx`) is on that adapter for every picker it has.
 * The cost is bounded: this is display only, for one hour a year, and the form
 * value is untouched unless the reader edits the field — see the mount test in
 * `DateTimeInput.test.tsx` and `startTimeToSubmit`, which does not resend a
 * start time the reader never touched.
 */
export function wallClockToDate(formValue: unknown): Date | null {
  if (typeof formValue !== 'string') {
    return null;
  }
  const match = WALL_CLOCK.exec(formValue);
  if (!match) {
    return null;
  }
  const [, year, month, day, hours, minutes, seconds] = match;
  const parts = {
    year: Number(year),
    month: Number(month),
    day: Number(day),
    hours: Number(hours),
    minutes: Number(minutes),
    seconds: Number(seconds ?? '0'),
  };
  const date = new Date(
    parts.year,
    parts.month - 1,
    parts.day,
    parts.hours,
    parts.minutes,
    parts.seconds
  );
  // `Date` maps a year below 100 into the 1900s, so `0050-…` would silently
  // become 1950. Restate the year to mean what it says.
  if (parts.year < 100) {
    date.setFullYear(parts.year);
  }
  if (Number.isNaN(date.getTime())) {
    return null;
  }
  // The pattern checks digit widths, not ranges, so month 13 or day 32 parse
  // and `Date` rolls them into a different date. Reject rather than render a
  // value nobody wrote — except the hour, which legitimately moves when the
  // wall clock falls in the local spring-forward gap (see above).
  const rolledOver =
    date.getFullYear() !== parts.year ||
    date.getMonth() !== parts.month - 1 ||
    date.getDate() !== parts.day;
  return rolledOver ? null : date;
}

/**
 * Write the picker's `Date` back as a wall-clock string.
 *
 * Minute precision, matching the `datetime-local` field it replaced and
 * `utcIsoToUtcInput`, which also only ever emits minutes. The asymmetry with
 * `wallClockToDate` — which accepts a seconds component so a stored value
 * carrying one still renders — is deliberate: seconds are readable but not
 * writable, because nothing in these forms lets anyone choose one.
 *
 * Returns `''` rather than `null` for a cleared or half-typed value: the form
 * field is a string, so an empty one has to stay a string for `required` rules
 * and for equality checks against a stored value.
 */
export function dateToWallClock(date: Date | null): string {
  if (!date || Number.isNaN(date.getTime())) {
    return '';
  }
  return `${padYear(date.getFullYear())}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
