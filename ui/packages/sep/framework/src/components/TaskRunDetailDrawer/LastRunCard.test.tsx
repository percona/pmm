import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { PropsWithChildren } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { TaskHistoryEntry, TaskHistoryStatus } from '../../hooks';
import { LastRunCard } from './LastRunCard';

vi.mock('@sep/api', () => ({
  apiClient: { get: vi.fn() },
  SEP_BASE_PATH: '/sep',
  RUNNING_STATUSES: new Set(['running', 'pending']),
  isRunningStatus: (status: string) =>
    status === 'running' || status === 'pending',
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

const renderCard = (onOpenRun = vi.fn()) => {
  render(<LastRunCard taskNames="backup-1" onOpenRun={onOpenRun} />, {
    wrapper: Wrapper,
  });
  return onOpenRun;
};

describe('LastRunCard', () => {
  beforeEach(() => {
    mockedGet.mockReset();
    mockedGet.mockResolvedValue({ data: { items: [] } });
  });

  it('reports a failed run at error severity, with its reason', async () => {
    mockedGet.mockResolvedValue({
      data: {
        items: [
          makeEntry(9, 'failed', {
            execution_request: {
              task: 'backup-1',
              target: 'host-1',
              meta: {},
              tracking: {
                error_message:
                  'xtrabackup: no space left on device\n  at /var/backups',
              },
            } as TaskHistoryEntry['execution_request'],
          }),
        ],
      },
    });

    renderCard();

    const failure = await screen.findByTestId('last-run-card-failure');
    expect(failure).toHaveClass('MuiAlert-colorError');
    // One line here; the traceback's remainder lives in the execution detail,
    // so the card keeps its height whatever the backend sends.
    expect(failure).toHaveTextContent('xtrabackup: no space left on device');
    expect(failure).not.toHaveTextContent('/var/backups');
  });

  it('survives a reload by fetching the run rather than being handed it', async () => {
    // The Overview's only failure signal used to be a banner riding router
    // state, which was gone after a refresh. This asks the server instead.
    mockedGet.mockResolvedValue({ data: { items: [makeEntry(9, 'failed')] } });

    renderCard();

    await waitFor(() => expect(mockedGet).toHaveBeenCalledOnce());
    expect(
      await screen.findByTestId('last-run-card-failure')
    ).toHaveTextContent(/open the log/);
  });

  it('shows the duration of a finished run', async () => {
    mockedGet.mockResolvedValue({ data: { items: [makeEntry(8, 'success')] } });

    renderCard();

    expect(await screen.findByText('2m 7s')).toBeInTheDocument();
    expect(screen.getByText('Duration')).toBeInTheDocument();
  });

  it('labels an in-flight run as elapsed rather than showing no duration', async () => {
    // The wire leaves `duration` null until a run finishes, so the card
    // derives the number from `started_at` instead of rendering an em-dash.
    // The tick itself is covered by useElapsedSeconds and the drawer suite.
    mockedGet.mockResolvedValue({
      data: {
        items: [
          makeEntry(7, 'running', {
            started_at: new Date(Date.now() - 45_000).toISOString(),
          }),
        ],
      },
    });

    renderCard();

    expect(await screen.findByText('Elapsed')).toBeInTheDocument();
    expect(screen.queryByText('Duration')).not.toBeInTheDocument();
    expect(screen.getByText(/^4[45]\.\ds$/)).toBeInTheDocument();
  });

  it('says the task has not run yet rather than showing an empty run', async () => {
    renderCard();

    expect(await screen.findByTestId('last-run-card-empty')).toHaveTextContent(
      'This task has not run yet.'
    );
  });

  it('reports a failed lookup', async () => {
    mockedGet.mockRejectedValue(new Error('history unreachable'));

    renderCard();

    expect(await screen.findByTestId('last-run-card-error')).toHaveTextContent(
      'history unreachable'
    );
  });

  it('asks the caller to open the run in one click', async () => {
    // No run is handed back: the caller opens the detail by task name so it
    // keeps following the newest run rather than freezing this row.
    mockedGet.mockResolvedValue({ data: { items: [makeEntry(6, 'success')] } });
    const onOpenRun = renderCard();

    await userEvent.click(await screen.findByTestId('last-run-card-view'));

    expect(onOpenRun).toHaveBeenCalledOnce();
    expect(onOpenRun).toHaveBeenCalledWith();
  });
});
