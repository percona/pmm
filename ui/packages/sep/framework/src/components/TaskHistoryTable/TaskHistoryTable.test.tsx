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

import {
  render,
  renderHook,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SnackbarProvider } from 'notistack';
import type { PropsWithChildren } from 'react';
import { TaskHistoryTable } from './TaskHistoryTable';
import { StatusBadge } from './StatusBadge';
import { useTaskHistory } from '../../hooks/useTaskHistory';
import type {
  TaskHistoryEntry,
  TaskHistoryStatus,
} from './TaskHistoryTable.types';

vi.mock('@sep/api', () => {
  const RUNNING_STATUSES = new Set(['running', 'pending']);
  return {
    apiClient: {
      get: vi.fn(),
      post: vi.fn(),
    },
    RUNNING_STATUSES,
    isRunningStatus: (status: string) => RUNNING_STATUSES.has(status),
  };
});

import { apiClient } from '@sep/api';

const mockedApiClient = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

function makeEntry(
  id: number,
  status: TaskHistoryStatus,
  overrides: Partial<TaskHistoryEntry> = {}
): TaskHistoryEntry {
  return {
    id,
    status,
    display_name: `task-${id}`,
    started_at: new Date(Date.UTC(2026, 0, 1, 0, id)).toISOString(),
    finished_at:
      status === 'running' || status === 'pending'
        ? null
        : new Date(Date.UTC(2026, 0, 1, 0, id, 30)).toISOString(),
    duration: status === 'running' || status === 'pending' ? null : 30,
    executed_by: 'admin',
    has_logs: true,
    task: { id, name: `task-${id}` } as TaskHistoryEntry['task'],
    execution_request: {
      task: `task-${id}`,
      target: `host-${id}`,
      meta: {},
      tracking: {},
    } as TaskHistoryEntry['execution_request'],
    ...overrides,
  };
}

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function Wrapper({
  children,
  client,
}: PropsWithChildren<{ client: QueryClient }>) {
  return (
    <QueryClientProvider client={client}>
      <SnackbarProvider>{children}</SnackbarProvider>
    </QueryClientProvider>
  );
}

describe('StatusBadge', () => {
  it.each([
    ['success', 'Done'],
    ['failed', 'Failed'],
    ['running', 'Running'],
    ['pending', 'Pending'],
    ['stopped', 'Stopped'],
    ['lost', 'Lost'],
    ['stale', 'Stale'],
  ] as const)('renders %s as %s label', (status, label) => {
    render(<StatusBadge status={status} />);
    expect(screen.getByText(label)).toBeInTheDocument();
    const chip = screen.getByText(label).closest('[data-status]');
    expect(chip).toHaveAttribute('data-status', status);
  });
});

describe('TaskHistoryTable rendering', () => {
  const client = makeQueryClient();

  it('renders rows for provided data and shows status badges', () => {
    const data = [
      makeEntry(1, 'success'),
      makeEntry(2, 'failed'),
      makeEntry(3, 'running'),
    ];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );

    expect(screen.getByText('Done')).toBeInTheDocument();
    expect(screen.getByText('Failed')).toBeInTheDocument();
    expect(screen.getByText('Running')).toBeInTheDocument();
    expect(screen.getByText('task-1')).toBeInTheDocument();
    expect(screen.getByText('host-2')).toBeInTheDocument();
  });

  it('shows empty state when no rows', () => {
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={[]} disablePolling />
      </Wrapper>
    );
    expect(screen.getByText('No task history')).toBeInTheDocument();
  });

  it('renders chain chips with separators when meta has _chain_task_names', () => {
    const data = [
      makeEntry(1, 'success', {
        execution_request: {
          task: 't',
          target: 'h',
          meta: { _chain_task_names: ['a', 'b', 'c'] },
          tracking: {},
        } as unknown as TaskHistoryEntry['execution_request'],
      }),
    ];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );
    const chain = screen.getByTestId('chain-display');
    expect(within(chain).getByText('a')).toBeInTheDocument();
    expect(within(chain).getByText('b')).toBeInTheDocument();
    expect(within(chain).getByText('c')).toBeInTheDocument();
    expect(within(chain).getAllByText('→')).toHaveLength(2);
  });
});

describe('TaskHistoryTable actions', () => {
  const client = makeQueryClient();

  it('shows stop button only for running/pending rows', () => {
    const data = [makeEntry(1, 'success'), makeEntry(2, 'running')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );
    const stopButtons = screen.queryAllByRole('button', { name: 'Stop task' });
    expect(stopButtons).toHaveLength(1);
  });

  it('calls onViewLogs when view-logs clicked', async () => {
    const onViewLogs = vi.fn();
    const data = [makeEntry(1, 'success')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling onViewLogs={onViewLogs} />
      </Wrapper>
    );
    await userEvent.click(screen.getByRole('button', { name: 'View logs' }));
    expect(onViewLogs).toHaveBeenCalledOnce();
    expect(onViewLogs.mock.calls[0][0].id).toBe(1);
  });

  it('opens confirm dialog and invokes onStopTask on Stop click', async () => {
    const onStopTask = vi.fn();
    const data = [makeEntry(2, 'running')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling onStopTask={onStopTask} />
      </Wrapper>
    );
    await userEvent.click(screen.getByRole('button', { name: 'Stop task' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/Are you sure/i)).toBeInTheDocument();
    await userEvent.click(within(dialog).getByRole('button', { name: 'Stop' }));
    expect(onStopTask).toHaveBeenCalledOnce();
    expect(onStopTask.mock.calls[0][0].id).toBe(2);
  });

  it('disables the Stop button when no onStopTask is provided (presentational)', () => {
    const data = [makeEntry(2, 'running')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );
    expect(screen.getByRole('button', { name: 'Stop task' })).toBeDisabled();
  });

  it('disables the Stop button while isStopping is true (presentational)', () => {
    const onStopTask = vi.fn();
    const data = [makeEntry(2, 'running')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable
          data={data}
          disablePolling
          onStopTask={onStopTask}
          isStopping
        />
      </Wrapper>
    );
    expect(screen.getByRole('button', { name: 'Stop task' })).toBeDisabled();
  });

  it('enables the Stop button for running rows when onStopTask is provided and not stopping', () => {
    const onStopTask = vi.fn();
    const data = [makeEntry(2, 'running')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling onStopTask={onStopTask} />
      </Wrapper>
    );
    expect(screen.getByRole('button', { name: 'Stop task' })).toBeEnabled();
  });

  it('skips stop callback when dialog Cancel clicked', async () => {
    const onStopTask = vi.fn();
    const data = [makeEntry(2, 'running')];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling onStopTask={onStopTask} />
      </Wrapper>
    );
    await userEvent.click(screen.getByRole('button', { name: 'Stop task' }));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Cancel' })
    );
    expect(onStopTask).not.toHaveBeenCalled();
  });

  it('shows download button only for completed rows with downloadable artifacts', () => {
    const data = [
      makeEntry(1, 'success', { has_logs: true }),
      makeEntry(2, 'success', { has_logs: false }),
      makeEntry(3, 'running', { has_logs: true }),
    ];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );
    expect(
      screen.getAllByRole('button', { name: 'Download files' })
    ).toHaveLength(1);
  });

  it('opens built-in files dialog when no onDownloadFiles callback provided', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: { 'output/result.txt': { size: 512, is_dir: false } },
    });
    const data = [makeEntry(5, 'success', { has_logs: true })];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );
    await userEvent.click(
      screen.getByRole('button', { name: 'Download files' })
    );
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByText(/Download files/i)).toBeInTheDocument();
  });

  it('calls onDownloadFiles callback instead of opening built-in dialog when provided', async () => {
    const onDownloadFiles = vi.fn();
    const data = [makeEntry(5, 'success', { has_logs: true })];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable
          data={data}
          disablePolling
          onDownloadFiles={onDownloadFiles}
        />
      </Wrapper>
    );
    await userEvent.click(
      screen.getByRole('button', { name: 'Download files' })
    );
    expect(onDownloadFiles).toHaveBeenCalledOnce();
    expect(onDownloadFiles.mock.calls[0][0].id).toBe(5);
    // Built-in dialog should NOT open
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

describe('TaskHistoryTable sort + pagination', () => {
  const client = makeQueryClient();

  function taskNamesInOrder(): string[] {
    const cells = screen.getAllByRole('cell');
    return cells
      .map((c) => c.textContent ?? '')
      .filter((t) => /^task-\d+$/.test(t));
  }

  it('reorders rows when Started column header is toggled', async () => {
    const data = [
      makeEntry(1, 'success'),
      makeEntry(2, 'success'),
      makeEntry(3, 'success'),
    ];
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );

    // Initial sort is started_at desc → task-3, task-2, task-1.
    const initial = taskNamesInOrder();
    expect(initial).toEqual(['task-3', 'task-2', 'task-1']);

    await userEvent.click(screen.getByText('Started'));
    const ascending = taskNamesInOrder();
    expect(ascending).toEqual(['task-1', 'task-2', 'task-3']);
    expect(ascending).not.toEqual(initial);
  });

  it('paginates: next page swaps the visible rows', async () => {
    const data = Array.from({ length: 25 }, (_, i) =>
      makeEntry(i + 1, 'success', {
        // Override started_at so ascending id matches descending default sort.
        started_at: new Date(Date.UTC(2026, 0, 1, 0, 25 - i)).toISOString(),
      })
    );
    render(
      <Wrapper client={client}>
        <TaskHistoryTable data={data} disablePolling />
      </Wrapper>
    );

    const firstPageNames = taskNamesInOrder();
    expect(firstPageNames).toHaveLength(10);
    expect(firstPageNames[0]).toBe('task-1');

    await userEvent.click(screen.getByLabelText(/Go to next page/i));

    const secondPageNames = taskNamesInOrder();
    expect(secondPageNames).toHaveLength(10);
    expect(secondPageNames[0]).toBe('task-11');
    expect(secondPageNames).not.toEqual(firstPageNames);
  });
});

describe('TaskHistoryTable connected stop mutation', () => {
  beforeEach(() => {
    mockedApiClient.get.mockReset();
    mockedApiClient.post.mockReset();
  });

  it('without onStopTask, posts to stop endpoint and refetches', async () => {
    const runningEntry = makeEntry(42, 'running');
    const stoppedEntry = makeEntry(42, 'stopped', { duration: 10 });

    let returnRunning = true;
    mockedApiClient.get.mockImplementation(async () => ({
      data: {
        items: [returnRunning ? runningEntry : stoppedEntry],
        total: 1,
        offset: 0,
        limit: 10,
      },
    }));
    mockedApiClient.post.mockImplementation(async () => {
      returnRunning = false;
      return { data: stoppedEntry };
    });

    const client = makeQueryClient();
    render(
      <Wrapper client={client}>
        <TaskHistoryTable taskName="my-task" disablePolling />
      </Wrapper>
    );

    await waitFor(() =>
      expect(screen.getByText('Running')).toBeInTheDocument()
    );

    await userEvent.click(screen.getByRole('button', { name: 'Stop task' }));
    const dialog = await screen.findByRole('dialog');
    await userEvent.click(within(dialog).getByRole('button', { name: 'Stop' }));

    await waitFor(() =>
      expect(mockedApiClient.post).toHaveBeenCalledWith(
        '/sep/task-history/42/stop/'
      )
    );

    await waitFor(() =>
      expect(screen.getByText('Stopped')).toBeInTheDocument()
    );
    expect(mockedApiClient.get.mock.calls.length).toBeGreaterThan(1);
  });
});

describe('useTaskHistory polling', () => {
  beforeEach(() => {
    mockedApiClient.get.mockReset();
  });

  const wait = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));

  it('polls only while running tasks exist, then stops', async () => {
    let returnRunning = true;
    mockedApiClient.get.mockImplementation(async () => ({
      data: {
        items: returnRunning
          ? [makeEntry(1, 'running')]
          : [makeEntry(1, 'success')],
        total: 1,
        offset: 0,
        limit: 10,
      },
    }));

    const client = makeQueryClient();
    const { result } = renderHook(
      () => useTaskHistory({ pollingIntervalMs: 50 }),
      {
        wrapper: ({ children }) => (
          <Wrapper client={client}>{children}</Wrapper>
        ),
      }
    );

    await waitFor(() =>
      expect(result.current.data?.items[0].status).toBe('running')
    );
    const callsAfterFirst = mockedApiClient.get.mock.calls.length;
    await wait(160);
    expect(mockedApiClient.get.mock.calls.length).toBeGreaterThan(
      callsAfterFirst
    );

    returnRunning = false;
    await waitFor(() =>
      expect(result.current.data?.items[0].status).toBe('success')
    );
    await wait(80);
    const callsAfterStable = mockedApiClient.get.mock.calls.length;
    await wait(200);
    expect(mockedApiClient.get.mock.calls.length).toBe(callsAfterStable);
  });

  it('does not poll when disablePolling=true even with running rows', async () => {
    mockedApiClient.get.mockResolvedValue({
      data: {
        items: [makeEntry(1, 'running')],
        total: 1,
        offset: 0,
        limit: 10,
      },
    });

    const client = makeQueryClient();
    renderHook(
      () => useTaskHistory({ pollingIntervalMs: 30, disablePolling: true }),
      {
        wrapper: ({ children }) => (
          <Wrapper client={client}>{children}</Wrapper>
        ),
      }
    );

    await waitFor(() => expect(mockedApiClient.get).toHaveBeenCalledTimes(1));
    await wait(150);
    expect(mockedApiClient.get).toHaveBeenCalledTimes(1);
  });
});
