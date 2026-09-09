import { act, render, screen, waitFor } from '@testing-library/react';
import type { PropsWithChildren } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { TaskHistoryEntry, TaskHistoryStatus } from '../../hooks';
import { TaskRunDetailDrawer } from './TaskRunDetailDrawer';

// Manual factory rather than `importOriginal`: it keeps axios out of the graph,
// and the running-status set and predicate mirror the real module.
vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn() },
  SEP_BASE_PATH: '/sep',
  RUNNING_STATUSES: new Set(['running', 'pending']),
  isRunningStatus: (status: string) =>
    status === 'running' || status === 'pending',
}));

// The viewer holds an SSE connection and is covered by its own suite; this one
// is about what the drawer puts around it, and which run it hands over.
vi.mock('../TaskLogViewer', () => ({
  TaskLogViewer: ({
    taskHistoryId,
    taskStatus,
  }: {
    taskHistoryId: number | string;
    taskStatus?: string;
  }) => (
    <div
      data-testid="log-viewer"
      data-history-id={String(taskHistoryId)}
      data-task-status={taskStatus ?? ''}
    />
  ),
}));

import { apiClient } from '@sep/api';

const mockedGet = apiClient.get as unknown as ReturnType<typeof vi.fn>;

function makeEntry(
  id: number,
  status: TaskHistoryStatus,
  overrides: Partial<TaskHistoryEntry> = {}
): TaskHistoryEntry {
  const terminal = status !== 'running' && status !== 'pending';
  return {
    id,
    status,
    display_name: `backup-${id}`,
    started_at: '2026-09-07T10:00:00Z',
    finished_at: terminal ? '2026-09-07T10:02:07Z' : null,
    duration: terminal ? 127 : null,
    executed_by: 'admin',
    has_logs: true,
    log_capture: 'complete',
    task: { id, name: `backup-${id}` } as TaskHistoryEntry['task'],
    execution_request: {
      task: `backup-${id}`,
      target: 'host-1',
      meta: {},
      tracking: {},
    } as TaskHistoryEntry['execution_request'],
    ...overrides,
  };
}

function Wrapper({ children }: PropsWithChildren) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0, staleTime: 0 } },
  });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

const renderDrawer = (
  props: Partial<Parameters<typeof TaskRunDetailDrawer>[0]>
) =>
  render(<TaskRunDetailDrawer open onClose={() => {}} {...props} />, {
    wrapper: Wrapper,
  });

describe('TaskRunDetailDrawer', () => {
  beforeEach(() => {
    mockedGet.mockReset();
    mockedGet.mockResolvedValue({ data: { items: [] } });
  });

  it('is announced as a dialog named by its heading', () => {
    renderDrawer({ entry: makeEntry(11, 'success') });

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAccessibleName(/backup-11 #11/);
  });

  describe('a run handed in directly', () => {
    it('shows the run status, its duration and its log', () => {
      renderDrawer({ entry: makeEntry(12, 'success') });

      expect(screen.getByText(/backup-12 #12/)).toBeInTheDocument();
      expect(screen.getByText('Duration')).toBeInTheDocument();
      expect(screen.getByTestId('run-detail-duration')).toHaveTextContent(
        '2m 7s'
      );
      expect(screen.getByTestId('log-viewer')).toHaveAttribute(
        'data-history-id',
        '12'
      );
      // Nothing failed, so no failure block.
      expect(
        screen.queryByTestId('run-detail-failure')
      ).not.toBeInTheDocument();
      // It resolves nothing by name when the row is already in hand.
      expect(mockedGet).not.toHaveBeenCalled();
    });

    it('renders a failed run at error severity', () => {
      renderDrawer({ entry: makeEntry(13, 'failed') });

      const failure = screen.getByTestId('run-detail-failure');
      expect(failure).toHaveClass('MuiAlert-colorError');
      // No reason on the wire yet (SEP-2000), so it points at the log instead
      // of rendering an empty error.
      expect(failure).toHaveTextContent(/only record of what went wrong/);
    });

    it('shows the failure reason once the wire carries one', () => {
      // The seam SEP-2000 lands on: a reason on the row, or in the tracking
      // bag it is likely to arrive in first.
      renderDrawer({
        entry: makeEntry(14, 'failed', {
          execution_request: {
            task: 'backup-14',
            target: 'host-1',
            meta: {},
            tracking: { error_message: 'xtrabackup: no space left on device' },
          } as TaskHistoryEntry['execution_request'],
        }),
      });

      expect(screen.getByTestId('run-detail-failure')).toHaveTextContent(
        'xtrabackup: no space left on device'
      );
    });

    it('explains a terminal run that produced no readable log', () => {
      renderDrawer({
        entry: makeEntry(15, 'success', {
          has_logs: false,
          log_capture: 'incomplete',
        }),
      });

      // `log_capture` tells "produced nothing" apart from "output was lost",
      // which is the difference between a usable backup and an unknown one.
      expect(screen.getByTestId('run-detail-no-log')).toHaveTextContent(
        /lost before it could be stored/
      );
      expect(screen.queryByTestId('log-viewer')).not.toBeInTheDocument();
    });

    it('says a run was not a script failure when the executor could not launch it', () => {
      renderDrawer({ entry: makeEntry(16, 'unlaunchable') });

      expect(screen.getByTestId('run-detail-note')).toHaveTextContent(
        /not a script failure/
      );
      expect(
        screen.queryByTestId('run-detail-failure')
      ).not.toBeInTheDocument();
    });
  });

  describe('a run in flight', () => {
    beforeEach(() => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date('2026-09-07T10:00:45Z'));
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('counts elapsed time up and streams the log', () => {
      renderDrawer({ entry: makeEntry(20, 'running') });

      // The wire carries no duration for a run in flight, so the number is
      // derived from `started_at` and labelled as elapsed, not duration.
      expect(screen.getByText('Elapsed')).toBeInTheDocument();
      expect(screen.getByTestId('run-detail-duration')).toHaveTextContent(
        '45.0s'
      );

      act(() => {
        vi.advanceTimersByTime(3000);
      });

      expect(screen.getByTestId('run-detail-duration')).toHaveTextContent(
        '48.0s'
      );
      // The viewer is told the run is live, which is what makes it stream.
      expect(screen.getByTestId('log-viewer')).toHaveAttribute(
        'data-task-status',
        'running'
      );
    });

    it('offers the log for a running run that has none recorded yet', () => {
      renderDrawer({ entry: makeEntry(21, 'running', { has_logs: false }) });

      expect(screen.getByTestId('log-viewer')).toBeInTheDocument();
    });
  });

  describe('a run resolved by task name', () => {
    it('shows the newest run for the task', async () => {
      mockedGet.mockResolvedValue({
        data: {
          items: [
            makeEntry(30, 'success', { started_at: '2026-09-07T08:00:00Z' }),
            makeEntry(31, 'failed', { started_at: '2026-09-07T09:00:00Z' }),
          ],
        },
      });

      renderDrawer({ taskNames: 'backup-30' });

      // Ordering is not trusted from the response: the newest of the page wins.
      await waitFor(() => expect(screen.getByText(/#31/)).toBeInTheDocument());
      expect(screen.getByTestId('run-detail-failure')).toBeInTheDocument();
      expect(mockedGet).toHaveBeenCalledOnce();
    });

    it('opens the run that started at a given time, not the newest', async () => {
      // A schedule knows when its own last run fired but not which history row
      // that was, and a task that also runs manually will have newer rows.
      mockedGet.mockResolvedValue({
        data: {
          items: [
            makeEntry(40, 'failed', { started_at: '2026-09-07T02:00:04Z' }),
            makeEntry(41, 'success', { started_at: '2026-09-07T11:30:00Z' }),
          ],
        },
      });

      renderDrawer({ taskNames: 'backup-40', at: '2026-09-07T02:00:00Z' });

      // Nearest, not exact: `last_run_at` is when the scheduler fired, which is
      // near but not identical to the row's `started_at`.
      await waitFor(() => expect(screen.getByText(/#40/)).toBeInTheDocument());
      expect(screen.queryByText(/#41/)).not.toBeInTheDocument();
    });

    it('falls back to the newest run when the given time is unusable', async () => {
      mockedGet.mockResolvedValue({
        data: {
          items: [
            makeEntry(50, 'success', { started_at: '2026-09-07T08:00:00Z' }),
            makeEntry(51, 'success', { started_at: '2026-09-07T09:00:00Z' }),
          ],
        },
      });

      renderDrawer({ taskNames: 'backup-50', at: 'not-a-date' });

      await waitFor(() => expect(screen.getByText(/#51/)).toBeInTheDocument());
    });

    it('says so when the task has never run', async () => {
      renderDrawer({ taskNames: 'backup-99' });

      await waitFor(() =>
        expect(screen.getByTestId('run-detail-empty')).toBeInTheDocument()
      );
    });

    it('reports a failed lookup instead of rendering an empty run', async () => {
      mockedGet.mockRejectedValue(new Error('history unreachable'));

      renderDrawer({ taskNames: 'backup-98' });

      await waitFor(() =>
        expect(screen.getByTestId('run-detail-error')).toHaveTextContent(
          'history unreachable'
        )
      );
    });
  });

  describe('a run known only by id', () => {
    it('shows the log without a summary it cannot fill in', () => {
      // The post-create connectivity check reports a task_history_id and
      // nothing else, so there is no status, timing or reason to show.
      renderDrawer({ taskHistoryId: 555, taskLabel: 'Connectivity check' });

      expect(screen.getByText(/Connectivity check #555/)).toBeInTheDocument();
      expect(screen.getByTestId('log-viewer')).toHaveAttribute(
        'data-history-id',
        '555'
      );
      expect(screen.queryByText('Duration')).not.toBeInTheDocument();
      expect(screen.queryByTestId('run-detail-empty')).not.toBeInTheDocument();
      expect(mockedGet).not.toHaveBeenCalled();
    });
  });
});
