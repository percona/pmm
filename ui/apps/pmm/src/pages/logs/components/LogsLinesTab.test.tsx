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

// Local time with milliseconds, written out independently of the page's helper.
const clock = (d: Date) =>
  `${d.toLocaleTimeString(undefined, { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })}.${String(d.getMilliseconds()).padStart(3, '0')}`;

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
    expect(within(pg).getByText('pid')).toBeInTheDocument();
    expect(within(pg).getByText('42')).toBeInTheDocument();
    expect(within(pg).queryByText('user')).not.toBeInTheDocument();

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

  it('shows context lines with the time of day and the level name', async () => {
    const mongo = {
      ...line('mongo line', 'mongod'),
      severityNumber: 9,
      severityText: 'I',
    };
    searchLogs.mockResolvedValue({ lines: [mongo] });
    getLogContext.mockResolvedValue({ lines: [mongo], anchorIndex: 0 });
    renderTab();
    fireEvent.click(await screen.findByText('mongo line'));
    fireEvent.click(screen.getByText('Show context'));
    const anchor = await screen.findByTestId('log-context-anchor');
    expect(anchor).toHaveTextContent(clock(new Date(mongo.time)));
    expect(within(anchor).getByText('INFO')).toBeInTheDocument();
  });

  it('names the level the same way for every source', async () => {
    searchLogs.mockResolvedValue({
      lines: [
        {
          ...line('mongo line', 'mongod'),
          severityNumber: 9,
          severityText: 'I',
        },
      ],
    });
    renderTab();
    const row = await screen.findByTestId('log-line-row');
    expect(within(row).getByText('INFO')).toBeInTheDocument();
    expect(within(row).queryByText('I')).not.toBeInTheDocument();
    fireEvent.click(row);
    const details = screen.getByTestId('log-line-details');
    expect(within(details).getByText('original level')).toBeInTheDocument();
    expect(within(details).getByText('I')).toBeInTheDocument();
  });

  it('shows the time of day and the date once per day', async () => {
    searchLogs.mockResolvedValue({
      lines: [
        line('late'),
        line('early'),
        { ...line('day before'), time: '2026-10-15T10:30:00.000Z' },
      ],
    });
    renderTab();
    const rows = await screen.findAllByTestId('log-line-row');
    const time = new Date('2026-10-17T10:30:00.000Z');
    expect(rows[0]).toHaveTextContent(clock(time));
    const days = screen.getAllByTestId('log-day-separator');
    expect(days).toHaveLength(2);
    expect(days[0]).toHaveTextContent(
      time.toLocaleDateString(undefined, {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })
    );
  });

  it('names the source only when it differs from the node', async () => {
    searchLogs.mockResolvedValue({
      lines: [line('same'), { ...line('other'), source: 'pmm-agent' }],
    });
    renderTab();
    const rows = await screen.findAllByTestId('log-line-row');
    expect(within(rows[0]).getByTestId('log-line-origin')).toHaveTextContent(
      /^pg-node$/
    );
    expect(within(rows[1]).getByTestId('log-line-origin')).toHaveTextContent(
      'pg-node · pmm-agent'
    );
  });
});
