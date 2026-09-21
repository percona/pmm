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
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';

const { apiMock, usePluginTasksMock, authMock } = vi.hoisted(() => ({
  apiMock: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  usePluginTasksMock: vi.fn(),
  /** Flipped per test to cover the read-only (non-admin) rendering. */
  authMock: { canMutate: true },
}));

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: apiMock,
  usePluginTasks: (...args: unknown[]) => usePluginTasksMock(...args),
  useAuth: () => ({
    isAdmin: authMock.canMutate,
    canMutate: authMock.canMutate,
  }),
}));
// `fetchAllPluginListPages` (used by `useScheduledTasksForPlugin`) calls the
// package-internal `apiClient` bound in `../client`, not the barrel export
// above, so it needs its own mock pointing at the same spy.
vi.mock('@sep/api/src/client', () => ({ apiClient: apiMock }));

import { DEFAULT_PLUGIN_LIST_LIMIT } from '@sep/api';
import { formatTimestamp } from '../../utils/formatTimestamp';
import { ScheduledTasksPanel } from './ScheduledTasksPanel';
import type { PeriodicTaskResponse } from './hooks';

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
    next_runs: [],
    timezone: 'UTC',
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

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function renderPanel(ui: ReactNode) {
  return render(
    <QueryClientProvider client={makeClient()}>{ui}</QueryClientProvider>
  );
}

beforeEach(() => {
  apiMock.get.mockReset();
  apiMock.post.mockReset();
  apiMock.put.mockReset();
  apiMock.delete.mockReset();
  usePluginTasksMock.mockReset();
  authMock.canMutate = true;
});

/**
 * The create POST, ignoring the schedule-preview POST that the form issues
 * while the user types. Both go through the same mocked client.
 */
function createCalls() {
  return apiMock.post.mock.calls.filter(
    ([url]) => !String(url).includes('schedule/preview')
  );
}

function setup(periodic: PeriodicTaskResponse[]) {
  usePluginTasksMock.mockReturnValue({
    data: {
      items: [{ name: 'plugin-task' }, { name: 'other-plugin-task' }],
      pagination: null,
    },
    isLoading: false,
    isError: false,
  });
  apiMock.get.mockResolvedValue({ data: periodic });
}

describe('ScheduledTasksPanel', () => {
  it('shows empty state when there are no scheduled tasks for the plugin', async () => {
    setup([]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    await waitFor(() => {
      expect(
        screen.getByText(/No scheduled tasks for myplugin/i)
      ).toBeInTheDocument();
    });
  });

  it('names the app rather than its registry key when given a display name', async () => {
    setup([]);
    // A nested app is mounted under a scoped key (`mysql_backups/restore`).
    renderPanel(
      <ScheduledTasksPanel
        pluginName="mysql_backups/restore"
        displayName="MySQL Restores"
      />
    );

    await waitFor(() => {
      expect(
        screen.getByText(/No scheduled tasks for MySQL Restores/i)
      ).toBeInTheDocument();
    });
    expect(screen.queryByText(/mysql_backups\/restore/)).toBeNull();
  });

  it('renders each plugin task with period and run count', async () => {
    setup([
      makePeriodic({ id: 1, task: 'plugin-task', total_run_count: 3 }),
      makePeriodic({ id: 2, task: 'foreign-task', total_run_count: 99 }),
    ]);

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    await waitFor(() => {
      expect(screen.getByTestId('scheduled-task-row-1')).toBeInTheDocument();
    });
    expect(
      screen.queryByTestId('scheduled-task-row-2')
    ).not.toBeInTheDocument();
    expect(screen.getByText('every 1 hours')).toBeInTheDocument();
    expect(screen.getByText('3')).toBeInTheDocument();
  });

  it('short-circuits the periodic fetch after one request for a bare array', async () => {
    setup([makePeriodic({ id: 1 })]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    await waitFor(() => {
      expect(screen.getByTestId('scheduled-task-row-1')).toBeInTheDocument();
    });
    expect(apiMock.get).toHaveBeenCalledTimes(1);
    expect(apiMock.get).toHaveBeenCalledWith('/sep/periodic-tasks/', {
      params: { offset: 0, limit: DEFAULT_PLUGIN_LIST_LIMIT },
    });
  });

  it('walks paginated periodic envelopes before filtering to the plugin', async () => {
    usePluginTasksMock.mockReturnValue({
      data: { items: [{ name: 'plugin-task' }], pagination: null },
      isLoading: false,
      isError: false,
    });
    apiMock.get
      .mockResolvedValueOnce({
        data: {
          items: [
            makePeriodic({ id: 1, task: 'plugin-task', name: 'page-one' }),
          ],
          total: 3,
          offset: 0,
          limit: 1,
        },
      })
      .mockResolvedValueOnce({
        data: {
          items: [
            makePeriodic({ id: 2, task: 'foreign-task', name: 'page-two' }),
          ],
          total: 3,
          offset: 1,
          limit: 1,
        },
      })
      .mockResolvedValueOnce({
        data: {
          items: [
            makePeriodic({ id: 3, task: 'plugin-task', name: 'page-three' }),
          ],
          total: 3,
          offset: 2,
          limit: 1,
        },
      });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    await waitFor(() => {
      expect(screen.getByTestId('scheduled-task-row-3')).toBeInTheDocument();
    });
    expect(screen.getByTestId('scheduled-task-row-1')).toBeInTheDocument();
    expect(
      screen.queryByTestId('scheduled-task-row-2')
    ).not.toBeInTheDocument();
    expect(apiMock.get).toHaveBeenCalledTimes(3);
  });

  it('toggles enabled via PUT when the switch is clicked', async () => {
    setup([makePeriodic({ id: 7, enabled: true })]);
    apiMock.put.mockResolvedValue({ data: { id: 7 } });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    const toggle = await screen.findByLabelText(/Enable plugin-task/i);
    await userEvent.click(toggle);

    await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
    expect(apiMock.put).toHaveBeenCalledWith(
      '/sep/periodic-tasks/7',
      expect.objectContaining({ enabled: false })
    );
  });

  it('opens delete confirmation and only fires DELETE on confirm', async () => {
    setup([makePeriodic({ id: 9 })]);
    apiMock.delete.mockResolvedValue({ data: null });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    const deleteBtn = await screen.findByTestId('scheduled-task-delete-9');
    await user.click(deleteBtn);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /Cancel/i }));
    expect(apiMock.delete).not.toHaveBeenCalled();

    await user.click(await screen.findByTestId('scheduled-task-delete-9'));
    await user.click(screen.getByRole('button', { name: /^Delete$/ }));

    await waitFor(() =>
      expect(apiMock.delete).toHaveBeenCalledWith('/sep/periodic-tasks/9')
    );
  });

  it('creates an interval task via POST when filling the create form', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 42 }) });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');

    const everyInput = within(form).getByTestId('sched-form-interval-every');
    await user.clear(everyInput);
    await user.type(everyInput, '5');

    await user.click(within(form).getByRole('button', { name: /Create/i }));

    await waitFor(() => expect(createCalls()).toHaveLength(1));
    const [url, body] = createCalls()[0];
    expect(url).toBe('/sep/periodic-tasks/plugin-task/');
    expect(body).toMatchObject({
      task: 'plugin-task',
      enabled: true,
      interval: { every: 5, period: 'hours' },
      crontab: null,
    });
  });

  it('switches the create form to cron mode and submits a crontab body', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 43 }) });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');

    await user.click(within(form).getByTestId('sched-form-toggle-mode'));

    const cronInput = within(form).getByTestId('sched-form-cron');
    await user.type(cronInput, '*/5 * * * *');

    expect(
      within(form).getByTestId('sched-form-cron-preview')
    ).toHaveTextContent(/every 5 minutes/i);

    await user.click(within(form).getByRole('button', { name: /Create/i }));

    await waitFor(() => expect(createCalls()).toHaveLength(1));
    const [, body] = createCalls()[0];
    expect(body.interval).toBeNull();
    expect(body.crontab).toMatchObject({
      minute: '*/5',
      hour: '*',
      day_of_month: '*',
      month_of_year: '*',
      day_of_week: '*',
    });
    expect(body.start_time).toBeNull();
  });

  it('rejects an invalid cron expression and does not POST', async () => {
    setup([]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');

    await user.click(within(form).getByTestId('sched-form-toggle-mode'));
    await user.type(within(form).getByTestId('sched-form-cron'), 'not-a-cron');
    await user.click(within(form).getByRole('button', { name: /Create/i }));

    expect(createCalls()).toHaveLength(0);
  });

  it('rejects an empty interval-every value and does not POST', async () => {
    setup([]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');

    await user.clear(within(form).getByTestId('sched-form-interval-every'));
    await user.click(within(form).getByRole('button', { name: /Create/i }));

    expect(createCalls()).toHaveLength(0);
  });

  it('disables the toggle while a previous toggle is in flight', async () => {
    setup([makePeriodic({ id: 7, enabled: true })]);
    let resolvePut: (v: unknown) => void = () => {};
    apiMock.put.mockReturnValue(
      new Promise((resolve) => {
        resolvePut = resolve;
      })
    );

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const toggle = await screen.findByLabelText(/Enable plugin-task/i);
    await userEvent.click(toggle);

    await waitFor(() => expect(toggle).toBeDisabled());
    resolvePut({ data: {} });
  });

  it('round-trips an existing cron task into the edit form and submits an updated crontab', async () => {
    setup([
      makePeriodic({
        id: 21,
        interval: null,
        crontab: {
          minute: '0',
          hour: '6',
          day_of_month: '*',
          month_of_year: '*',
          day_of_week: '*',
          timezone: 'UTC',
        },
        period: '0 6 * * *',
      }),
    ]);
    apiMock.put.mockResolvedValue({ data: { id: 21 } });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-task-edit-21'));
    const form = await screen.findByTestId('scheduled-task-form');

    const cronInput = within(form).getByTestId('sched-form-cron');
    expect(cronInput).toHaveValue('0 6 * * *');

    await user.clear(cronInput);
    await user.type(cronInput, '*/15 * * * *');
    await user.click(within(form).getByRole('button', { name: /Save/i }));

    await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
    const [url, body] = apiMock.put.mock.calls[0];
    expect(url).toBe('/sep/periodic-tasks/21');
    expect(body.interval).toBeNull();
    expect(body.crontab).toMatchObject({
      minute: '*/15',
      hour: '*',
      day_of_month: '*',
      month_of_year: '*',
      day_of_week: '*',
    });
  });

  it('submits chain_task_names in execute_request when a chain is configured', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 50 }) });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');

    const chainBuilder = within(form).getByTestId('chain-builder');
    await user.click(within(chainBuilder).getByRole('combobox'));
    const option = await screen.findByRole('option', {
      name: 'other-plugin-task',
    });
    await user.click(option);

    await user.click(within(form).getByRole('button', { name: /Create/i }));

    await waitFor(() => expect(createCalls()).toHaveLength(1));
    const [, body] = createCalls()[0];
    expect(body.execute_request).toMatchObject({
      chain_task_names: ['other-plugin-task'],
      chain_on_failure: false,
    });
  });

  it('shows a panel-level error when a toggle mutation fails', async () => {
    setup([makePeriodic({ id: 60, enabled: true })]);
    apiMock.put.mockRejectedValue(new Error('toggle blew up'));

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    const toggle = await screen.findByLabelText(/Enable plugin-task/i);
    await userEvent.click(toggle);

    const alert = await screen.findByTestId('scheduled-tasks-action-error');
    expect(alert).toHaveTextContent(/toggle blew up/i);
  });

  it('shows a panel-level error when a delete mutation fails', async () => {
    setup([makePeriodic({ id: 61 })]);
    apiMock.delete.mockRejectedValue(new Error('delete blew up'));

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-task-delete-61'));
    await user.click(screen.getByRole('button', { name: /^Delete$/ }));

    const alert = await screen.findByTestId('scheduled-tasks-action-error');
    expect(alert).toHaveTextContent(/delete blew up/i);
  });

  it('shows an inline form error when a create mutation fails', async () => {
    setup([]);
    apiMock.post.mockRejectedValue(new Error('create blew up'));

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');
    await user.click(within(form).getByRole('button', { name: /Create/i }));

    await waitFor(() => {
      expect(within(form).getByText(/create blew up/i)).toBeInTheDocument();
    });
  });

  it('submits an edit via PUT with the updated schedule', async () => {
    setup([makePeriodic({ id: 11 })]);
    apiMock.put.mockResolvedValue({ data: makePeriodic({ id: 11 }) });

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-task-edit-11'));
    const form = await screen.findByTestId('scheduled-task-form');

    const everyInput = within(form).getByTestId('sched-form-interval-every');
    await user.clear(everyInput);
    await user.type(everyInput, '10');
    await user.click(within(form).getByRole('button', { name: /Save/i }));

    await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
    const [url, body] = apiMock.put.mock.calls[0];
    expect(url).toBe('/sep/periodic-tasks/11');
    expect(body.interval).toMatchObject({ every: 10, period: 'hours' });
  });
});

describe('ScheduledTasksPanel — write access', () => {
  it('renders add, edit, delete and the enable toggle for a session that may mutate', async () => {
    setup([makePeriodic({ id: 1 })]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    await waitFor(() => {
      expect(screen.getByTestId('scheduled-task-row-1')).toBeInTheDocument();
    });
    expect(screen.getByTestId('scheduled-tasks-add')).toBeInTheDocument();
    expect(screen.getByTestId('scheduled-task-edit-1')).toBeInTheDocument();
    expect(screen.getByTestId('scheduled-task-delete-1')).toBeInTheDocument();
    expect(screen.getByLabelText(/Enable plugin-task/i)).toBeInTheDocument();
  });

  it('renders no add, edit, delete or enable toggle for a non-admin', async () => {
    authMock.canMutate = false;
    setup([makePeriodic({ id: 1 })]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    await waitFor(() => {
      expect(screen.getByTestId('scheduled-task-row-1')).toBeInTheDocument();
    });
    expect(screen.queryByTestId('scheduled-tasks-add')).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('scheduled-task-edit-1')
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId('scheduled-task-delete-1')
    ).not.toBeInTheDocument();
    expect(
      screen.queryByLabelText(/Enable plugin-task/i)
    ).not.toBeInTheDocument();
    // The schedule stays readable, enabled state included.
    expect(
      within(screen.getByTestId('scheduled-task-row-1')).getByText('Enabled')
    ).toBeInTheDocument();
  });

  it('opens the execution detail from the last-run cell', async () => {
    // The periodic-task API reports a last-run status but no task history id,
    // so the cell has to open the run by task name. It was inert before, which
    // left the schedules table as much of a dead end as the history row.
    setup([
      makePeriodic({
        id: 1,
        task: 'plugin-task',
        last_run_status: 'failed',
        last_run_at: '2026-09-07T02:00:00Z',
      }),
    ]);

    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

    const chip = await waitFor(() =>
      screen.getByRole('button', { name: /Failed/ })
    );
    await userEvent.click(chip);

    expect(
      await screen.findByRole('button', { name: 'Close run details' })
    ).toBeInTheDocument();
  });

  // PMM-15454: the screen has to say which zone it is talking about. Two
  // different zones are on it at once - the one a schedule fires in, and the
  // one its timestamps are rendered in - and neither number means anything
  // until the screen names them.
  describe('stating the timezone in force', () => {
    it('names the zone each schedule fires in, from the backend', async () => {
      setup([
        makePeriodic({
          id: 31,
          interval: null,
          timezone: 'Europe/Lisbon',
          crontab: {
            minute: '0',
            hour: '2',
            day_of_month: '*',
            month_of_year: '*',
            day_of_week: '*',
            timezone: 'Europe/Lisbon',
          },
          period: '0 2 * * *',
        }),
      ]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      expect(
        await screen.findByTestId('scheduled-task-timezone-31')
      ).toHaveTextContent('Europe/Lisbon');
    });

    it('states UTC for an interval schedule, which has no zone of its own', async () => {
      setup([makePeriodic({ id: 32, timezone: 'UTC' })]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      expect(
        await screen.findByTestId('scheduled-task-timezone-32')
      ).toHaveTextContent('UTC');
    });

    it('names the render zone on the timestamp itself, not only in the header', async () => {
      setup([
        makePeriodic({
          id: 34,
          timezone: 'Europe/Lisbon',
          next_run_at: '2026-03-01T02:30:00Z',
        }),
      ]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      const row = await screen.findByTestId('scheduled-task-row-34');
      // The row names Lisbon beside the recurrence; the timestamps next to it
      // are still the reader's zone, and have to say so where they are read.
      expect(
        within(row).getByTestId('scheduled-task-timezone-34')
      ).toHaveTextContent('Runs in Europe/Lisbon');

      const formatted = formatTimestamp('2026-03-01T02:30:00Z');
      expect(within(row).getByTitle(formatted!.title)).toBeInTheDocument();
      expect(formatted!.title).toContain(
        `(${Intl.DateTimeFormat().resolvedOptions().timeZone})`
      );
    });

    it('names the zone the table renders its timestamps in', async () => {
      setup([makePeriodic({ id: 33 })]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      const notice = await screen.findByTestId(
        'scheduled-tasks-display-timezone'
      );
      expect(notice).toHaveTextContent(
        `Times shown in ${Intl.DateTimeFormat().resolvedOptions().timeZone}`
      );
    });

    it('omits the display-zone notice when there is nothing to read', async () => {
      setup([]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      await screen.findByText(/No scheduled tasks for myplugin/i);
      expect(
        screen.queryByTestId('scheduled-tasks-display-timezone')
      ).not.toBeInTheDocument();
    });

    it('states UTC on the create form and labels the start-time field with it', async () => {
      setup([]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      expect(
        within(form).getByTestId('sched-form-timezone-notice')
      ).toHaveTextContent('Runs in UTC');
      expect(within(form).getByLabelText(/Start time \(UTC\)/i)).toBeVisible();
    });

    it('tracks the picked zone in cron mode', async () => {
      setup([]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      await user.click(within(form).getByTestId('sched-form-toggle-mode'));

      const picker = within(form).getByTestId('sched-form-timezone');
      await user.clear(picker);
      await user.type(picker, 'Europe/Lisbon');
      await user.click(
        await screen.findByRole('option', { name: 'Europe/Lisbon' })
      );

      await waitFor(() =>
        expect(
          within(form).getByTestId('sched-form-timezone-notice')
        ).toHaveTextContent('Runs in Europe/Lisbon')
      );
    });

    it('sends the start time as the UTC wall clock that was typed', async () => {
      setup([]);
      apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 60 }) });

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      fireEvent.change(within(form).getByTestId('sched-form-start-time'), {
        target: { value: '2026-03-01T02:30' },
      });
      await user.click(within(form).getByRole('button', { name: /Create/i }));

      await waitFor(() => expect(createCalls()).toHaveLength(1));
      const [, body] = createCalls()[0];
      // Not shifted by the runner's zone: what the field said is what is sent.
      expect(body.start_time).toBe('2026-03-01T02:30:00.000Z');
    });

    it('leaves a stored start time byte-identical when an unrelated field is edited', async () => {
      // The field carries minutes; the stored value carries seconds. Saving an
      // edit to the interval must not round the schedule's first fire down.
      setup([makePeriodic({ id: 62, start_time: '2026-03-01T02:30:45.123Z' })]);
      apiMock.put.mockResolvedValue({ data: { id: 62 } });

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();

      await user.click(await screen.findByTestId('scheduled-task-edit-62'));
      const form = await screen.findByTestId('scheduled-task-form');

      const every = within(form).getByTestId('sched-form-interval-every');
      await user.clear(every);
      await user.type(every, '6');
      await user.click(within(form).getByRole('button', { name: /Save/i }));

      await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
      const [, body] = apiMock.put.mock.calls[0];
      expect(body.interval).toMatchObject({ every: 6 });
      expect(body.start_time).toBe('2026-03-01T02:30:45.123Z');
    });

    it('sends the edited start time when the field itself is changed', async () => {
      setup([makePeriodic({ id: 63, start_time: '2026-03-01T02:30:45.123Z' })]);
      apiMock.put.mockResolvedValue({ data: { id: 63 } });

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();

      await user.click(await screen.findByTestId('scheduled-task-edit-63'));
      const form = await screen.findByTestId('scheduled-task-form');

      fireEvent.change(within(form).getByTestId('sched-form-start-time'), {
        target: { value: '2026-04-02T07:15' },
      });
      await user.click(within(form).getByRole('button', { name: /Save/i }));

      await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
      const [, body] = apiMock.put.mock.calls[0];
      expect(body.start_time).toBe('2026-04-02T07:15:00.000Z');
    });

    it('offers a stored zone the runtime does not list, so it can be restored', async () => {
      // The backend accepts aliases such as `US/Eastern` that
      // `Intl.supportedValuesOf` omits; the picker must still hold them.
      setup([
        makePeriodic({
          id: 64,
          interval: null,
          timezone: 'US/Eastern',
          crontab: {
            minute: '0',
            hour: '2',
            day_of_month: '*',
            month_of_year: '*',
            day_of_week: '*',
            timezone: 'US/Eastern',
          },
          period: '0 2 * * *',
        }),
      ]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();

      await user.click(await screen.findByTestId('scheduled-task-edit-64'));
      const form = await screen.findByTestId('scheduled-task-form');

      const picker = within(form).getByTestId('sched-form-timezone');
      expect(picker).toHaveValue('US/Eastern');
      expect(
        within(form).getByTestId('sched-form-timezone-notice')
      ).toHaveTextContent('Runs in US/Eastern');

      // Displaying the value is not enough: MUI shows an off-list value while
      // refusing to offer it, so changing zone would be a one-way door.
      await user.click(picker);
      expect(
        await screen.findByRole('option', { name: 'US/Eastern' })
      ).toBeInTheDocument();
    });

    it('round-trips a stored start time back into the field unshifted', async () => {
      setup([makePeriodic({ id: 61, start_time: '2026-03-01T02:30:00Z' })]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();

      await user.click(await screen.findByTestId('scheduled-task-edit-61'));
      const form = await screen.findByTestId('scheduled-task-form');

      expect(within(form).getByTestId('sched-form-start-time')).toHaveValue(
        '2026-03-01T02:30'
      );
    });
  });

  // PMM-15454: chaining is not wired up for these apps, so the column was an
  // unbroken run of em dashes advertising something the reader cannot use.
  describe('the Chain column', () => {
    it('is hidden while no schedule carries a chain', async () => {
      setup([
        makePeriodic({ id: 41 }),
        makePeriodic({ id: 42, task: 'plugin-task' }),
      ]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      await screen.findByTestId('scheduled-task-row-41');
      expect(
        screen.queryByRole('columnheader', { name: 'Chain' })
      ).not.toBeInTheDocument();
    });

    it('keeps the row width matching the header when it is hidden', async () => {
      setup([makePeriodic({ id: 43 })]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      const row = await screen.findByTestId('scheduled-task-row-43');
      expect(within(row).getAllByRole('cell')).toHaveLength(
        screen.getAllByRole('columnheader').length
      );
    });

    it('keeps the row width matching the header for a read-only session', async () => {
      authMock.canMutate = false;
      setup([makePeriodic({ id: 46 })]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      const row = await screen.findByTestId('scheduled-task-row-46');
      expect(within(row).getAllByRole('cell')).toHaveLength(
        screen.getAllByRole('columnheader').length
      );
    });

    it('returns as soon as one schedule carries a chain', async () => {
      setup([
        makePeriodic({ id: 44 }),
        makePeriodic({
          id: 45,
          execute_request: {
            meta: {},
            chain_task_names: ['other-plugin-task'],
            chain_on_failure: false,
          },
        }),
      ]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);

      expect(
        await screen.findByRole('columnheader', { name: 'Chain' })
      ).toBeInTheDocument();
      expect(screen.getByText('other-plugin-task')).toBeInTheDocument();
    });
  });

  // PMM-15454 / PMM-15480: the upcoming runs come from the scheduler's own
  // objects, so the form and the saved schedule cannot disagree about "next".
  describe('previewing the next runs', () => {
    function previewCalls() {
      return apiMock.post.mock.calls.filter(([url]) =>
        String(url).includes('schedule/preview')
      );
    }

    it('shows the next three runs the backend reports, as clock time in the schedule zone', async () => {
      setup([]);
      // Close enough to now that a relative renderer (formatTimestamp) would
      // show "in 8 hours" / "tomorrow" / "in 2 days" instead of a clock time.
      const hoursFromNow = (h: number) =>
        new Date(Date.now() + h * 60 * 60 * 1000).toISOString();
      const runIsos = [
        hoursFromNow(8),
        hoursFromNow(32),
        hoursFromNow(56),
        hoursFromNow(80),
      ];
      apiMock.post.mockResolvedValue({
        data: {
          timezone: 'Europe/Lisbon',
          next_run_at: runIsos[0],
          next_runs: runIsos,
        },
      });

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      const runs = await within(form).findByTestId(
        'sched-form-next-runs',
        {},
        { timeout: 3000 }
      );
      await waitFor(() =>
        expect(runs).toHaveTextContent(/Next runs \(Europe\/Lisbon\):/)
      );
      // Three, not the four the backend offered, rendered as a clock time in
      // the schedule's own zone rather than the reader's or a relative phrase.
      for (const iso of runIsos.slice(0, 3)) {
        expect(runs.textContent).toContain(
          new Date(iso).toLocaleString(undefined, {
            timeZone: 'Europe/Lisbon',
            dateStyle: 'medium',
            timeStyle: 'short',
          })
        );
      }
      expect(runs.textContent).not.toContain(
        new Date(runIsos[3]).toLocaleString(undefined, {
          timeZone: 'Europe/Lisbon',
          dateStyle: 'medium',
          timeStyle: 'short',
        })
      );
      expect(runs.textContent).not.toMatch(/in \d+ (hours?|days?)|tomorrow/);
    });

    describe('for a reader outside the schedule zone', () => {
      // In a UTC runner the panel header and an interval preview name the same
      // zone, so the disagreement these tests guard against never shows up.
      beforeEach(() => {
        vi.stubEnv('TZ', 'America/Sao_Paulo');
      });
      afterEach(() => {
        vi.unstubAllEnvs();
      });

      const utcClock = (iso: string) =>
        new Date(iso).toLocaleString(undefined, {
          timeZone: 'UTC',
          dateStyle: 'medium',
          timeStyle: 'short',
        });

      it('names the zone of an interval preview the header does not cover', async () => {
        setup([]);
        const runIsos = [
          '2026-09-20T06:13:00Z',
          '2026-09-20T07:13:00Z',
          '2026-09-20T08:13:00Z',
        ];
        apiMock.post.mockResolvedValue({
          data: {
            timezone: 'UTC',
            next_run_at: runIsos[0],
            next_runs: runIsos,
          },
        });

        renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
        const user = userEvent.setup();
        await user.click(await screen.findByTestId('scheduled-tasks-add'));
        const form = await screen.findByTestId('scheduled-task-form');

        expect(
          screen.getByTestId('scheduled-tasks-display-timezone')
        ).toHaveTextContent('Times shown in America/Sao_Paulo');

        const runs = await within(form).findByTestId(
          'sched-form-next-runs',
          {},
          { timeout: 3000 }
        );
        await waitFor(() => expect(runs).toHaveTextContent(/^Next runs/));
        expect(runs).toHaveTextContent(
          `Next runs (UTC): ${runIsos.map(utcClock).join(', ')}`
        );
      });

      it('names the zone of a cron preview written in another zone', async () => {
        setup([
          makePeriodic({
            id: 70,
            interval: null,
            timezone: 'Europe/Lisbon',
            crontab: {
              minute: '0',
              hour: '3',
              day_of_month: '*',
              month_of_year: '*',
              day_of_week: '*',
              timezone: 'Europe/Lisbon',
            },
            period: '0 3 * * *',
          }),
        ]);
        apiMock.post.mockResolvedValue({
          data: {
            timezone: 'Europe/Lisbon',
            next_run_at: '2026-09-21T02:00:00Z',
            next_runs: ['2026-09-21T02:00:00Z'],
          },
        });

        renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
        const user = userEvent.setup();
        await user.click(await screen.findByTestId('scheduled-task-edit-70'));
        const form = await screen.findByTestId('scheduled-task-form');

        await waitFor(
          () =>
            expect(
              within(form).getByTestId('sched-form-next-runs')
            ).toHaveTextContent('Next runs (Europe/Lisbon): '),
          { timeout: 3000 }
        );
      });
    });

    it('states the zone the backend resolved, not the one the form assumed', async () => {
      setup([]);
      apiMock.post.mockResolvedValue({
        data: {
          timezone: 'Europe/Lisbon',
          next_run_at: null,
          next_runs: [],
        },
      });

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      await waitFor(
        () =>
          expect(
            within(form).getByTestId('sched-form-timezone-notice')
          ).toHaveTextContent('Runs in Europe/Lisbon'),
        { timeout: 3000 }
      );
    });

    it('asks for no preview while the cron expression is invalid', async () => {
      setup([]);

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      await user.click(within(form).getByTestId('sched-form-toggle-mode'));
      // Interval mode previews on mount; only what cron mode asks for counts.
      apiMock.post.mockClear();
      await user.type(within(form).getByTestId('sched-form-cron'), 'nonsense');

      await waitFor(() => expect(previewCalls()).toHaveLength(0), {
        timeout: 1500,
      });
      expect(
        within(form).queryByTestId('sched-form-next-runs')
      ).not.toBeInTheDocument();
    });

    it('says so when the preview cannot be worked out', async () => {
      setup([]);
      apiMock.post.mockRejectedValue(new Error('422'));

      renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('scheduled-tasks-add'));
      const form = await screen.findByTestId('scheduled-task-form');

      await waitFor(
        () =>
          expect(
            within(form).getByTestId('sched-form-next-runs')
          ).toHaveTextContent(/Could not work out the next runs/),
        { timeout: 3000 }
      );
    });
  });
});
