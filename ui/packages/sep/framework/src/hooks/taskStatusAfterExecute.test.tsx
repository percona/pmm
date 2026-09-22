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

import { act, render, screen, waitFor } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { apiClient, usePluginTask } from '@sep/api';
import { createTestQueryClient } from '../../tests/queryWrapper';
import { useExecuteTask } from './useTaskHistory';

/**
 * The detail page reads the task for its header chip and hands execution to a
 * child action bar, so a finished mutation re-renders the child and never the
 * component holding the task. These tests keep that split, against a fake
 * backend whose task status moves the way the real one does: `null` until the
 * first run, `pending` once execute returns, then `running`, then terminal.
 */

const PLUGIN = 'mysql_backups';
const TASK = 'nightly';
const POLL_MS = 5000;

type TaskRecord = { name: string; status: string | null };
type Execute = ReturnType<typeof useExecuteTask>['mutateAsync'];

let serverStatus: string | null;
let execute: Execute | undefined;

function ActionBar() {
  execute = useExecuteTask(PLUGIN).mutateAsync;
  return null;
}

function DetailPage() {
  const { data } = usePluginTask<TaskRecord>(PLUGIN, TASK);
  return (
    <>
      <output data-testid="header-status">
        {data === undefined ? 'loading' : String(data.status)}
      </output>
      <ActionBar />
    </>
  );
}

function renderDetailPage() {
  render(
    <QueryClientProvider client={createTestQueryClient()}>
      <DetailPage />
    </QueryClientProvider>
  );
}

async function expectHeaderStatus(status: string) {
  await waitFor(() =>
    expect(screen.getByTestId('header-status')).toHaveTextContent(status)
  );
}

async function executeTask() {
  const run = execute;
  if (!run) {
    throw new Error('the action bar has not rendered');
  }
  await act(async () => {
    await run({ taskName: TASK });
  });
}

function taskRequestCount() {
  return vi.mocked(apiClient.get).mock.calls.length;
}

beforeEach(() => {
  serverStatus = null;
  execute = undefined;
  vi.spyOn(apiClient, 'get').mockImplementation(async (url: string) => {
    if (url === `/apps/${PLUGIN}/${TASK}`) {
      return { data: { name: TASK, status: serverStatus } };
    }
    throw new Error(`unexpected GET ${url}`);
  });
  vi.spyOn(apiClient, 'post').mockImplementation(async () => {
    serverStatus = 'pending';
    return { data: { task_name: TASK, task_id: 1, status: 'pending' } };
  });
});

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe('task status on the page that executes the task', () => {
  it('picks up the run the execute started', async () => {
    renderDetailPage();
    await expectHeaderStatus('null');

    await executeTask();

    await expectHeaderStatus('pending');
  });

  it('follows the run to its terminal status, then stops asking', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    renderDetailPage();
    await expectHeaderStatus('null');

    await executeTask();
    await expectHeaderStatus('pending');

    serverStatus = 'running';
    await act(() => vi.advanceTimersByTimeAsync(POLL_MS));
    await expectHeaderStatus('running');

    serverStatus = 'success';
    await act(() => vi.advanceTimersByTimeAsync(POLL_MS));
    await expectHeaderStatus('success');

    const requestsAtRest = taskRequestCount();
    await act(() => vi.advanceTimersByTimeAsync(POLL_MS * 3));
    expect(taskRequestCount()).toBe(requestsAtRest);
  });

  it('does not poll a task that has never run', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    renderDetailPage();
    await expectHeaderStatus('null');

    const requestsAtRest = taskRequestCount();
    await act(() => vi.advanceTimersByTimeAsync(POLL_MS * 3));
    expect(taskRequestCount()).toBe(requestsAtRest);
  });
});
