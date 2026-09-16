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
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

const { apiMock, usePluginTasksMock } = vi.hoisted(() => ({
  apiMock: { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn() },
  usePluginTasksMock: vi.fn(),
}));

vi.mock('@sep/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@sep/api')>()),
  apiClient: apiMock,
  usePluginTasks: (...args: unknown[]) => usePluginTasksMock(...args),
  useAuth: () => ({ isAdmin: true, canMutate: true }),
}));
vi.mock('@sep/api/src/client', () => ({ apiClient: apiMock }));

import { PluginSchedulePage } from './PluginSchedulePage';
import type { PluginSchema } from '@sep/api';
import type { PeriodicTaskResponse } from '../ScheduledTasksPanel/hooks';

const schema = {
  name: 'myplugin',
  display_name: 'My Plugin',
  item_display_name: 'task',
  item_display_name_plural: 'tasks',
  forms: [],
  list_view: { columns: [] },
} as unknown as PluginSchema;

function makePeriodic(
  overrides: Partial<PeriodicTaskResponse> = {}
): PeriodicTaskResponse {
  return {
    id: 5,
    name: 'periodic-5',
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

function renderAt(path: string, periodic: PeriodicTaskResponse[]) {
  usePluginTasksMock.mockReturnValue({
    data: { items: [{ name: 'plugin-task' }], pagination: null },
    isLoading: false,
    isError: false,
  });
  apiMock.get.mockResolvedValue({ data: periodic });
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/apps/p" element={<div>the plugin list</div>} />
          <Route
            path="/apps/p/schedule/*"
            element={
              <PluginSchedulePage pluginName="myplugin" schema={schema} />
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  apiMock.get.mockReset();
  apiMock.post.mockReset();
  usePluginTasksMock.mockReset();
});

// PMM-15456: a schedule is created and edited on a page, the same pattern a
// plugin task already used, so the schedules area is a small route tree rather
// than one screen that grows a form inside its table.
describe('PluginSchedulePage', () => {
  it('shows the schedules list at its index route', async () => {
    renderAt('/apps/p/schedule', [makePeriodic()]);

    expect(
      await screen.findByRole('heading', { name: 'Schedules' })
    ).toBeInTheDocument();
    expect(screen.getByTestId('scheduled-tasks-panel')).toBeInTheDocument();
  });

  it('opens the create page from the header action', async () => {
    const user = userEvent.setup();
    renderAt('/apps/p/schedule', []);

    await user.click(await screen.findByTestId('scheduled-tasks-add'));

    expect(
      await screen.findByRole('heading', { name: 'New schedule' })
    ).toBeInTheDocument();
    expect(screen.getByTestId('scheduled-task-form')).toBeInTheDocument();
  });

  it('opens the edit page from a row action', async () => {
    const user = userEvent.setup();
    renderAt('/apps/p/schedule', [makePeriodic({ id: 5 })]);

    await user.click(await screen.findByTestId('scheduled-task-edit-5'));

    expect(
      await screen.findByRole('heading', { name: 'Edit schedule' })
    ).toBeInTheDocument();
  });

  it('returns to the list from the create page', async () => {
    const user = userEvent.setup();
    renderAt('/apps/p/schedule/new', []);

    await user.click(
      await screen.findByRole('button', { name: 'Back to schedules' })
    );

    expect(
      await screen.findByRole('heading', { name: 'Schedules' })
    ).toBeInTheDocument();
  });

  it('returns to the list from the edit page, one level deeper', async () => {
    const user = userEvent.setup();
    renderAt('/apps/p/schedule/5/edit', [makePeriodic({ id: 5 })]);

    await user.click(
      await screen.findByRole('button', { name: 'Back to schedules' })
    );

    expect(
      await screen.findByRole('heading', { name: 'Schedules' })
    ).toBeInTheDocument();
  });

  it('leaves the schedules area entirely from the list', async () => {
    const user = userEvent.setup();
    renderAt('/apps/p/schedule', []);

    await user.click(
      await screen.findByRole('button', { name: 'Back to list' })
    );

    expect(await screen.findByText('the plugin list')).toBeInTheDocument();
  });
});
