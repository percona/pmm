import { render, screen } from '@testing-library/react';
import { Route, Routes } from 'react-router-dom';
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

const renderPage = (path = '/logs') =>
  render(
    wrapWithQueryProvider(
      <TestWrapper routerProps={{ initialEntries: [path] }}>
        <Routes>
          <Route path="logs/:tab?" element={<LogsPage />} />
        </Routes>
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
    getLogFacets.mockResolvedValue({ nodeNames: [], sources: [], presets: [] });
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

  it('selects the Patterns tab from the path', async () => {
    renderPage('/logs/patterns');
    expect(await screen.findByTestId('logs-tab-patterns')).toHaveAttribute(
      'aria-selected',
      'true'
    );
  });

  it('explains that label-based access filters block logs', async () => {
    getLogVolume.mockRejectedValue(apiError(7, 'denied'));
    renderPage();
    expect(
      await screen.findByText(
        'Logs are not available to users with label-based access filters.'
      )
    ).toBeInTheDocument();
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
    getLogVolume.mockResolvedValue({ buckets: [], step: '60s' });
    renderPage();
    expect(await screen.findByTestId('logs-empty-state')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /settings/i })).toHaveAttribute(
      'href',
      '/settings/otel'
    );
  });
});
