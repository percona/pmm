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
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

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

import { formatTimestamp } from '../../utils/formatTimestamp';
import { PluginScheduleFormPage } from './PluginScheduleFormPage';
import type { PeriodicTaskResponse } from '../ScheduledTasksPanel/hooks';

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

/**
 * The create POST, ignoring the schedule-preview POST that the form issues
 * while the user types. Both go through the same mocked client.
 */
function createCalls() {
  return apiMock.post.mock.calls.filter(
    ([url]) => !String(url).includes('schedule/preview')
  );
}

const LIST_SENTINEL = 'the schedules list';

function renderForm(mode: 'create' | 'edit', id?: number) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  const path =
    mode === 'create' ? '/apps/p/schedule/new' : `/apps/p/schedule/${id}/edit`;
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/apps/p/schedule" element={<div>{LIST_SENTINEL}</div>} />
          <Route
            path="/apps/p/schedule/new"
            element={
              <PluginScheduleFormPage pluginName="myplugin" mode="create" />
            }
          />
          <Route
            path="/apps/p/schedule/:id/edit"
            element={
              <PluginScheduleFormPage pluginName="myplugin" mode="edit" />
            }
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

/** The form, once its schedules query has resolved. */
const form = () => screen.findByTestId('scheduled-task-form');

beforeEach(() => {
  apiMock.get.mockReset();
  apiMock.post.mockReset();
  apiMock.put.mockReset();
  apiMock.delete.mockReset();
  usePluginTasksMock.mockReset();
  authMock.canMutate = true;
});

describe('PluginScheduleFormPage — create', () => {
  it('creates an interval task via POST when filling the form', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 42 }) });
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    const every = within(el).getByTestId('text-input-interval-every');
    await user.clear(every);
    await user.type(every, '5');
    await user.click(within(el).getByRole('button', { name: /Create/i }));

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

  it('returns to the schedules list once the create succeeds', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 42 }) });
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();
    await user.click(within(el).getByRole('button', { name: /Create/i }));

    expect(await screen.findByText(LIST_SENTINEL)).toBeInTheDocument();
  });

  it('leaves for the list without creating anything when cancelled', async () => {
    setup([]);
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();
    await user.click(within(el).getByRole('button', { name: /Cancel/i }));

    expect(await screen.findByText(LIST_SENTINEL)).toBeInTheDocument();
    expect(createCalls()).toHaveLength(0);
  });

  it('switches to cron mode and submits a crontab body', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 43 }) });
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    await user.click(within(el).getByTestId('radio-option-cron'));
    await user.type(
      within(el).getByTestId('text-input-cron-expression'),
      '*/5 * * * *'
    );

    expect(within(el).getByTestId('sched-form-cron-preview')).toHaveTextContent(
      /every 5 minutes/i
    );

    await user.click(within(el).getByRole('button', { name: /Create/i }));

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
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    await user.click(within(el).getByTestId('radio-option-cron'));
    await user.type(
      within(el).getByTestId('text-input-cron-expression'),
      'not-a-cron'
    );
    await user.click(within(el).getByRole('button', { name: /Create/i }));

    // Said twice on purpose: once under the field as its validation error,
    // once in the humanised preview that cannot describe the expression.
    await waitFor(() =>
      expect(
        within(el).getAllByText(/Invalid cron expression/i).length
      ).toBeGreaterThanOrEqual(2)
    );
    expect(within(el).getByTestId('sched-form-cron-preview')).toHaveTextContent(
      /Invalid cron expression/i
    );
    expect(createCalls()).toHaveLength(0);
  });

  it('rejects an empty interval-every value and does not POST', async () => {
    setup([]);
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    await user.clear(within(el).getByTestId('text-input-interval-every'));
    await user.click(within(el).getByRole('button', { name: /Create/i }));

    expect(
      await within(el).findByText(/Enter how often this runs/i)
    ).toBeInTheDocument();
    expect(createCalls()).toHaveLength(0);
  });

  it('submits chain_task_names in execute_request when a chain is configured', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 50 }) });
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    const chainBuilder = within(el).getByTestId('chain-builder');
    await user.click(within(chainBuilder).getByRole('combobox'));
    await user.click(
      await screen.findByRole('option', { name: 'other-plugin-task' })
    );

    await user.click(within(el).getByRole('button', { name: /Create/i }));

    await waitFor(() => expect(createCalls()).toHaveLength(1));
    const [, body] = createCalls()[0];
    expect(body.execute_request).toMatchObject({
      chain_task_names: ['other-plugin-task'],
      chain_on_failure: false,
    });
  });

  it('shows the failure on the form when the create mutation fails', async () => {
    setup([]);
    apiMock.post.mockRejectedValue(new Error('create blew up'));
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();
    await user.click(within(el).getByRole('button', { name: /Create/i }));

    expect(await within(el).findByText(/create blew up/i)).toBeInTheDocument();
    expect(screen.queryByText(LIST_SENTINEL)).not.toBeInTheDocument();
  });
});

describe('PluginScheduleFormPage — edit', () => {
  it('round-trips an existing cron task and submits an updated crontab', async () => {
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
    const user = userEvent.setup();

    renderForm('edit', 21);
    const el = await form();

    const cron = within(el).getByTestId('text-input-cron-expression');
    expect(cron).toHaveValue('0 6 * * *');

    await user.clear(cron);
    await user.type(cron, '*/15 * * * *');
    await user.click(within(el).getByRole('button', { name: /Save/i }));

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

  it('submits an edit via PUT with the updated schedule', async () => {
    setup([makePeriodic({ id: 11 })]);
    apiMock.put.mockResolvedValue({ data: makePeriodic({ id: 11 }) });
    const user = userEvent.setup();

    renderForm('edit', 11);
    const el = await form();

    const every = within(el).getByTestId('text-input-interval-every');
    await user.clear(every);
    await user.type(every, '10');
    await user.click(within(el).getByRole('button', { name: /Save/i }));

    await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
    const [url, body] = apiMock.put.mock.calls[0];
    expect(url).toBe('/sep/periodic-tasks/11');
    expect(body.interval).toMatchObject({ every: 10, period: 'hours' });
  });

  it('names the task it is editing instead of offering a task picker', async () => {
    setup([makePeriodic({ id: 11 })]);

    renderForm('edit', 11);
    const el = await form();

    expect(within(el).getByTestId('sched-form-task-name')).toHaveTextContent(
      'plugin-task'
    );
    expect(
      within(el).queryByTestId('select-task-button')
    ).not.toBeInTheDocument();
  });

  it('says so when the schedule is gone', async () => {
    setup([makePeriodic({ id: 11 })]);

    renderForm('edit', 999);

    expect(
      await screen.findByText(/That schedule no longer exists/i)
    ).toBeInTheDocument();
  });
});

describe('PluginScheduleFormPage — write access', () => {
  it('withholds the form from a session that may not mutate', async () => {
    authMock.canMutate = false;
    setup([]);

    renderForm('create');

    expect(
      await screen.findByTestId('plugin-schedule-read-only')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('scheduled-task-form')).not.toBeInTheDocument();
    // The back chrome stays: nothing links here for such a session, so anyone
    // who arrived did so by URL and needs a way out.
    expect(
      screen.getByRole('button', { name: 'Back to schedules' })
    ).toBeInTheDocument();
  });
});

// PMM-15454: the form has to say which zone it is talking about, and the start
// time it sends has to be the wall clock the field showed.
describe('PluginScheduleFormPage — stating the timezone in force', () => {
  it('states UTC on the create form and labels the start-time field with it', async () => {
    setup([]);

    renderForm('create');
    const el = await form();

    expect(
      within(el).getByTestId('sched-form-timezone-notice')
    ).toHaveTextContent('Runs in UTC');
    expect(within(el).getByLabelText(/Start time \(UTC\)/i)).toBeVisible();
  });

  it('tracks the picked zone in cron mode', async () => {
    setup([]);
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    await user.click(within(el).getByTestId('radio-option-cron'));

    const picker = within(el).getByTestId('text-input-cron-timezone');
    await user.clear(picker);
    await user.type(picker, 'Europe/Lisbon');
    await user.click(
      await screen.findByRole('option', { name: 'Europe/Lisbon' })
    );

    await waitFor(() =>
      expect(
        within(el).getByTestId('sched-form-timezone-notice')
      ).toHaveTextContent('Runs in Europe/Lisbon')
    );
  });

  it('sends the start time as the UTC wall clock that was shown', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({ data: makePeriodic({ id: 60 }) });
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    fireEvent.change(within(el).getByTestId('date-time-picker-start-time'), {
      target: { value: '03/01/2026 02:30 AM' },
    });
    await user.click(within(el).getByRole('button', { name: /Create/i }));

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
    const user = userEvent.setup();

    renderForm('edit', 62);
    const el = await form();

    const every = within(el).getByTestId('text-input-interval-every');
    await user.clear(every);
    await user.type(every, '6');
    await user.click(within(el).getByRole('button', { name: /Save/i }));

    await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
    const [, body] = apiMock.put.mock.calls[0];
    expect(body.interval).toMatchObject({ every: 6 });
    expect(body.start_time).toBe('2026-03-01T02:30:45.123Z');
  });

  it('does not move a start time the reader never touched', async () => {
    // 02:30 UTC on 2026-03-08 falls inside America/New_York's spring-forward
    // gap, so a reader there sees the picker render 03:30 (see
    // `DateTimeInput/wallClockValue.ts`). Saving an edit to the interval must
    // still send the stored instant, not the hour the display shifted to.
    setup([makePeriodic({ id: 65, start_time: '2026-03-08T02:30:00.000Z' })]);
    apiMock.put.mockResolvedValue({ data: { id: 65 } });
    const user = userEvent.setup();

    renderForm('edit', 65);
    const el = await form();

    const every = within(el).getByTestId('text-input-interval-every');
    await user.clear(every);
    await user.type(every, '4');
    await user.click(within(el).getByRole('button', { name: /Save/i }));

    await waitFor(() => expect(apiMock.put).toHaveBeenCalledTimes(1));
    const [, body] = apiMock.put.mock.calls[0];
    expect(body.interval).toMatchObject({ every: 4 });
    expect(body.start_time).toBe('2026-03-08T02:30:00.000Z');
  });

  it('sends the edited start time when the field itself is changed', async () => {
    setup([makePeriodic({ id: 63, start_time: '2026-03-01T02:30:45.123Z' })]);
    apiMock.put.mockResolvedValue({ data: { id: 63 } });
    const user = userEvent.setup();

    renderForm('edit', 63);
    const el = await form();

    fireEvent.change(within(el).getByTestId('date-time-picker-start-time'), {
      target: { value: '04/02/2026 07:15 AM' },
    });
    await user.click(within(el).getByRole('button', { name: /Save/i }));

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
    const user = userEvent.setup();

    renderForm('edit', 64);
    const el = await form();

    const picker = within(el).getByTestId('text-input-cron-timezone');
    expect(picker).toHaveValue('US/Eastern');
    expect(
      within(el).getByTestId('sched-form-timezone-notice')
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

    renderForm('edit', 61);
    const el = await form();

    expect(within(el).getByTestId('date-time-picker-start-time')).toHaveValue(
      '03/01/2026 02:30 AM'
    );
  });
});

// PMM-15454 / PMM-15480: the upcoming runs come from the scheduler's own
// objects, so the form and the saved schedule cannot disagree about "next".
describe('PluginScheduleFormPage — previewing the next runs', () => {
  function previewCalls() {
    return apiMock.post.mock.calls.filter(([url]) =>
      String(url).includes('schedule/preview')
    );
  }

  it('shows the next three runs the backend reports', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({
      data: {
        timezone: 'Europe/Lisbon',
        next_run_at: '2026-03-01T02:00:00Z',
        next_runs: [
          '2026-03-01T02:00:00Z',
          '2026-03-02T02:00:00Z',
          '2026-03-03T02:00:00Z',
          '2026-03-04T02:00:00Z',
        ],
      },
    });

    renderForm('create');
    const el = await form();

    const runs = await within(el).findByTestId(
      'sched-form-next-runs',
      {},
      { timeout: 3000 }
    );
    await waitFor(() => expect(runs).toHaveTextContent(/Next runs:/));
    // Three, not the four the backend offered.
    for (const iso of [
      '2026-03-01T02:00:00Z',
      '2026-03-02T02:00:00Z',
      '2026-03-03T02:00:00Z',
    ]) {
      expect(runs.textContent).toContain(formatTimestamp(iso)!.display);
    }
    expect(runs.textContent).not.toContain(
      formatTimestamp('2026-03-04T02:00:00Z')!.display
    );
  });

  it('states the zone the backend resolved, not the one the form assumed', async () => {
    setup([]);
    apiMock.post.mockResolvedValue({
      data: { timezone: 'Europe/Lisbon', next_run_at: null, next_runs: [] },
    });

    renderForm('create');
    const el = await form();

    await waitFor(
      () =>
        expect(
          within(el).getByTestId('sched-form-timezone-notice')
        ).toHaveTextContent('Runs in Europe/Lisbon'),
      { timeout: 3000 }
    );
  });

  it('asks for no preview while the cron expression is invalid', async () => {
    setup([]);
    const user = userEvent.setup();

    renderForm('create');
    const el = await form();

    await user.click(within(el).getByTestId('radio-option-cron'));
    // Interval mode previews on mount; only what cron mode asks for counts.
    apiMock.post.mockClear();
    await user.type(
      within(el).getByTestId('text-input-cron-expression'),
      'nonsense'
    );

    await waitFor(() => expect(previewCalls()).toHaveLength(0), {
      timeout: 1500,
    });
    expect(
      within(el).queryByTestId('sched-form-next-runs')
    ).not.toBeInTheDocument();
  });

  it('says so when the preview cannot be worked out', async () => {
    setup([]);
    apiMock.post.mockRejectedValue(new Error('422'));

    renderForm('create');
    const el = await form();

    await waitFor(
      () =>
        expect(
          within(el).getByTestId('sched-form-next-runs')
        ).toHaveTextContent(/Could not work out the next runs/),
      { timeout: 3000 }
    );
  });
});
