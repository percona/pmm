import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { LogLine } from 'api/logs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { lineKey } from '../utils/logDisplay';
import { LogsLinesTab } from './LogsLinesTab';

const { searchLogs, getLogContext, getLogPatterns } = vi.hoisted(() => ({
  searchLogs: vi.fn(),
  getLogContext: vi.fn(),
  getLogPatterns: vi.fn(),
}));

const { listManagedServices } = vi.hoisted(() => ({
  listManagedServices: vi.fn(),
}));

vi.mock('api/services', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/services')>()),
  listManagedServices,
}));

vi.mock('api/logs', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/logs')>()),
  searchLogs,
  getLogContext,
  getLogVolume: vi.fn(),
  getLogPatterns,
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
  patternHash: '5',
});

// Local time with milliseconds, written out independently of the page's helper.
const clock = (d: Date) =>
  `${d.toLocaleTimeString(undefined, { hour12: false, hour: '2-digit', minute: '2-digit', second: '2-digit' })}.${String(d.getMilliseconds()).padStart(3, '0')}`;

const filter = {
  start: '2026-10-17T10:00:00.000Z',
  end: '2026-10-17T11:00:00.000Z',
};

const renderTab = (props: Partial<Parameters<typeof LogsLinesTab>[0]> = {}) =>
  render(
    wrapWithQueryProvider(
      <TestWrapper>
        <LogsLinesTab filter={filter} {...props} />
      </TestWrapper>
    )
  );

const pattern = {
  pattern: 'deadlock detected on ?',
  count: 120,
  previousCount: 40,
  maxSeverityNumber: 17,
  firstSeen: '2026-10-17T10:02:00.000Z',
  lastSeen: '2026-10-17T10:58:00.000Z',
  sample: 'deadlock detected on 1',
  isNew: false,
  patternHash: '5',
  topSources: ['pg-node', 'pg-2'],
};

describe('LogsLinesTab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listManagedServices.mockResolvedValue({ services: [] });
    getLogPatterns.mockResolvedValue([pattern]);
  });

  it('shows a logfmt line by its message', async () => {
    searchLogs.mockResolvedValue({
      lines: [
        line(
          'time="2026-10-17T10:30:00.000+00:00" level=info msg="Configuration reloaded." component=victoriametrics',
          'pmm_managed'
        ),
      ],
    });
    renderTab();
    const row = await screen.findByTestId('log-line-row');
    expect(row).toHaveTextContent(
      'Configuration reloaded. component=victoriametrics'
    );
    expect(row).not.toHaveTextContent('time=');
  });

  it("shows how often the line's pattern was logged", async () => {
    searchLogs.mockResolvedValue({ lines: [line('deadlock detected on 7')] });
    renderTab();
    fireEvent.click(await screen.findByText('deadlock detected on 7'));
    const summary = await screen.findByTestId('log-pattern-summary');
    await waitFor(() => expect(summary).toHaveTextContent('120 lines'));
    expect(summary).toHaveTextContent('deadlock detected on ?');
    expect(summary).toHaveTextContent('3× the previous range');
    expect(summary).toHaveTextContent('pg-node, pg-2');
    expect(getLogPatterns).toHaveBeenCalledWith({
      ...filter,
      patternHash: '5',
    });
  });

  it('puts the opened line on the chart and takes it off when closed', async () => {
    const onFocus = vi.fn();
    const l = line('deadlock detected on 7');
    searchLogs.mockResolvedValue({ lines: [l] });
    renderTab({ onFocus, focusedKey: lineKey(l) });
    fireEvent.click(await screen.findByText('deadlock detected on 7'));
    expect(onFocus).toHaveBeenLastCalledWith(l);
    fireEvent.click(screen.getByTestId('log-line-row'));
    expect(onFocus).toHaveBeenLastCalledWith(null);
  });

  it("filters to the line's pattern or hides it", async () => {
    const onFilterChange = vi.fn();
    searchLogs.mockResolvedValue({ lines: [line('deadlock detected on 7')] });
    renderTab({ onFilterChange });
    fireEvent.click(await screen.findByText('deadlock detected on 7'));
    await screen.findByText('120 lines', { exact: false });
    fireEvent.click(
      screen.getByRole('button', { name: 'Show only this pattern' })
    );
    expect(onFilterChange).toHaveBeenLastCalledWith({
      ...filter,
      pattern: 'deadlock detected on ?',
    });
    fireEvent.click(screen.getByRole('button', { name: 'Hide this pattern' }));
    expect(onFilterChange).toHaveBeenLastCalledWith({
      ...filter,
      excludePatternHashes: ['5'],
    });
  });

  it('copies the line and a link to it', async () => {
    searchLogs.mockResolvedValue({ lines: [line('deadlock detected on 7')] });
    renderTab();
    fireEvent.click(await screen.findByText('deadlock detected on 7'));
    fireEvent.click(screen.getByRole('button', { name: 'Copy line' }));
    expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith(
      'deadlock detected on 7'
    );
    fireEvent.click(screen.getByRole('button', { name: 'Copy link' }));
    expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith(
      expect.stringContaining('/pmm-ui/logs/lines?from=')
    );
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

  it('opens the service dashboard and QAN in new tabs', async () => {
    listManagedServices.mockResolvedValue({
      services: [
        { serviceName: 'other-pg', serviceType: 'postgresql', nodeName: 'x' },
        {
          serviceName: 'pg-svc',
          serviceType: 'postgresql',
          nodeName: 'pg-node',
        },
      ],
    });
    searchLogs.mockResolvedValue({ lines: [line('pg line')] });
    renderTab();
    fireEvent.click(await screen.findByText('pg line'));
    const dashboard = await screen.findByRole('link', {
      name: 'Open dashboard',
    });
    await waitFor(() =>
      expect(dashboard).toHaveAttribute(
        'href',
        expect.stringContaining(
          'postgresql-instance-summary?var-service_name=pg-svc'
        )
      )
    );
    expect(dashboard).toHaveAttribute('target', '_blank');
    const qan = screen.getByRole('link', { name: 'Open QAN' });
    expect(qan).toHaveAttribute(
      'href',
      expect.stringContaining('/graph/d/pmm-qan/pmm-query-analytics')
    );
    expect(qan).toHaveAttribute('target', '_blank');
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

  it('shows MongoDB details in the row and attr as JSON when expanded', async () => {
    searchLogs.mockResolvedValue({
      lines: [
        {
          ...line('Applied op', 'mongodb'),
          severityNumber: 9,
          severityText: 'I',
          attributes: {
            attr: JSON.stringify({
              CRUD: { op: 'u', ns: 'config.transactions' },
              durationMillis: 1,
            }),
            component: 'REPL',
            s: 'I',
          },
        },
      ],
    });
    renderTab();
    const row = await screen.findByTestId('log-line-row');
    expect(row).toHaveTextContent(
      'Applied op · config.transactions · update · 1 ms'
    );
    fireEvent.click(row);
    const details = screen.getByTestId('log-line-details');
    expect(within(details).getByTestId('log-line-attr')).toHaveTextContent(
      '"ns": "config.transactions"'
    );
    expect(within(details).queryByText('attr')).not.toBeInTheDocument();
    expect(within(details).queryByText('s')).not.toBeInTheDocument();
    expect(within(details).getByText('component')).toBeInTheDocument();
  });

  it('shows MongoDB details in the context dialog', async () => {
    const applied = {
      ...line('Applied op', 'mongodb'),
      attributes: {
        attr: JSON.stringify({
          CRUD: { op: 'i', ns: 'shop.orders' },
          durationMillis: 2,
        }),
      },
    };
    searchLogs.mockResolvedValue({ lines: [applied] });
    getLogContext.mockResolvedValue({ lines: [applied], anchorIndex: 0 });
    renderTab();
    fireEvent.click(await screen.findByTestId('log-line-row'));
    fireEvent.click(screen.getByText('Show context'));
    expect(await screen.findByTestId('log-context-anchor')).toHaveTextContent(
      'Applied op · shop.orders · insert · 2 ms'
    );
  });

  it('shows a multi-line statement on one line in the row', async () => {
    searchLogs.mockResolvedValue({
      lines: [line('statement: \n\t\tSELECT\n\t\t  datid,\n\t\t  datname')],
    });
    renderTab();
    expect(await screen.findByTestId('log-line-row')).toHaveTextContent(
      'statement: SELECT datid, datname'
    );
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
