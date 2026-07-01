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

import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { SnippetExecutionAccordion } from './SnippetExecutionAccordion';
import { apiClient, type PluginSchema } from '@sep/api';
import type { TaskHistoryEntry } from '../TaskHistoryTable';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn(), post: vi.fn() },
}));

vi.mock('../TaskLogViewer', () => ({
  TaskLogViewer: ({ taskHistoryId }: { taskHistoryId: number }) => (
    <div data-testid="task-log-viewer" data-task-id={taskHistoryId} />
  ),
}));

vi.mock('../TaskHistoryTable', () => ({
  TaskHistoryTable: ({
    data,
    onStopTask,
  }: {
    data?: TaskHistoryEntry[];
    onStopTask?: (entry: TaskHistoryEntry) => void;
  }) => (
    <div data-testid="task-history-table" data-row-count={data?.length ?? 0}>
      {data?.[0] && onStopTask ? (
        <button type="button" onClick={() => onStopTask(data[0])}>
          Stop {String(data[0].id)}
        </button>
      ) : null}
    </div>
  ),
}));

const mockedApi = apiClient as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
};

type FormSection = NonNullable<PluginSchema['forms']>[number];

function makeSchema(extraFields: FormSection['fields'] = []): PluginSchema {
  return {
    name: 'snippets',
    display_name: 'Test Snippet',
    forms: [
      {
        title: 'Execution',
        fields: [
          {
            type: 'host',
            name: 'executor_host',
            label: 'Executor Host',
            required: true,
          },
          {
            type: 'string',
            name: 'table_name',
            label: 'Table Name',
            required: false,
          },
          ...extraFields,
        ],
      },
    ],
  };
}

function renderWithProviders(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, staleTime: Infinity } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

describe('SnippetExecutionAccordion', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders title in collapsed state without fetching schema', () => {
    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
      />
    );

    expect(screen.getByText('Check Script')).toBeInTheDocument();
    expect(mockedApi.get).not.toHaveBeenCalled();
  });

  it('uses filename as fallback title', () => {
    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
      />
    );

    expect(screen.getByText('check.sh')).toBeInTheDocument();
  });

  it('fetches schema and renders form fields when expanded', async () => {
    mockedApi.get.mockResolvedValue({ data: makeSchema() });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Table Name/i)).toBeInTheDocument();
    });
  });

  it('omits executor_host field when executorHost prop provided', async () => {
    mockedApi.get.mockResolvedValue({ data: makeSchema() });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Table Name/i)).toBeInTheDocument();
    });

    expect(screen.queryByLabelText(/Executor Host/i)).not.toBeInTheDocument();
  });

  it('posts to snippets execute endpoint on submit with executorHost injected', async () => {
    const user = userEvent.setup();
    mockedApi.get.mockResolvedValue({ data: makeSchema() });
    mockedApi.post.mockResolvedValue({ data: { task_id: 7 } });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Table Name/i)).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: /execute/i }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        expect.stringContaining('check.sh'),
        {
          executor_host: 'db1',
          sudo: false,
          args: {},
        }
      );
    });
  });

  it('posts executor_host from rendered form when executorHost is not provided', async () => {
    const user = userEvent.setup();
    mockedApi.get.mockImplementation((url: string) =>
      Promise.resolve({
        data:
          url === '/sep/hosts/'
            ? [{ id: 'db2', name: 'db2', address: '10.0.0.2' }]
            : makeSchema(),
        headers: {},
      })
    );
    mockedApi.post.mockResolvedValue({ data: { task_id: 7 } });

    renderWithProviders(
      <SnippetExecutionAccordion snippetFilename="check.sh" defaultExpanded />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Executor Host/i)).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText(/Executor Host/i));
    await user.click(await screen.findByRole('option', { name: 'db2' }));
    await user.click(screen.getByRole('button', { name: /execute/i }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        expect.stringContaining('check.sh'),
        {
          executor_host: 'db2',
          sudo: false,
          args: {},
        }
      );
    });
  });

  it('shows TaskLogViewer after successful execution', async () => {
    const user = userEvent.setup();
    mockedApi.get.mockResolvedValue({ data: makeSchema() });
    mockedApi.post.mockResolvedValue({ data: { task_id: 42 } });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: /execute/i })
      ).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: /execute/i }));

    await waitFor(() => {
      const viewer = screen.getByTestId('task-log-viewer');
      expect(viewer).toBeInTheDocument();
      expect(viewer).toHaveAttribute('data-task-id', '42');
    });
  });

  it('renders TaskHistoryTable when showHistory is true', async () => {
    mockedApi.get.mockImplementation((url: string) =>
      Promise.resolve({
        data: url.endsWith('/history') ? { items: [] } : makeSchema(),
      })
    );

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
        showHistory
      />
    );

    await waitFor(() => {
      expect(screen.getByTestId('task-history-table')).toBeInTheDocument();
    });
    expect(mockedApi.get).toHaveBeenCalledWith(
      '/apps/snippets/snippet/history?snippet_filename=check.sh'
    );
  });

  it('wires the Stop button to the stop-task endpoint with the row id', async () => {
    mockedApi.get.mockImplementation((url: string) =>
      Promise.resolve({
        data: url.includes('/snippet/history')
          ? {
              items: [
                {
                  id: 99,
                  status: 'running',
                  has_logs: false,
                  execution_request: {
                    task: 's',
                    target: 'h',
                    meta: {},
                    tracking: {},
                  },
                  task: { id: 1, name: 's' },
                },
              ],
            }
          : makeSchema(),
      })
    );
    mockedApi.post.mockResolvedValue({ data: { id: 99, status: 'stopped' } });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
        showHistory
      />
    );

    await userEvent.click(
      await screen.findByRole('button', { name: 'Stop 99' })
    );

    await waitFor(() =>
      expect(mockedApi.post).toHaveBeenCalledWith('/sep/task-history/99/stop/')
    );

    // The stop hook only invalidates ['task-history']; this accordion's history
    // is keyed under ['snippets', filename, 'history'], so the wired onSuccess
    // must refetch it directly — assert the history endpoint is hit again.
    const historyGets = () =>
      mockedApi.get.mock.calls.filter((call) =>
        String(call[0]).includes('/snippet/history')
      ).length;
    await waitFor(() => expect(historyGets()).toBeGreaterThan(1));
  });

  it('does not render TaskHistoryTable when showHistory is false', async () => {
    mockedApi.get.mockResolvedValue({ data: makeSchema() });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Table Name/i)).toBeInTheDocument();
    });

    expect(screen.queryByTestId('task-history-table')).not.toBeInTheDocument();
  });

  it('renders sudo field when schema includes it', async () => {
    mockedApi.get.mockResolvedValue({
      data: makeSchema([
        { type: 'bool', name: 'sudo', label: 'Run with sudo', required: false },
      ]),
    });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Run with sudo/i)).toBeInTheDocument();
    });
  });

  it('shows executor_host field when executorHost is empty string (treated as not hoisting)', async () => {
    mockedApi.get.mockImplementation((url: string) =>
      Promise.resolve({
        data:
          url === '/sep/hosts/'
            ? [{ id: 'db2', name: 'db2', address: '10.0.0.2' }]
            : makeSchema(),
        headers: {},
      })
    );

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost=""
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Executor Host/i)).toBeInTheDocument();
    });
  });

  it('fetches schema with execution_only param', async () => {
    mockedApi.get.mockResolvedValue({ data: makeSchema() });

    renderWithProviders(
      <SnippetExecutionAccordion snippetFilename="check.sh" defaultExpanded />
    );

    await waitFor(() =>
      expect(mockedApi.get).toHaveBeenCalledWith(
        '/apps/snippets/snippet/schema?snippet_filename=check.sh',
        {
          params: { execution_only: true },
        }
      )
    );
  });

  it('passes executor_host from form (not hoisted) when executorHost is empty string', async () => {
    const user = userEvent.setup();
    mockedApi.get.mockImplementation((url: string) =>
      Promise.resolve({
        data:
          url === '/sep/hosts/'
            ? [{ id: 'db2', name: 'db2', address: '10.0.0.2' }]
            : makeSchema(),
        headers: {},
      })
    );
    mockedApi.post.mockResolvedValue({ data: { task_id: 9 } });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost=""
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Executor Host/i)).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText(/Executor Host/i));
    await user.click(await screen.findByRole('option', { name: 'db2' }));
    await user.click(screen.getByRole('button', { name: /execute/i }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        expect.stringContaining('check.sh'),
        {
          executor_host: 'db2',
          sudo: false,
          args: {},
        }
      );
    });
  });

  it('excludes sudo from args but includes at top level on submit', async () => {
    const user = userEvent.setup();
    mockedApi.get.mockResolvedValue({
      data: makeSchema([
        { type: 'bool', name: 'sudo', label: 'Run with sudo', required: false },
      ]),
    });
    mockedApi.post.mockResolvedValue({ data: { task_id: 7 } });

    renderWithProviders(
      <SnippetExecutionAccordion
        snippetFilename="check.sh"
        executorHost="db1"
        title="Check Script"
        defaultExpanded
      />
    );

    await waitFor(() => {
      expect(screen.getByLabelText(/Run with sudo/i)).toBeInTheDocument();
    });

    await user.click(screen.getByLabelText(/Run with sudo/i));
    await user.click(screen.getByRole('button', { name: /execute/i }));

    await waitFor(() => {
      expect(mockedApi.post).toHaveBeenCalledWith(
        expect.stringContaining('check.sh'),
        {
          executor_host: 'db1',
          sudo: true,
          args: {},
        }
      );
    });

    // Verify sudo is NOT in args
    const callArgs = mockedApi.post.mock.calls[0][1];
    expect(callArgs.args).not.toHaveProperty('sudo');
  });
});
