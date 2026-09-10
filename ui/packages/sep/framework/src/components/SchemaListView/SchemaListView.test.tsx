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
import type { ListView } from '@sep/api';

const { useScheduledTasksForPluginMock } = vi.hoisted(() => ({
  useScheduledTasksForPluginMock: vi.fn(),
}));

// Mock only the schedule hook; pull the real period/select helpers from their
// leaf module so the by-name grouping and pick mirror production. Importing the
// leaf directly (instead of `importOriginal` on the barrel) keeps the mock
// order-independent when the whole suite runs.
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

import {
  SchemaListView,
  type RenderListColumnOverride,
} from './SchemaListView';
import type { PeriodicTaskResponse } from '../ScheduledTasksPanel';

const listView: ListView = {
  columns: [
    { key: 'name', label: 'Name' },
    { key: 'status', label: 'Status', format: 'status' },
  ],
};

const rows = [
  { id: 1, name: 'alpha', status: 'completed' },
  { id: 2, name: 'beta', status: 'failed' },
];

describe('SchemaListView — renderListColumn override', () => {
  it('renders the override for a matching column and falls back to formatCellValue otherwise', () => {
    const renderListColumn: RenderListColumnOverride = ({
      columnKey,
      value,
      row,
    }) =>
      columnKey === 'status' ? (
        <span data-testid={`custom-status-${row.id}`}>
          custom:{String(value)}
        </span>
      ) : undefined;

    render(
      <SchemaListView
        listView={listView}
        data={rows}
        renderListColumn={renderListColumn}
      />
    );

    // status column → override
    expect(screen.getByTestId('custom-status-1')).toHaveTextContent(
      'custom:completed'
    );
    expect(screen.getByTestId('custom-status-2')).toHaveTextContent(
      'custom:failed'
    );
    // name column → no override returned (undefined) → default formatCellValue (plain text)
    expect(screen.getByText('alpha')).toBeInTheDocument();
    expect(screen.getByText('beta')).toBeInTheDocument();
  });

  it('renders default formatCellValue for every column when no override is supplied', () => {
    render(<SchemaListView listView={listView} data={rows} />);
    expect(screen.getByText('alpha')).toBeInTheDocument();
    // status format renders a chip with the raw label text
    expect(screen.getByText('completed')).toBeInTheDocument();
    expect(screen.queryByTestId('custom-status-1')).toBeNull();
  });

  it('never routes the actions column through the override', () => {
    const actionsListView: ListView = {
      columns: [
        { key: 'name', label: 'Name' },
        { key: 'actions', label: '', format: 'actions' },
      ],
    };
    const renderListColumn = vi.fn<RenderListColumnOverride>(() => (
      <span>x</span>
    ));
    render(
      <SchemaListView
        listView={actionsListView}
        data={rows}
        onDeleteRow={() => {}}
        renderListColumn={renderListColumn}
      />
    );
    // override invoked only for the non-actions `name` column, never `actions`
    expect(renderListColumn).not.toHaveBeenCalledWith(
      expect.objectContaining({ columnKey: 'actions' })
    );
    // delete control still rendered by the bespoke actions branch
    expect(screen.getAllByLabelText('Delete').length).toBe(rows.length);
  });

  it('renders the em dash for a row missing a declared column key', () => {
    const datedListView: ListView = {
      columns: [
        { key: 'name', label: 'Name' },
        { key: 'created_at', label: 'Created', format: 'date' },
        { key: 'seen_at', label: 'Seen', format: 'relative' },
        { key: 'kind', label: 'Kind', format: 'chip' },
      ],
    };
    // A server-supplied schema can declare a column the row omits, which used
    // to render as the literal 'undefined' / 'Invalid Date' / 'NaNd ago'.
    render(<SchemaListView listView={datedListView} data={[{ id: 1 }]} />);

    expect(screen.queryByText('undefined')).toBeNull();
    expect(screen.queryByText('Invalid Date')).toBeNull();
    // Em dash for the plain and chip columns; the two time-based columns stay
    // empty rather than showing a placeholder for a time that never happened.
    expect(screen.getAllByText('—').length).toBe(2);
  });
});

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

const scheduleListView: ListView = {
  columns: [
    { key: 'name', label: 'Name' },
    { key: 'schedule', label: 'Schedule', format: 'schedule' },
  ],
};

describe('SchemaListView — schedule-column glue', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW);
    useScheduledTasksForPluginMock.mockReset();
    useScheduledTasksForPluginMock.mockReturnValue({
      periodicTasks: [],
      isLoading: false,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('fetches the plugin schedule and joins it by trimmed task name into the cell', () => {
    useScheduledTasksForPluginMock.mockReturnValue({
      periodicTasks: [makePeriodic({ task: 'alpha' })],
      isLoading: false,
    });

    render(
      <SchemaListView
        listView={scheduleListView}
        // Stray whitespace on the row name must still join: the cell trims
        // before looking up, matching how the detail summary derives its key.
        data={[
          { id: 1, name: ' alpha ' },
          { id: 2, name: 'beta' },
        ]}
        pluginName="archives"
        disableSchedulePolling
      />
    );

    // fetch is scoped to the owning plugin
    expect(useScheduledTasksForPluginMock).toHaveBeenCalledWith('archives', {
      disablePolling: true,
    });
    // alpha row matched its schedule; beta row has none → unscheduled chip
    expect(screen.getByTestId('schedule-cell')).toBeInTheDocument();
    expect(screen.getByTestId('schedule-cell-unscheduled')).toBeInTheDocument();
  });

  it('groups several schedules per task name and renders the selected one', () => {
    useScheduledTasksForPluginMock.mockReturnValue({
      periodicTasks: [
        makePeriodic({
          id: 1,
          task: 'alpha',
          next_run_at: '2026-06-18T20:00:00Z',
          interval: { every: 1, period: 'days' },
          period: 'every 1 days',
        }),
        makePeriodic({
          id: 2,
          task: 'alpha',
          next_run_at: '2026-06-18T13:00:00Z',
          interval: { every: 1, period: 'hours' },
          period: 'every 1 hours',
        }),
      ],
      isLoading: false,
    });

    render(
      <SchemaListView
        listView={scheduleListView}
        data={[{ id: 1, name: 'alpha' }]}
        pluginName="archives"
        disableSchedulePolling
      />
    );

    // selectSchedule picks the soonest upcoming run for the grouped task
    expect(screen.getByTestId('schedule-cell-next-run')).toHaveTextContent(
      'in 1 hour'
    );
    expect(screen.getByLabelText('Recurs every 1 hours')).toBeInTheDocument();
  });

  it('issues no schedule fetch when the list view has no schedule column', () => {
    render(
      <SchemaListView listView={listView} data={rows} pluginName="archives" />
    );
    expect(useScheduledTasksForPluginMock).not.toHaveBeenCalled();
  });

  it('issues no schedule fetch when a schedule column exists but no plugin name is given', () => {
    render(<SchemaListView listView={scheduleListView} data={rows} />);
    expect(useScheduledTasksForPluginMock).not.toHaveBeenCalled();
  });
});
