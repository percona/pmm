import { fireEvent, render, screen, within } from '@testing-library/react';
import { LogLine } from 'api/logs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { LogsLinesTab } from './LogsLinesTab';

const { searchLogs, getLogContext } = vi.hoisted(() => ({
  searchLogs: vi.fn(),
  getLogContext: vi.fn(),
}));

vi.mock('api/logs', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/logs')>()),
  searchLogs,
  getLogContext,
  getLogVolume: vi.fn(),
  getLogPatterns: vi.fn(),
  getLogFacets: vi.fn(),
}));

const line = (body: string, preset = 'postgres'): LogLine => ({
  time: '2026-10-17T10:30:00.000Z',
  nodeName: 'pg-node',
  source: 'pg-node',
  preset,
  fileName: 'postgresql.log',
  severityNumber: 17,
  severityText: 'ERROR',
  body,
  bodyTruncated: false,
  attributes: { pid: '42', user: '' },
});

const filter = {
  start: '2026-10-17T10:00:00.000Z',
  end: '2026-10-17T11:00:00.000Z',
};

const renderTab = () =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <LogsLinesTab filter={filter} />
      </TestWrapper>
    )
  );

describe('LogsLinesTab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('loads older lines page by page', async () => {
    searchLogs
      .mockResolvedValueOnce({ lines: [line('first page')], nextCursor: 'c1' })
      .mockResolvedValueOnce({ lines: [line('second page')] });
    renderTab();
    expect(await screen.findByText('first page')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Load older' }));
    expect(await screen.findByText('second page')).toBeInTheDocument();
    expect(searchLogs).toHaveBeenLastCalledWith(filter, 'c1');
    expect(
      screen.queryByRole('button', { name: 'Load older' })
    ).not.toBeInTheDocument();
  });

  it('shows actions and offers QAN only for database logs', async () => {
    searchLogs.mockResolvedValue({
      lines: [line('pg line'), line('nginx line', 'nginx_access')],
    });
    renderTab();
    fireEvent.click(await screen.findByText('pg line'));
    const pg = screen.getByTestId('log-line-details');
    expect(within(pg).getByText('Show context')).toBeInTheDocument();
    expect(within(pg).getByText('Ask Assistant')).toBeInTheDocument();
    expect(within(pg).getByText('Open dashboard')).toBeInTheDocument();
    expect(within(pg).getByText('Open QAN')).toBeInTheDocument();
    expect(within(pg).getByText('pid: 42')).toBeInTheDocument();
    expect(within(pg).queryByText(/^user:/)).not.toBeInTheDocument();

    const rows = screen.getAllByTestId('log-line-row');
    fireEvent.click(rows[0]);
    fireEvent.click(rows[1]);
    const nginx = screen.getByTestId('log-line-details');
    expect(within(nginx).queryByText('Open QAN')).not.toBeInTheDocument();
  });

  it('highlights the anchor line in the context dialog', async () => {
    searchLogs.mockResolvedValue({ lines: [line('the line')] });
    // Two lines share the clicked line's millisecond; the server anchors the first, the dialog the clicked one.
    getLogContext.mockResolvedValue({
      lines: [
        { ...line('before'), time: '2026-10-17T10:29:59.000Z' },
        line('same millisecond'),
        line('the line'),
        { ...line('after'), time: '2026-10-17T10:30:01.000Z' },
      ],
      anchorIndex: 1,
    });
    renderTab();
    fireEvent.click(await screen.findByText('the line'));
    fireEvent.click(screen.getByText('Show context'));
    const anchor = await screen.findByTestId('log-context-anchor');
    expect(anchor).toHaveTextContent('the line');
    expect(getLogContext).toHaveBeenCalledWith(
      '2026-10-17T10:30:00.000Z',
      'pg-node',
      'postgresql.log',
      'pg-node'
    );
    expect(screen.getAllByTestId('log-context-line')).toHaveLength(3);
  });
});
