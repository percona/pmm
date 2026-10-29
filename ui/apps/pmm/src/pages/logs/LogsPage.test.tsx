import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { LogsPage } from './LogsPage';

const { getLogVolume, getLogFacets, searchLogs, getLogPatterns } = vi.hoisted(
  () => ({
    getLogVolume: vi.fn(),
    getLogFacets: vi.fn(),
    searchLogs: vi.fn(),
    getLogPatterns: vi.fn(),
  })
);

vi.mock('api/logs', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/logs')>()),
  getLogVolume,
  getLogFacets,
  searchLogs,
  getLogPatterns,
  getLogContext: vi.fn(),
}));

const apiError = (code: number, message: string) => ({
  isAxiosError: true,
  response: { status: code === 7 ? 403 : 504, data: { code, message } },
});

const Location = () => {
  const { search } = useLocation();
  return <div data-testid="location">{search}</div>;
};

const renderPage = (path = '/logs') =>
  render(
    wrapWithQueryProvider(
      <TestWrapper routerProps={{ initialEntries: [path] }}>
        <Routes>
          <Route path="logs/:tab?" element={<LogsPage />} />
        </Routes>
        <Location />
      </TestWrapper>
    )
  );

describe('LogsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getLogVolume.mockResolvedValue({
      buckets: [
        {
          start: '2026-10-17T10:00:00Z',
          error: 1,
          warning: 0,
          info: 5,
          other: 0,
        },
      ],
      step: '60s',
    });
    getLogFacets.mockResolvedValue({
      nodeNames: [],
      sources: [],
      presets: [],
      clusters: [],
      environments: [],
      replicationSets: [],
    });
    searchLogs.mockResolvedValue({ lines: [] });
    getLogPatterns.mockResolvedValue([]);
  });

  it('renders the filter bar and the Lines tab', async () => {
    renderPage();
    expect(await screen.findByTestId('logs-filter-bar')).toBeInTheDocument();
    expect(screen.getByTestId('logs-tab-lines')).toHaveAttribute(
      'aria-selected',
      'true'
    );
    expect(await screen.findByTestId('logs-volume-chart')).toBeInTheDocument();
  });

  it('has no Logs heading but keeps the document title', async () => {
    renderPage();
    await screen.findByTestId('logs-filter-bar');
    expect(screen.queryByRole('heading', { name: 'Logs' })).toBeNull();
    expect(document.title).toContain('Logs');
  });

  it('follows the opened line on the chart until its chip is cleared', async () => {
    searchLogs.mockResolvedValue({
      lines: [
        {
          time: '2026-10-17T10:00:30.000Z',
          nodeName: 'pg-node',
          source: 'pg-node',
          preset: 'postgres',
          fileName: 'postgresql.log',
          severityNumber: 17,
          severityText: 'ERROR',
          body: 'deadlock detected on 7',
          bodyTruncated: false,
          attributes: {},
          patternHash: '5',
        },
      ],
    });
    getLogPatterns.mockResolvedValue([
      {
        pattern: 'deadlock detected on ?',
        count: 3,
        previousCount: 0,
        maxSeverityNumber: 17,
        firstSeen: '2026-10-17T10:00:30.000Z',
        lastSeen: '2026-10-17T10:00:30.000Z',
        sample: 'deadlock detected on 7',
        isNew: true,
        patternHash: '5',
        topSources: ['pg-node'],
      },
    ]);
    renderPage('/logs?range=1h');
    fireEvent.click(await screen.findByText('deadlock detected on 7'));
    await waitFor(() =>
      expect(getLogVolume).toHaveBeenCalledWith(
        expect.objectContaining({ patternHash: '5' })
      )
    );
    const chip = await screen.findByTestId('logs-volume-overlay-chip');
    expect(chip).toHaveTextContent('Showing: deadlock detected on ?');
    fireEvent.click(
      chip.querySelector('[data-testid="CancelIcon"]') as Element
    );
    expect(
      screen.queryByTestId('logs-volume-overlay-chip')
    ).not.toBeInTheDocument();
  });

  it('hides a pattern from a line', async () => {
    searchLogs.mockResolvedValue({
      lines: [
        {
          time: '2026-10-17T10:00:30.000Z',
          nodeName: 'pg-node',
          source: 'pg-node',
          preset: 'postgres',
          fileName: 'postgresql.log',
          severityNumber: 17,
          severityText: 'ERROR',
          body: 'deadlock detected on 7',
          bodyTruncated: false,
          attributes: {},
          patternHash: '5',
        },
      ],
    });
    renderPage('/logs?range=1h');
    fireEvent.click(await screen.findByText('deadlock detected on 7'));
    fireEvent.click(
      await screen.findByRole('button', { name: 'Hide this pattern' })
    );
    expect(screen.getByTestId('location')).toHaveTextContent('hide=5');
  });

  it('counts the matching lines on the Lines tab', async () => {
    renderPage();
    await screen.findByTestId('logs-volume-chart');
    expect(screen.getByTestId('logs-tab-lines')).toHaveTextContent('Lines 6');
  });

  it('goes back to the range before a zoom', async () => {
    renderPage('/logs?range=1h');
    expect(
      screen.queryByRole('button', { name: 'Back to previous range' })
    ).not.toBeInTheDocument();
    const [bucket] = await screen.findAllByTestId('logs-volume-bucket');
    fireEvent.mouseDown(bucket);
    fireEvent.mouseUp(bucket);
    expect(screen.getByTestId('location')).toHaveTextContent('from=');
    fireEvent.click(
      await screen.findByRole('button', { name: 'Back to previous range' })
    );
    expect(await screen.findByTestId('location')).toHaveTextContent(
      /^\?range=1h$/
    );
    expect(
      screen.queryByRole('button', { name: 'Back to previous range' })
    ).not.toBeInTheDocument();
  });

  it('selects the Patterns tab from the path', async () => {
    renderPage('/logs/patterns');
    expect(await screen.findByTestId('logs-tab-patterns')).toHaveAttribute(
      'aria-selected',
      'true'
    );
  });

  it('explains that label-based access filters block logs', async () => {
    getLogVolume.mockRejectedValue(
      apiError(
        7,
        'Logs are not available to users with label-based access filters.'
      )
    );
    renderPage();
    expect(
      await screen.findByText(
        'Logs are not available to users with label-based access filters.'
      )
    ).toBeInTheDocument();
  });

  it('does not blame access filters for other denials', async () => {
    // nginx's answer when the user's role is not allowed.
    getLogVolume.mockRejectedValue(apiError(7, 'Access denied'));
    renderPage();
    expect(await screen.findByText('Access denied')).toBeInTheDocument();
    expect(
      screen.queryByText(
        'Logs are not available to users with label-based access filters.'
      )
    ).not.toBeInTheDocument();
  });

  it('shows the server message on a timeout', async () => {
    getLogVolume.mockRejectedValue(
      apiError(
        4,
        'The query took longer than 30 s; narrow the time range or add filters.'
      )
    );
    renderPage();
    expect(
      await screen.findByText(
        'The query took longer than 30 s; narrow the time range or add filters.'
      )
    ).toBeInTheDocument();
  });

  it('points to log collection settings when the range is empty', async () => {
    // The API returns every bucket of the range, empty ones included.
    getLogVolume.mockResolvedValue({
      buckets: [
        {
          start: '2026-10-17T10:00:00Z',
          error: 0,
          warning: 0,
          info: 0,
          other: 0,
        },
        {
          start: '2026-10-17T10:01:00Z',
          error: 0,
          warning: 0,
          info: 0,
          other: 0,
        },
      ],
      step: '60s',
    });
    renderPage();
    expect(await screen.findByTestId('logs-empty-state')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /settings/i })).toHaveAttribute(
      'href',
      '/settings/otel'
    );
  });
});
