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

import cronstrue from 'cronstrue';
import type { PeriodicTaskResponse } from './hooks';

// Both moved to `utils/formatTimestamp`, which is now the single implementation
// of either, and are re-exported from here because the scheduled-task
// surfaces, the schedule cell and two test files import them from this path.
export {
  formatAbsoluteTime,
  formatRelativeTime,
} from '../../utils/formatTimestamp';

/** Plain-language description of a periodic task's recurrence. */
export interface PeriodDescription {
  /** Human-readable recurrence (for example, "every 1 hours" or a cron phrase). */
  display: string;
  /** Optional raw detail (the cron expression, with timezone) shown on hover. */
  tooltip?: string;
}

/**
 * Describe a periodic task's recurrence in plain language.
 *
 * Crontab schedules are humanised via `cronstrue` with the raw expression (and
 * timezone, when present) kept as the tooltip; interval schedules render as
 * "every N period". Shared by the scheduled-tasks table, the list schedule
 * cell, and the detail schedule summary so the wording stays consistent.
 */
export function describePeriod(task: PeriodicTaskResponse): PeriodDescription {
  if (task.crontab) {
    const expr = `${task.crontab.minute} ${task.crontab.hour} ${task.crontab.day_of_month} ${task.crontab.month_of_year} ${task.crontab.day_of_week}`;
    try {
      const human = cronstrue.toString(expr);
      const text = human.charAt(0).toLowerCase() + human.slice(1);
      const tz = task.crontab.timezone;
      return {
        display: text,
        tooltip: tz ? `${expr} (${tz})` : expr,
      };
    } catch {
      return { display: expr, tooltip: 'Invalid cron expression' };
    }
  }
  if (task.interval) {
    return { display: `every ${task.interval.every} ${task.interval.period}` };
  }
  return { display: task.period || '—' };
}

/**
 * Pick the single schedule to surface for a task that may own several periodic
 * schedules (the backend does not constrain a `task` to one schedule). Prefers
 * the soonest upcoming run — the earliest `next_run_at` — and falls back to the
 * first candidate so a disabled-only task (no `next_run_at`) still shows
 * something. Shared by the list cell and the detail summary so both always
 * select the same schedule for a given task.
 */
export function selectSchedule(
  candidates: PeriodicTaskResponse[]
): PeriodicTaskResponse | undefined {
  if (candidates.length <= 1) {
    return candidates[0];
  }
  const withNextRun = candidates.filter(
    (t): t is PeriodicTaskResponse & { next_run_at: string } =>
      Boolean(t.next_run_at)
  );
  const soonest = [...withNextRun].sort(
    (a, b) =>
      new Date(a.next_run_at).getTime() - new Date(b.next_run_at).getTime()
  )[0];
  return soonest ?? candidates[0];
}
