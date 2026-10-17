import { fireEvent, render, screen, within } from '@testing-library/react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { LogsPage } from '../LogsPage';

const { getLogPatterns } = vi.hoisted(() => ({ getLogPatterns: vi.fn() }));

vi.mock('api/logs', async (importOriginal) => ({
  ...(await importOriginal<typeof import('api/logs')>()),
  getLogPatterns,
  getLogVolume: vi.fn().mockResolvedValue({ buckets: [], step: '60s' }),
  getLogFacets: vi
    .fn()
    .mockResolvedValue({ nodeNames: [], sources: [], presets: [] }),
  searchLogs: vi.fn().mockResolvedValue({ lines: [] }),
  getLogContext: vi.fn(),
}));

const Location = () => {
  const { pathname, search } = useLocation();
  return <div data-testid="location">{pathname + search}</div>;
};

const renderAt = (path: string) =>
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

const pattern = (text: string, count: number, isNew: boolean) => ({
  pattern: text,
  count,
  maxSeverityNumber: 17,
  firstSeen: '2026-10-17T10:00:00Z',
  lastSeen: '2026-10-17T10:30:00Z',
  sample: text,
  isNew,
});

describe('LogsPatternsTab', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getLogPatterns.mockResolvedValue([
      pattern('deadlock detected', 12, true),
      pattern('connection ? from ?', 5, false),
    ]);
  });

  it('marks only new patterns', async () => {
    renderAt('/logs/patterns?range=1h');
    const rows = await screen.findAllByTestId('log-pattern-row');
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText('new')).toBeInTheDocument();
    expect(within(rows[1]).queryByText('new')).not.toBeInTheDocument();
  });

  it('opens the lines of a pattern', async () => {
    renderAt('/logs/patterns?range=1h&source=pg');
    const rows = await screen.findAllByTestId('log-pattern-row');
    fireEvent.click(rows[1]);
    expect(screen.getByTestId('location')).toHaveTextContent(
      '/logs/lines?range=1h&source=pg&pattern=connection+%3F+from+%3F'
    );
  });

  it('does not query patterns while the Lines tab is open', async () => {
    renderAt('/logs?range=1h');
    expect(await screen.findByTestId('logs-filter-bar')).toBeInTheDocument();
    expect(getLogPatterns).not.toHaveBeenCalled();
  });
});
