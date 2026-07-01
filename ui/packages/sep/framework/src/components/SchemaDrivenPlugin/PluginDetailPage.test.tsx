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

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { SnackbarProvider } from 'notistack';
import type { PluginSchema } from '@sep/api';
import { PluginDetailPage, resolveTabFromSplat } from './PluginDetailPage';

const mockDeleteMutate = vi.fn();
const mockExecuteMutate = vi.fn();
const mockUsePluginTask = vi.fn();
const mockUsePluginEntityDetail = vi.fn();
const { stopMutate } = vi.hoisted(() => ({ stopMutate: vi.fn() }));
interface MockStatsResult {
  data: unknown;
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
}
const mockUseTaskStats = vi.fn<(...args: unknown[]) => MockStatsResult>(() => ({
  data: undefined,
  isLoading: false,
  isError: false,
}));

// Manual factory keeps axios out of the resolution graph.
vi.mock('@sep/api', () => ({
  usePluginTask: (...args: unknown[]) => mockUsePluginTask(...args),
  // Consumed transitively by the generic ScheduleSummary (gated on
  // capabilities.scheduling) via useScheduledTasksForPlugin. No tasks ->
  // the summary renders its "Not scheduled" state, leaving the execute/delete
  // flows under test untouched.
  usePluginTasks: () => ({
    data: [],
    isLoading: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useDeletePluginTask: () => ({
    mutateAsync: mockDeleteMutate,
    isPending: false,
  }),
  useDeletePluginEntity: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePluginEntityDetail: (...args: unknown[]) =>
    mockUsePluginEntityDetail(...args),
  // Needed by useTaskLogs / useExecutionEvents in the component tree
  getToken: () => null,
  refreshAccessToken: vi.fn(),
  emitUnauthorized: vi.fn(),
  apiClient: { get: vi.fn(), post: vi.fn(), defaults: {} },
  setTokenProvider: vi.fn(),
  ApiError: class ApiError extends Error {
    status?: number;
    constructor(details: { status?: number; message: string }) {
      super(details.message);
      this.status = details.status;
    }
  },
}));

vi.mock('../../hooks', () => ({
  useTaskHistoryByName: () => ({
    data: { items: [] },
    isLoading: false,
    error: null,
  }),
  useTaskHistoryByNames: () => ({
    data: { items: [] },
    isLoading: false,
    error: null,
  }),
  useExecuteTask: () => ({
    mutateAsync: mockExecuteMutate,
    isPending: false,
  }),
  useStopTaskHistory: () => ({ mutate: stopMutate, isPending: false }),
}));

// Logs tab renders the real TaskHistoryTable; stub it to capture the wired
// onStopTask handler (no other test exercises the Logs tab / this component).
vi.mock('../TaskHistoryTable', () => ({
  TaskHistoryTable: ({
    onStopTask,
  }: {
    onStopTask?: (entry: { id: number }) => void;
  }) =>
    onStopTask ? (
      <button type="button" onClick={() => onStopTask({ id: 7 })}>
        Stop row
      </button>
    ) : (
      <div data-testid="task-history-table" />
    ),
}));

vi.mock('../../hooks/useTaskStats', () => ({
  useTaskStats: (taskName?: string, enabled?: boolean) =>
    mockUseTaskStats(taskName, enabled),
}));

vi.mock('./DetailSyntaxHighlighter', () => ({
  default: ({ value, language }: { value: unknown; language: string }) => (
    <pre data-testid="detail-syntax-highlighter" data-language={language}>
      {String(value)}
    </pre>
  ),
}));

const schema: PluginSchema = {
  pluginName: 'checksums',
  display_name: 'Checksum',
  description: 'Test',
  capabilities: { scheduling: true },
  list_view: {
    columns: [
      { key: 'name', label: 'Name' },
      { key: 'status', label: 'Status', format: 'status' },
    ],
    default_sort: '-id',
  },
  formSchema: { sections: [] },
} as unknown as PluginSchema;

function makeClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
}

function renderAt(path: string) {
  return render(
    <QueryClientProvider client={makeClient()}>
      <SnackbarProvider>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route
              path="/apps/:plugin/task/:id/*"
              element={
                <PluginDetailPage schema={schema} pluginName="checksums" />
              }
            />
            <Route path="/apps/:plugin" element={<div>list page</div>} />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    </QueryClientProvider>
  );
}

describe('PluginDetailPage — detail_view sections', () => {
  function executionSchema(
    overrides: Partial<PluginSchema> = {}
  ): PluginSchema {
    return {
      pluginName: 'checksums',
      display_name: 'Checksum',
      description: 'Test',
      capabilities: {},
      list_view: {
        columns: [
          { key: 'name', label: 'Name' },
          { key: 'status', label: 'Status', format: 'status' },
        ],
        default_sort: '-id',
      },
      formSchema: { sections: [] },
      detail_view: {
        sections: [
          {
            title: 'Execution',
            fields: [
              { path: 'data.meta.command', label: 'Command' },
              { path: 'data.meta.args', label: 'Args' },
              { path: 'data.meta.target', label: 'Target' },
            ],
          },
        ],
      },
      ...overrides,
    } as unknown as PluginSchema;
  }

  it('renders a section with each labelled field resolved from the task', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: {
          meta: {
            command: 'pt-table-checksum',
            args: '--foo',
            target: 'pmm',
          },
        },
      },
      isLoading: false,
    });

    renderWithSchema(executionSchema());

    expect(
      screen.getByRole('heading', { name: 'Execution' })
    ).toBeInTheDocument();
    expect(screen.getByText('pt-table-checksum')).toBeInTheDocument();
    expect(screen.getByText('--foo')).toBeInTheDocument();
    expect(screen.getByText('pmm')).toBeInTheDocument();
  });

  it('skips fields whose path resolves to undefined', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: { meta: { command: 'pt-table-checksum' } },
      },
      isLoading: false,
    });

    renderWithSchema(executionSchema());

    expect(screen.getByText('Command')).toBeInTheDocument();
    expect(screen.queryByText('Args')).toBeNull();
    expect(screen.queryByText('Target')).toBeNull();
  });

  it('skips fields whose path resolves to an empty string', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: {
          meta: { command: '', args: '--foo', target: 'pmm' },
        },
      },
      isLoading: false,
    });

    renderWithSchema(executionSchema());

    expect(screen.queryByText('Command')).toBeNull();
    expect(screen.getByText('Args')).toBeInTheDocument();
  });

  it('hides the whole section when every field resolves empty', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed', data: {} },
      isLoading: false,
    });

    renderWithSchema(executionSchema());

    expect(screen.queryByRole('heading', { name: 'Execution' })).toBeNull();
  });

  it('renders multiple sections in declared order', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: {
          meta: { command: 'cmd', args: '--foo', target: 'pmm' },
          parent: 'parent-task',
        },
      },
      isLoading: false,
    });

    renderWithSchema(
      executionSchema({
        detail_view: {
          sections: [
            {
              title: 'Execution',
              fields: [{ path: 'data.meta.command', label: 'Command' }],
            },
            {
              title: 'Chain',
              fields: [{ path: 'data.parent', label: 'Parent' }],
            },
          ],
        },
      } as unknown as PluginSchema)
    );

    const headings = screen
      .getAllByRole('heading', { level: 6 })
      .map((h) => h.textContent);
    const execIdx = headings.indexOf('Execution');
    const chainIdx = headings.indexOf('Chain');
    expect(execIdx).toBeGreaterThanOrEqual(0);
    expect(chainIdx).toBeGreaterThanOrEqual(0);
    expect(execIdx).toBeLessThan(chainIdx);
  });

  it('does not render any section cards when detail_view is undefined', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: { meta: { command: 'cmd' } },
      },
      isLoading: false,
    });

    renderWithSchema(
      executionSchema({ detail_view: undefined } as PluginSchema)
    );

    expect(screen.queryByRole('heading', { name: 'Execution' })).toBeNull();
  });

  it('passes DetailField.highlight through to the syntax highlighter', async () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: { meta: { command: 'SELECT 1' } },
      },
      isLoading: false,
    });

    renderWithSchema(
      executionSchema({
        detail_view: {
          sections: [
            {
              title: 'Execution',
              fields: [
                {
                  path: 'data.meta.command',
                  label: 'Command',
                  highlight: 'sql',
                },
              ],
            },
          ],
        },
      } as unknown as PluginSchema)
    );

    const hl = await screen.findByTestId('detail-syntax-highlighter');
    expect(hl.getAttribute('data-language')).toBe('sql');
    expect(hl.textContent).toBe('SELECT 1');
  });

  it('renders boolean false and numeric zero leaves', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: { meta: { command: false, args: 0 } },
      },
      isLoading: false,
    });

    renderWithSchema(executionSchema());

    expect(screen.getByText('No')).toBeInTheDocument();
    expect(screen.getByText('0')).toBeInTheDocument();
  });
});

describe('resolveTabFromSplat', () => {
  it('returns overview when splat is undefined', () => {
    expect(resolveTabFromSplat(undefined)).toBe('overview');
  });

  it('returns overview when splat is empty', () => {
    expect(resolveTabFromSplat('')).toBe('overview');
  });

  it('returns logs when splat is "logs"', () => {
    expect(resolveTabFromSplat('logs')).toBe('logs');
  });

  it('returns logs when splat has trailing slash', () => {
    expect(resolveTabFromSplat('logs/')).toBe('logs');
  });

  it('returns logs for nested logs sub-paths', () => {
    expect(resolveTabFromSplat('logs/123')).toBe('logs');
  });

  it('returns overview for non-logs paths', () => {
    expect(resolveTabFromSplat('overview')).toBe('overview');
    expect(resolveTabFromSplat('something-else')).toBe('overview');
  });
});

describe('PluginDetailPage execute flow', () => {
  it('confirms then calls execute mutation on success', async () => {
    mockExecuteMutate.mockReset();
    mockExecuteMutate.mockResolvedValue({ id: 99 });
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/FECHK');

    await userEvent.click(screen.getByTestId('plugin-task-execute'));

    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByTestId('plugin-task-execute-confirm')
    );

    await waitFor(() =>
      expect(mockExecuteMutate).toHaveBeenCalledWith({ taskName: 'FECHK' })
    );
  });

  it('shows error snackbar and closes dialog on execute failure', async () => {
    mockExecuteMutate.mockReset();
    mockExecuteMutate.mockRejectedValue(new Error('Execute failed'));
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/FECHK');

    await userEvent.click(screen.getByTestId('plugin-task-execute'));

    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByTestId('plugin-task-execute-confirm')
    );

    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    );
    await waitFor(() =>
      expect(screen.getByText('Execute failed')).toBeInTheDocument()
    );
  });

  it('closes dialog without calling execute when cancelled', async () => {
    mockExecuteMutate.mockReset();
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/FECHK');

    await userEvent.click(screen.getByTestId('plugin-task-execute'));

    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Cancel' })
    );

    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    );
    expect(mockExecuteMutate).not.toHaveBeenCalled();
  });

  it('executes a selected derived task when custom execute actions are provided', async () => {
    mockExecuteMutate.mockReset();
    mockExecuteMutate.mockResolvedValue({ id: 99 });
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'pbm-backup',
        status: 'completed',
        derived_tasks: [
          {
            name: 'pbm-backup-logical',
            backup_type: 'pbm_logical',
            status: null,
          },
        ],
      },
      isLoading: false,
    });

    render(
      <QueryClientProvider client={makeClient()}>
        <SnackbarProvider>
          <MemoryRouter initialEntries={['/apps/backup_mongo/task/pbm-backup']}>
            <Routes>
              <Route
                path="/apps/:plugin/task/:id/*"
                element={
                  <PluginDetailPage
                    schema={schema}
                    pluginName="backup_mongo"
                    getTaskExecuteActions={(task) => [
                      {
                        label: 'Sync Config',
                        taskName: String(task.name),
                        testId: 'backup-mongo-sync-config',
                      },
                      {
                        label: 'Run Logical Backup',
                        taskName: 'pbm-backup-logical',
                        testId: 'backup-mongo-logical-backup',
                      },
                    ]}
                  />
                }
              />
            </Routes>
          </MemoryRouter>
        </SnackbarProvider>
      </QueryClientProvider>
    );

    await userEvent.click(screen.getByTestId('backup-mongo-logical-backup'));

    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByTestId('plugin-task-execute-confirm')
    );

    await waitFor(() =>
      expect(mockExecuteMutate).toHaveBeenCalledWith({
        taskName: 'pbm-backup-logical',
      })
    );
  });

  it('forwards executeBody from custom execute actions to the mutation', async () => {
    mockExecuteMutate.mockReset();
    mockExecuteMutate.mockResolvedValue({ id: 99 });
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'my-alter', status: 'completed' },
      isLoading: false,
    });

    const executeBody = {
      chain_task_names: ['my-alter'],
      chain_on_failure: false,
    };

    render(
      <QueryClientProvider client={makeClient()}>
        <SnackbarProvider>
          <MemoryRouter initialEntries={['/apps/alters/task/my-alter']}>
            <Routes>
              <Route
                path="/apps/:plugin/task/:id/*"
                element={
                  <PluginDetailPage
                    schema={schema}
                    pluginName="alters"
                    getTaskExecuteActions={() => [
                      {
                        label: 'Pre-checks',
                        taskName: 'my-alter-pre-checks',
                        testId: 'alters-pre-checks-execute',
                        executeBody,
                      },
                    ]}
                  />
                }
              />
            </Routes>
          </MemoryRouter>
        </SnackbarProvider>
      </QueryClientProvider>
    );

    await userEvent.click(screen.getByTestId('alters-pre-checks-execute'));

    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByTestId('plugin-task-execute-confirm')
    );

    await waitFor(() =>
      expect(mockExecuteMutate).toHaveBeenCalledWith({
        taskName: 'my-alter-pre-checks',
        executeBody,
      })
    );
  });
});

function renderWithSchema(
  customSchema: PluginSchema,
  path = '/apps/checksums/task/FECHK'
) {
  return render(
    <QueryClientProvider client={makeClient()}>
      <SnackbarProvider>
        <MemoryRouter initialEntries={[path]}>
          <Routes>
            <Route
              path="/apps/:plugin/task/:id/*"
              element={
                <PluginDetailPage
                  schema={customSchema}
                  pluginName="checksums"
                />
              }
            />
            <Route path="/apps/:plugin" element={<div>list page</div>} />
          </Routes>
        </MemoryRouter>
      </SnackbarProvider>
    </QueryClientProvider>
  );
}

function makeSchema(
  capabilities: Record<string, boolean> | undefined
): PluginSchema {
  return {
    pluginName: 'checksums',
    display_name: 'Checksum',
    description: 'Test',
    capabilities,
    list_view: {
      columns: [
        { key: 'name', label: 'Name' },
        { key: 'status', label: 'Status', format: 'status' },
      ],
      default_sort: '-id',
    },
    formSchema: { sections: [] },
  } as unknown as PluginSchema;
}

const POPULATED_STATS = {
  engine: 'nomad',
  total: 5,
  status: { pass: 4, fail: 1 },
  duration: {
    average_seconds: 1.234,
    last_seconds: 0.987,
    total_seconds: 6.17,
  },
  last_finished_at: new Date(Date.now() - 60_000).toISOString(),
};

describe('PluginDetailPage — StatsCard integration', () => {
  beforeEach(() => {
    mockUseTaskStats.mockReset();
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    });
  });

  it('renders the StatsCard when capabilities.stats is true', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    mockUseTaskStats.mockReturnValue({
      data: POPULATED_STATS,
      isLoading: false,
      isError: false,
    });
    renderWithSchema(makeSchema({ stats: true }));
    expect(screen.getByText('Executions')).toBeInTheDocument();
  });

  it('forwards both taskName and enabled to useTaskStats', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    renderWithSchema(makeSchema({ stats: true }));
    expect(mockUseTaskStats).toHaveBeenCalled();
    const lastCall = mockUseTaskStats.mock.calls.at(-1) ?? [];
    // Guard against the prior mock that dropped every arg after the first,
    // which hid regressions around the ``enabled`` flag.
    expect(lastCall[0]).toBe('FECHK');
    expect(lastCall[1]).toBe(true);
  });

  it('does not render the StatsCard when capabilities.stats is false', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    renderWithSchema(makeSchema({ stats: false }));
    expect(screen.queryByText('Executions')).toBeNull();
    expect(mockUseTaskStats).not.toHaveBeenCalled();
  });

  it('does not render the StatsCard when capabilities.stats is absent', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    renderWithSchema(makeSchema({}));
    expect(screen.queryByText('Executions')).toBeNull();
    expect(mockUseTaskStats).not.toHaveBeenCalled();
  });

  it('does not render the StatsCard when capabilities is undefined', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    renderWithSchema(makeSchema(undefined));
    expect(screen.queryByText('Executions')).toBeNull();
    expect(mockUseTaskStats).not.toHaveBeenCalled();
  });

  it('does not render the StatsCard when task.name is missing', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, status: 'completed' },
      isLoading: false,
    });
    renderWithSchema(makeSchema({ stats: true }));
    // Hook is invoked but with undefined name, so it disables itself; the card
    // returns null before rendering the section header.
    expect(screen.queryByText('Executions')).toBeNull();
    expect(screen.queryByRole('heading', { name: 'Stats' })).toBeNull();
  });

  it('does not render the StatsCard when task.name is numeric', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 42, status: 'completed' },
      isLoading: false,
    });
    renderWithSchema(makeSchema({ stats: true }));
    expect(screen.queryByText('Executions')).toBeNull();
    expect(screen.queryByRole('heading', { name: 'Stats' })).toBeNull();
  });

  it('renders empty state when stats.total === 0', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    mockUseTaskStats.mockReturnValue({
      data: {
        engine: 'nomad',
        total: 0,
        status: { pass: 0, fail: 0 },
        duration: {
          average_seconds: null,
          last_seconds: null,
          total_seconds: null,
        },
        last_finished_at: null,
      },
      isLoading: false,
      isError: false,
    });
    renderWithSchema(makeSchema({ stats: true }));
    expect(screen.getByText('No execution history yet')).toBeInTheDocument();
    expect(screen.queryByText('Executions')).toBeNull();
  });

  it('keeps the Task information section when stats query errors', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed' },
      isLoading: false,
    });
    mockUseTaskStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error('boom'),
    });
    renderWithSchema(makeSchema({ stats: true }));
    expect(screen.getByText('Task information')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent(
      'Could not load execution stats'
    );
  });
});

describe('PluginDetailPage — Logs tab stop wiring', () => {
  beforeEach(() => {
    stopMutate.mockReset();
  });

  it('wires the Logs-tab table Stop action to the stop-task mutation with the row id', async () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'running' },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/FECHK/logs');

    await userEvent.click(
      await screen.findByRole('button', { name: 'Stop row' })
    );

    expect(stopMutate).toHaveBeenCalledWith(7);
  });
});

describe('PluginDetailPage — PII Anonymization section', () => {
  it('renders PII section with entity chips when capability is true and entities present', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        anonymized_entities: ['EMAIL_ADDRESS', 'IP_ADDRESS'],
      },
      isLoading: false,
    });

    renderWithSchema(makeSchema({ pii_anonymization: true }));

    expect(
      screen.getByRole('heading', { name: 'PII Anonymization' })
    ).toBeInTheDocument();
    const chips = screen.getAllByTestId('pii-entity-chip');
    expect(chips).toHaveLength(2);
    expect(chips[0]).toHaveTextContent('EMAIL ADDRESS');
    expect(chips[1]).toHaveTextContent('IP ADDRESS');
  });

  it('renders empty state message when capability is true but entities list is empty', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        anonymized_entities: [],
      },
      isLoading: false,
    });

    renderWithSchema(makeSchema({ pii_anonymization: true }));

    expect(
      screen.getByRole('heading', { name: 'PII Anonymization' })
    ).toBeInTheDocument();
    expect(
      screen.getByText('No PII entities configured for anonymization.')
    ).toBeInTheDocument();
  });

  it('does not render PII section when capability is false', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        anonymized_entities: ['EMAIL_ADDRESS'],
      },
      isLoading: false,
    });

    renderWithSchema(makeSchema({ pii_anonymization: false }));

    expect(
      screen.queryByRole('heading', { name: 'PII Anonymization' })
    ).toBeNull();
  });

  it('does not render PII section when capability is absent', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        anonymized_entities: ['EMAIL_ADDRESS'],
      },
      isLoading: false,
    });

    renderWithSchema(makeSchema({}));

    expect(
      screen.queryByRole('heading', { name: 'PII Anonymization' })
    ).toBeNull();
  });

  it('does not render PII section when capabilities is undefined', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        anonymized_entities: ['EMAIL_ADDRESS'],
      },
      isLoading: false,
    });

    renderWithSchema(makeSchema(undefined));

    expect(
      screen.queryByRole('heading', { name: 'PII Anonymization' })
    ).toBeNull();
  });

  it('suppresses anonymize_mask and anonymized_entities from the Task information extras', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        anonymize_mask: 2,
        anonymized_entities: ['EMAIL_ADDRESS'],
      },
      isLoading: false,
    });

    renderWithSchema(makeSchema({ pii_anonymization: true }));

    expect(screen.queryByText('Anonymize Mask')).toBeNull();
    expect(screen.queryByText('Anonymized Entities')).toBeNull();
  });
});

describe('PluginDetailPage — overview_hidden_fields', () => {
  function schemaWithHidden(overview_hidden_fields?: string[]): PluginSchema {
    return {
      pluginName: 'checksums',
      display_name: 'Checksum',
      description: 'Test',
      capabilities: {},
      list_view: {
        columns: [
          { key: 'name', label: 'Name' },
          { key: 'status', label: 'Status', format: 'status' },
        ],
        default_sort: '-name',
        ...(overview_hidden_fields !== undefined
          ? { overview_hidden_fields }
          : {}),
      },
      formSchema: { sections: [] },
    } as unknown as PluginSchema;
  }

  it('hides baseline keys (id, backend, data, etc.) when overview_hidden_fields is absent', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        backend: 'nomad',
        data: {},
        extra_visible: 'hello',
      },
      isLoading: false,
    });

    renderWithSchema(schemaWithHidden());

    expect(screen.queryByText('Id')).toBeNull();
    expect(screen.queryByText('Backend')).toBeNull();
    expect(screen.queryByText('Data')).toBeNull();
    // A non-hidden extra still renders
    expect(screen.getByText('Extra Visible')).toBeInTheDocument();
  });

  it('hides a schema-declared key in addition to the baseline', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        foo: 'secret',
        extra_visible: 'hello',
      },
      isLoading: false,
    });

    renderWithSchema(schemaWithHidden(['foo']));

    expect(screen.queryByText('Foo')).toBeNull();
    // Unrelated extra field still renders
    expect(screen.getByText('Extra Visible')).toBeInTheDocument();
  });

  it('hides entity-level overview_hidden_fields in multi-entity detail page', () => {
    function multiEntitySchemaWithHidden(
      overview_hidden_fields?: string[]
    ): PluginSchema {
      return {
        pluginName: 'inventory',
        display_name: 'Inventory',
        description: 'Test',
        capabilities: {},
        entities: [
          {
            name: 'services',
            display_name: 'Services',
            description: 'Service entities',
            forms: [],
            list_view: {
              columns: [
                { key: 'name', label: 'Name' },
                { key: 'status', label: 'Status', format: 'status' },
              ],
              default_sort: '-name',
              ...(overview_hidden_fields !== undefined
                ? { overview_hidden_fields }
                : {}),
            },
          },
        ],
        list_view: {
          columns: [{ key: 'name', label: 'Name' }],
          default_sort: '-name',
        },
        formSchema: { sections: [] },
      } as unknown as PluginSchema;
    }

    mockUsePluginEntityDetail.mockReturnValue({
      data: {
        id: 1,
        name: 'mysql-01',
        status: 'active',
        foo: 'secret',
        extra_visible: 'hello',
      },
      isLoading: false,
      error: null,
    });

    const customSchema = multiEntitySchemaWithHidden(['foo']);
    render(
      <QueryClientProvider client={makeClient()}>
        <SnackbarProvider>
          <MemoryRouter initialEntries={['/apps/inventory/services/1']}>
            <Routes>
              <Route
                path="/apps/:plugin/:entityName/:id/*"
                element={
                  <PluginDetailPage
                    schema={customSchema}
                    pluginName="inventory"
                  />
                }
              />
            </Routes>
          </MemoryRouter>
        </SnackbarProvider>
      </QueryClientProvider>
    );

    expect(screen.queryByText('Foo')).toBeNull();
    expect(screen.queryByText('foo')).toBeNull();
    // Unrelated extra field still renders
    expect(screen.getByText('extra_visible')).toBeInTheDocument();
    expect(screen.getByText('hello')).toBeInTheDocument();
  });
});

describe('PluginDetailPage — Edit affordance', () => {
  it('enables Edit and links to the edit route when the task has a stored form', () => {
    mockUsePluginTask.mockReturnValue({
      data: {
        id: 1,
        name: 'FECHK',
        status: 'completed',
        data: { _form: { task_name: 'FECHK' } },
      },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/FECHK');

    const edit = screen.getByTestId('plugin-task-edit');
    expect(edit).not.toBeDisabled();
    expect(edit).toHaveAttribute('href', '/apps/checksums/task/FECHK/edit');
  });

  it('disables Edit for a task with no stored form (legacy or legacy-form-created)', () => {
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'FECHK', status: 'completed', data: { meta: {} } },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/FECHK');

    expect(screen.getByTestId('plugin-task-edit')).toBeDisabled();
  });
});

describe('PluginDetailPage delete flow', () => {
  it('confirms then calls delete mutation and navigates to list on success', async () => {
    mockDeleteMutate.mockReset();
    mockDeleteMutate.mockResolvedValue(undefined);
    mockUsePluginTask.mockReturnValue({
      data: { id: 1, name: 'check1', status: 'completed' },
      isLoading: false,
    });

    renderAt('/apps/checksums/task/check1');

    await userEvent.click(screen.getByTestId('plugin-task-delete'));

    const dialog = await screen.findByRole('dialog');
    await userEvent.click(
      within(dialog).getByRole('button', { name: 'Delete' })
    );

    await waitFor(() =>
      expect(mockDeleteMutate).toHaveBeenCalledWith('check1')
    );
    await waitFor(() =>
      expect(screen.getByText('list page')).toBeInTheDocument()
    );
  });
});
