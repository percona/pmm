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
 * The one place a timestamp becomes text.
 *
 * Before this, five formats reached the screen across the app — a locale date,
 * a locale date-time, a hand-rolled `3d ago`, and raw ISO strings with and
 * without microseconds — so the same run could be read three ways depending on
 * which surface showed it. Every surface now routes through
 * {@link formatTimestamp}, and the relative / absolute helpers below are the
 * only implementations of either.
 */

const RELATIVE_DIVISIONS: {
  amount: number;
  unit: Intl.RelativeTimeFormatUnit;
}[] = [
  { amount: 60, unit: 'seconds' },
  { amount: 60, unit: 'minutes' },
  { amount: 24, unit: 'hours' },
  { amount: 7, unit: 'days' },
  { amount: 4.34524, unit: 'weeks' },
  { amount: 12, unit: 'months' },
  { amount: Number.POSITIVE_INFINITY, unit: 'years' },
];

/**
 * How far from now — in either direction — a timestamp may sit before
 * {@link formatTimestamp} stops rendering it as a relative time. The bound is
 * exclusive: exactly this far out already renders absolute.
 *
 * Relative time answers "how long until / since" well at the scale someone is
 * actually tracking: a run from this morning, a schedule firing tonight. Past
 * about a week it stops answering anything — `412 days ago` is a number the
 * reader has to convert back into a date before it means what the date would
 * have meant directly. Seven days is the boundary because it is where people
 * switch from counting to naming: "3 days ago" is a position in a week someone
 * still holds in their head, "5 weeks ago" is not.
 *
 * The window is symmetric on purpose. A next scheduled run is the one place
 * the relative form earns its keep in the future tense — "in 2 hours" answers
 * "is this about to fire", which is the question a schedule is read to answer
 * — and clamping it at the same distance keeps `in 14 months` from appearing
 * for a far-future start time.
 */
const RELATIVE_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;

/**
 * Format a timestamp as a signed relative time ("in 2 hours", "3 days ago").
 *
 * Returns the raw input unchanged when it is not a valid date, so a malformed
 * value from the wire is shown rather than silently rendered as
 * `Invalid Date`. `now` is injectable so tests stay deterministic.
 */
export function formatRelativeTime(
  value: string,
  now: number = Date.now()
): string {
  const target = new Date(value).getTime();
  if (Number.isNaN(target)) {
    return value;
  }
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
  let duration = (target - now) / 1000;
  for (const division of RELATIVE_DIVISIONS) {
    if (Math.abs(duration) < division.amount) {
      return rtf.format(Math.round(duration), division.unit);
    }
    duration /= division.amount;
  }
  return value;
}

/** Format a timestamp as a locale absolute date-time, or `—` when absent. */
export function formatAbsoluteTime(value: string | null | undefined): string {
  if (!value) {
    return '—';
  }
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString();
}

/** A timestamp rendered for display, with the full value to hang on `title`. */
export interface FormattedTimestamp {
  /** What the cell shows — relative for recent past times, absolute otherwise. */
  display: string;
  /**
   * The unabbreviated local date-time, for a `title` attribute. Always present
   * and always absolute, including when `display` is already absolute: a reader
   * hovering to check a date should never find the same string they hovered.
   */
  title: string;
}

/**
 * Render a timestamp as the app's single format: relative while it is within a
 * week of now in either direction, absolute beyond that, with the full local
 * date-time always available on hover.
 *
 * Returns `null` for an absent value rather than a placeholder: the callers
 * that want an em-dash apply their own, and a never-executed task's "Last
 * Executed" cell is better empty than filled with a mark that reads as data.
 *
 * `now` is injectable so tests stay deterministic.
 */
export function formatTimestamp(
  value: string | null | undefined,
  now: number = Date.now()
): FormattedTimestamp | null {
  if (value === null || value === undefined || value === '') {
    return null;
  }
  const target = new Date(value).getTime();
  if (Number.isNaN(target)) {
    // Not a date at all. Show the raw value on both, so a schema pointing a
    // date column at the wrong key is visible rather than blank.
    return { display: value, title: value };
  }
  const title = formatAbsoluteTime(value);
  const withinWindow = Math.abs(now - target) < RELATIVE_WINDOW_MS;
  return {
    display: withinWindow ? formatRelativeTime(value, now) : title,
    title,
  };
}
