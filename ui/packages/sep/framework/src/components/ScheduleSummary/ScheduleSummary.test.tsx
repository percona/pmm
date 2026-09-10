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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';

const { useScheduledTasksForPluginMock } = vi.hoisted(() => ({
  useScheduledTasksForPluginMock: vi.fn(),
}));

// Mock only the schedule hook; pull the real period/time helpers from their
// leaf module so wording stays accurate. Importing the leaf directly (instead
// of `importOriginal` on the barrel) keeps the mock order-independent when the
// whole suite runs.
vi.mock('../ScheduledTasksPanel', async () => {
  const periods = await import('../ScheduledTasksPanel/periods');
  const lastRun = await import('../ScheduledTasksPanel/LastRunStatus');
  return {
    useScheduledTasksForPlugin: (...args: unknown[]) =>
      useScheduledTasksForPluginMock(...args),
    describePeriod: periods.describePeriod,
    formatRelativeTime: periods.formatRelativeTime,
    formatAbsoluteTime: periods.formatAbsoluteTime,
    selectSchedule: periods.selectSchedule,
    LastRunStatus: lastRun.LastRunStatus,
  };
});

import { ScheduleSummary } from './ScheduleSummary';
import type { PeriodicTaskResponse } from '../ScheduledTasksPanel';

const NOW = new Date('2026-06-18T12:00:00Z');

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
    interval: { every: 2, period: 'hours' },
    crontab: null,
    execute_request: null,
    period: 'every 2 hours',
    next_run_at: '2026-06-18T14:00:00Z',
    ...overrides,
  };
}

function setup(periodicTasks: PeriodicTaskResponse[], isLoading = false) {
  useScheduledTasksForPluginMock.mockReturnValue({ periodicTasks, isLoading });
}

function renderSummary(taskName = 'plugin-task') {
  return render(
    <MemoryRouter>
      <ScheduleSummary
        pluginName="archives"
        taskName={taskName}
        scheduleHref="/apps/archives/schedule"
        disablePolling
      />
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
  useScheduledTasksForPluginMock.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

describe('ScheduleSummary', () => {
  it('renders the next run and recurrence when the task is scheduled', () => {
    setup([makePeriodic()]);
    renderSummary();

    const scheduled = screen.getByTestId('schedule-summary-scheduled');
    expect(scheduled).toHaveTextContent('in 2 hours');
    expect(scheduled).toHaveTextContent('every 2 hours');
    expect(
      screen.queryByTestId('schedule-summary-unscheduled')
    ).not.toBeInTheDocument();
  });

  it('falls back to an em dash for the next run when a scheduled task has none', () => {
    setup([makePeriodic({ next_run_at: null })]);
    renderSummary();

    const scheduled = screen.getByTestId('schedule-summary-scheduled');
    expect(scheduled).toHaveTextContent('—');
    expect(scheduled).toHaveTextContent('every 2 hours');
  });

  it('picks the soonest upcoming schedule when a task has several', () => {
    setup([
      makePeriodic({
        id: 1,
        next_run_at: '2026-06-18T20:00:00Z',
        interval: { every: 1, period: 'days' },
        period: 'every 1 days',
      }),
      makePeriodic({
        id: 2,
        next_run_at: '2026-06-18T13:00:00Z',
        interval: { every: 1, period: 'hours' },
        period: 'every 1 hours',
      }),
    ]);
    renderSummary();

    const scheduled = screen.getByTestId('schedule-summary-scheduled');
    expect(scheduled).toHaveTextContent('in 1 hour');
    expect(scheduled).toHaveTextContent('every 1 hours');
  });

  it('renders a not-scheduled state with an add-a-schedule link', () => {
    setup([makePeriodic({ task: 'a-different-task' })]);
    renderSummary();

    expect(
      screen.getByTestId('schedule-summary-unscheduled')
    ).toHaveTextContent('Not scheduled');
    const link = screen.getByTestId('schedule-summary-add-link');
    expect(link).toHaveTextContent('Add a schedule');
    expect(link).toHaveAttribute('href', '/apps/archives/schedule');
  });
});
