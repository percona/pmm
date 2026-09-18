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
import { ScheduleCell } from './ScheduleCell';
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
    interval: { every: 1, period: 'hours' },
    crontab: null,
    execute_request: null,
    period: 'every 1 hours',
    next_run_at: '2026-06-18T14:00:00Z',
    ...overrides,
  };
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(NOW);
});

afterEach(() => {
  vi.useRealTimers();
});

describe('ScheduleCell', () => {
  it('renders a muted "Not scheduled" chip when the task has no schedule', () => {
    render(<ScheduleCell task={null} />);

    expect(screen.getByTestId('schedule-cell-unscheduled')).toHaveTextContent(
      'Not scheduled'
    );
    expect(screen.queryByTestId('schedule-cell')).not.toBeInTheDocument();
  });

  it('renders the relative next run and a periodicity icon when scheduled', () => {
    render(<ScheduleCell task={makePeriodic()} />);

    expect(screen.getByTestId('schedule-cell-next-run')).toHaveTextContent(
      'in 2 hours'
    );
    expect(screen.getByTestId('schedule-cell-periodicity')).toHaveAttribute(
      'aria-label',
      'Recurs every 1 hours'
    );
    expect(
      screen.queryByTestId('schedule-cell-unscheduled')
    ).not.toBeInTheDocument();
  });

  it('falls back to an em dash when a scheduled task has no next run', () => {
    render(<ScheduleCell task={makePeriodic({ next_run_at: null })} />);

    expect(screen.getByTestId('schedule-cell-next-run')).toHaveTextContent('—');
    expect(screen.getByTestId('schedule-cell-periodicity')).toBeInTheDocument();
  });

  it('shows a loading placeholder instead of "Not scheduled" while the schedule list loads', () => {
    render(<ScheduleCell task={null} isLoading />);

    expect(screen.getByTestId('schedule-cell-loading')).toBeInTheDocument();
    expect(
      screen.queryByTestId('schedule-cell-unscheduled')
    ).not.toBeInTheDocument();
  });

  it('still renders the schedule once matched even while the list reports loading', () => {
    render(<ScheduleCell task={makePeriodic()} isLoading />);

    expect(screen.getByTestId('schedule-cell-next-run')).toHaveTextContent(
      'in 2 hours'
    );
    expect(
      screen.queryByTestId('schedule-cell-loading')
    ).not.toBeInTheDocument();
  });
});
