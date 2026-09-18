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

import { describe, expect, it } from 'vitest';
import { describePeriod, formatRelativeTime, selectSchedule } from './periods';
import type { PeriodicTaskResponse } from './hooks';

const NOW = new Date('2026-06-18T12:00:00Z').getTime();

function makePeriodic(
  overrides: Partial<PeriodicTaskResponse> = {}
): PeriodicTaskResponse {
  return {
    id: 1,
    name: 'periodic-1',
    task: 'plugin-task',
    enabled: true,
    description: '',
    start_time: null,
    last_run_at: null,
    date_changed: null,
    total_run_count: 0,
    interval: { every: 1, period: 'hours' },
    crontab: null,
    execute_request: null,
    period: 'every 1 hours',
    next_run_at: null,
    ...overrides,
  };
}

describe('formatRelativeTime', () => {
  it('formats a future timestamp', () => {
    expect(formatRelativeTime('2026-06-18T14:00:00Z', NOW)).toBe('in 2 hours');
  });

  it('formats a past timestamp', () => {
    expect(formatRelativeTime('2026-06-15T12:00:00Z', NOW)).toBe('3 days ago');
  });

  it('returns the raw value for an invalid date', () => {
    expect(formatRelativeTime('not-a-date', NOW)).toBe('not-a-date');
  });
});

describe('describePeriod', () => {
  it('describes interval schedules', () => {
    expect(describePeriod(makePeriodic()).display).toBe('every 1 hours');
  });

  it('humanises cron schedules and keeps the raw expression as a tooltip', () => {
    const result = describePeriod(
      makePeriodic({
        interval: null,
        crontab: {
          minute: '0',
          hour: '9',
          day_of_month: '*',
          month_of_year: '*',
          day_of_week: '*',
          timezone: 'UTC',
        },
      })
    );
    expect(result.display).toContain('9:00');
    expect(result.tooltip).toBe('0 9 * * * (UTC)');
  });
});

describe('selectSchedule', () => {
  it('returns undefined when there are no candidates', () => {
    expect(selectSchedule([])).toBeUndefined();
  });

  it('returns the only candidate', () => {
    const only = makePeriodic({ id: 7 });
    expect(selectSchedule([only])).toBe(only);
  });

  it('prefers the soonest upcoming run when several share a task', () => {
    const later = makePeriodic({ id: 1, next_run_at: '2026-06-18T20:00:00Z' });
    const sooner = makePeriodic({ id: 2, next_run_at: '2026-06-18T13:00:00Z' });
    expect(selectSchedule([later, sooner])).toBe(sooner);
  });

  it('falls back to the first candidate when none have a next run', () => {
    const a = makePeriodic({ id: 1, next_run_at: null });
    const b = makePeriodic({ id: 2, next_run_at: null });
    expect(selectSchedule([a, b])).toBe(a);
  });
});
