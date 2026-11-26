import { QueryClient } from '@tanstack/react-query';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { deleteDumps, getDumpLogs, listDumps } from 'api/dump';
import { type Dump, DumpStatus } from 'types/dump.types';
import { TEST_DUMP, TEST_USER_VIEWER } from 'utils/testStubs';
import {
  wrapWithQueryProvider,
  wrapWithSnackbarProvider,
} from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { DumpPage } from './DumpPage';

vi.mock('api/dump');

const dump = (dumpId: string, overrides: Partial<Dump> = {}): Dump => ({
  ...TEST_DUMP,
  dumpId,
  ...overrides,
});

const READY = dump('ready', { createdAt: '2026-10-27T06:40:00Z' });
const CREATING = dump('creating', {
  status: DumpStatus.InProgress,
  serviceNames: [],
  createdAt: '2026-10-27T07:00:00Z',
});
const FAILED = dump('failed', {
  status: DumpStatus.Error,
  createdAt: '2026-10-26T06:00:00Z',
});

const renderPage = (
  dumps: Dump[],
  userContext?: Parameters<typeof TestWrapper>[0]['userContext']
) => {
  vi.mocked(listDumps).mockResolvedValue({ dumps });
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  render(
    <TestWrapper userContext={userContext}>
      {wrapWithQueryProvider(wrapWithSnackbarProvider(<DumpPage />), client)}
    </TestWrapper>
  );

  return { client };
};

const row = (dumpId: string) => screen.findByTestId(`dump-row-${dumpId}`);

const selectRows = async (...dumpIds: string[]) => {
  for (const dumpId of dumpIds) {
    fireEvent.click(
      within(await row(dumpId)).getByRole('checkbox', {
        name: 'Toggle select row',
      })
    );
  }
};

const hover = async (element: HTMLElement) => {
  fireEvent.mouseOver(element.parentElement as HTMLElement);
  return screen.findByRole('tooltip');
};

describe('DumpPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-27T08:00:00Z'));
    vi.mocked(deleteDumps).mockResolvedValue({});
    vi.mocked(getDumpLogs).mockResolvedValue({ logs: [], end: true });
  });

  afterEach(() => vi.useRealTimers());

  it('shows the empty state when there are no datasets', async () => {
    renderPage([]);

    expect(
      await screen.findByRole('heading', { name: 'No datasets yet' })
    ).toBeVisible();
    expect(screen.queryByTestId(/^dump-row-/)).not.toBeInTheDocument();
  });

  it('lists datasets with status, services, data window and created time', async () => {
    renderPage([READY, CREATING]);

    const ready = await row('ready');
    expect(ready).toHaveTextContent('Ready');
    expect(ready).toHaveTextContent('mysql-1');
    expect(ready).toHaveTextContent('Oct 26, 18:39 – Oct 27, 06:39');
    expect(ready).toHaveTextContent('1 hour ago');
    expect(await row('creating')).toHaveTextContent('All services');
  });

  it('orders datasets newest first by default', async () => {
    renderPage([FAILED, READY, CREATING]);
    await row('ready');

    expect(
      screen
        .getAllByTestId(/^dump-row-/)
        .map((element) => element.dataset.testid)
    ).toEqual(['dump-row-creating', 'dump-row-ready', 'dump-row-failed']);
  });

  it('disables Download with a reason until the dataset is ready', async () => {
    renderPage([CREATING]);
    const download = within(await row('creating')).getByTestId(
      'dump-download-creating'
    );

    expect(download).toBeDisabled();
    expect(await hover(download)).toHaveTextContent(
      'Available once the dataset is ready'
    );
  });

  it('disables Delete with a reason while the dataset is being created', async () => {
    renderPage([CREATING]);

    fireEvent.click(
      within(await row('creating')).getByRole('button', {
        name: 'Row Actions',
      })
    );
    const remove = await screen.findByRole('menuitem', { name: /Delete/ });

    expect(remove).toHaveAttribute('aria-disabled', 'true');
    expect(remove).toHaveTextContent('Available once the export finishes');
  });

  it('opens the logs from the Failed status', async () => {
    renderPage([FAILED]);

    fireEvent.click(
      within(await row('failed')).getByRole('button', {
        name: 'See what went wrong',
      })
    );

    expect(await screen.findByTestId('dump-logs-modal')).toBeVisible();
  });

  it('counts only datasets still in the list as selected', async () => {
    const { client } = renderPage([READY, FAILED]);
    await selectRows('ready');
    expect(screen.getByTestId('dump-selected-count')).toHaveTextContent(
      '1 selected'
    );

    act(() => client.setQueryData(['dumps:list'], { dumps: [FAILED] }));

    await waitFor(() =>
      expect(
        screen.queryByTestId('dump-selected-count')
      ).not.toBeInTheDocument()
    );
  });

  it('explains disabled bulk actions', async () => {
    renderPage([READY, CREATING]);
    await selectRows('ready', 'creating');
    const download = screen.getByTestId('dump-download-selected');

    expect(download).toBeDisabled();
    expect(await hover(download)).toHaveTextContent(
      "1 of 2 selected datasets isn't ready"
    );
  });

  it('disables Create dataset while a dataset is being created', async () => {
    renderPage([CREATING]);
    await row('creating');
    const create = screen.getByTestId('create-dataset');

    expect(create).toHaveAttribute('aria-disabled', 'true');
    expect(await hover(create)).toHaveTextContent(
      'Another dataset is being created. You can create one when it finishes.'
    );
  });

  it('deletes selected datasets after confirming', async () => {
    renderPage([READY, FAILED]);
    await selectRows('ready', 'failed');

    fireEvent.click(screen.getByTestId('dump-delete-selected'));

    const modal = await screen.findByTestId('dump-delete-modal');
    expect(
      within(modal).getByTestId('dump-delete-modal-title')
    ).toHaveTextContent('Delete 2 datasets?');
    fireEvent.click(within(modal).getByTestId('dump-delete-modal-confirm'));

    await waitFor(() =>
      expect(vi.mocked(deleteDumps).mock.calls[0][0]).toEqual({
        dumpIds: ['ready', 'failed'],
      })
    );
    expect(
      await screen.findByText('Datasets deleted successfully.')
    ).toBeVisible();
  });

  it('warns before a bulk download', async () => {
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    renderPage([READY, dump('ready-2')]);
    await selectRows('ready', 'ready-2');

    fireEvent.click(screen.getByTestId('dump-download-selected'));

    expect(await screen.findByText(/^Downloading 2 archives/)).toBeVisible();
    await waitFor(() => expect(click).toHaveBeenCalledTimes(2), {
      timeout: 2000,
    });
    click.mockRestore();
  });

  it('shows the unauthorized state to non-admin users', () => {
    renderPage([READY], { user: TEST_USER_VIEWER, isLoading: false });

    expect(screen.getByTestId('unauthorized')).toBeInTheDocument();
    expect(screen.queryByTestId('pmm-dump-page')).not.toBeInTheDocument();
    expect(listDumps).not.toHaveBeenCalled();
  });
});
