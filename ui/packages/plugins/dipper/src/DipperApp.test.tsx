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

import { fireEvent, render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { DipperApp } from './DipperApp';
import {
  useDipperExecution,
  useDipperFormSchema,
  useDipperHistory,
  useDipperAppSchema,
} from './hooks';

const { stopMutate } = vi.hoisted(() => ({ stopMutate: vi.fn() }));

vi.mock('notistack', () => ({
  useSnackbar: () => ({ enqueueSnackbar: vi.fn() }),
}));

vi.mock('./hooks', () => ({
  useDipperAppSchema: vi.fn(),
  useDipperFormSchema: vi.fn(),
  useDipperHistory: vi.fn(),
  useDipperExecution: vi.fn(),
}));

vi.mock('@sep/framework', async () => {
  const { useFormContext } = await import('react-hook-form');
  return {
    ServiceSelector: ({ name }: { name: string }) => {
      const { setValue } = useFormContext();
      return (
        <button
          type="button"
          onClick={() => setValue(name, { id: 7, name: 'mysql-1', type: 'mysql' })}
        >
          Select service
        </button>
      );
    },
    SchemaFormRenderer: ({ onSubmit }: { onSubmit: (values: Record<string, unknown>) => void }) => (
      <form
        onSubmit={(event) => {
          event.preventDefault();
          onSubmit({
            executor_host: 'node-1',
            sudo: false,
            n: 0,
            empty: '',
            script_preview: 'ignored',
          });
        }}
      >
        <span>Dynamic execution form</span>
        <button type="submit">Execute</button>
      </form>
    ),
    TaskHistoryTable: ({
      data,
      onViewLogs,
      onStopTask,
    }: {
      data: Array<{ id: number; status: string; task?: { name: string } }>;
      onViewLogs: (entry: unknown) => void;
      onStopTask?: (entry: { id: number }) => void;
    }) => (
      <div>
        <span>history rows: {data.length}</span>
        <button type="button" onClick={() => onViewLogs(data[0])}>
          View logs
        </button>
        {data[0] && onStopTask ? (
          <button type="button" onClick={() => onStopTask(data[0])}>
            Stop {String(data[0].id)}
          </button>
        ) : null}
      </div>
    ),
    TaskLogViewer: ({ taskHistoryId }: { taskHistoryId: number }) => (
      <div>logs for {taskHistoryId}</div>
    ),
    useStopTaskHistory: () => ({ mutate: stopMutate, isPending: false }),
  };
});

const mockAppSchema = vi.mocked(useDipperAppSchema);
const mockFormSchema = vi.mocked(useDipperFormSchema);
const mockHistory = vi.mocked(useDipperHistory);
const mockExecution = vi.mocked(useDipperExecution);

describe('DipperApp', () => {
  const mutate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockAppSchema.mockReturnValue({
      data: {
        name: 'dipper',
        display_name: 'Collect Diagnostic Data',
        description: 'Run diagnostics.',
        forms: [],
        list_view: { columns: [] },
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useDipperAppSchema>);
    mockFormSchema.mockReturnValue({
      data: {
        name: 'dipper',
        display_name: 'MySQL Dipper',
        forms: [{ title: 'Execution', fields: [] }],
        list_view: { columns: [] },
      },
      isLoading: false,
      error: null,
    } as unknown as ReturnType<typeof useDipperFormSchema>);
    mockHistory.mockReturnValue({
      data: {
        items: [{ id: 42, status: 'running', task: { name: 'run-script' } }],
        total: 1,
        offset: 0,
        limit: 100,
      },
      isLoading: false,
      error: null,
    } as ReturnType<typeof useDipperHistory>);
    mockExecution.mockReturnValue({
      mutate,
      isPending: false,
      isError: false,
      error: null,
    } as unknown as ReturnType<typeof useDipperExecution>);
  });

  it('fetches context schema after service selection', async () => {
    render(<DipperApp />);

    fireEvent.click(screen.getByRole('button', { name: 'Select service' }));

    expect(mockFormSchema).toHaveBeenLastCalledWith(7, 'environment');
    expect(await screen.findByText('Dynamic execution form')).toBeInTheDocument();
  });

  it('submits selected service and nests payload args', async () => {
    render(<DipperApp />);

    fireEvent.click(screen.getByRole('button', { name: 'Select service' }));
    fireEvent.click(screen.getByRole('button', { name: 'Execute' }));

    expect(mutate).toHaveBeenCalledWith(
      {
        service_id: 7,
        collector_type: 'environment',
        executor_host: 'node-1',
        sudo: false,
        args: { n: 0 },
      },
      expect.any(Object),
    );
  });

  it('renders history and opens logs', async () => {
    render(<DipperApp />);

    expect(screen.getByText('history rows: 1')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'View logs' }));

    expect(screen.getByText('logs for 42')).toBeInTheDocument();
  });

  it('wires the Stop button to the stop-task mutation with the row id', () => {
    render(<DipperApp />);

    fireEvent.click(screen.getByRole('button', { name: 'Stop 42' }));

    // Wired with a per-call onSuccess that refetches the dipper-keyed history,
    // since the stop hook only invalidates ['task-history'].
    expect(stopMutate).toHaveBeenCalledWith(
      42,
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });
});
