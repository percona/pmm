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

import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
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

vi.mock('@sep/api', () => ({
  apiClient: apiMock,
  usePluginTasks: (...args: unknown[]) => usePluginTasksMock(...args),
  useAuth: () => ({
    isAdmin: authMock.canMutate,
    canMutate: authMock.canMutate,
  }),
}));

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

    await waitFor(() => expect(apiMock.post).toHaveBeenCalledTimes(1));
    const [url, body] = apiMock.post.mock.calls[0];
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

    await waitFor(() => expect(apiMock.post).toHaveBeenCalledTimes(1));
    const [, body] = apiMock.post.mock.calls[0];
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

    expect(apiMock.post).not.toHaveBeenCalled();
  });

  it('rejects an empty interval-every value and does not POST', async () => {
    setup([]);
    renderPanel(<ScheduledTasksPanel pluginName="myplugin" />);
    const user = userEvent.setup();

    await user.click(await screen.findByTestId('scheduled-tasks-add'));
    const form = await screen.findByTestId('scheduled-task-form');

    await user.clear(within(form).getByTestId('sched-form-interval-every'));
    await user.click(within(form).getByRole('button', { name: /Create/i }));

    expect(apiMock.post).not.toHaveBeenCalled();
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

    await waitFor(() => expect(apiMock.post).toHaveBeenCalledTimes(1));
    const [, body] = apiMock.post.mock.calls[0];
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
});
