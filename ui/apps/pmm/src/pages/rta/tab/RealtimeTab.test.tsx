import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import RealtimeTab from './RealtimeTab';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { MemoryRouter, Route, Routes, useNavigate } from 'react-router-dom';
import { TEST_REAL_TIME_SESSION } from 'utils/testStubs';

const { getRunningSessions } = vi.hoisted(() => ({
  getRunningSessions: vi.fn().mockResolvedValue([]),
}));

vi.mock('api/rta', () => ({
  getRunningSessions,
}));

const BackButton = () => {
  const navigate = useNavigate();

  return <button onClick={() => navigate(-1)}>Back</button>;
};

const renderComponent = (initialEntries = ['/']) =>
  render(
    wrapWithQueryProvider(
      <MemoryRouter initialEntries={initialEntries}>
        <Routes>
          <Route path="/" element={<RealtimeTab />} />
          <Route
            path="/previous"
            element={<div data-testid="previous-page">Previous</div>}
          />
          <Route
            path="/rta/selection"
            element={
              <div data-testid="realtime-selection">
                Selection
                <BackButton />
              </div>
            }
          />
          <Route
            path="/rta/sessions"
            element={<div data-testid="realtime-sessions">Sessions</div>}
          />
        </Routes>
      </MemoryRouter>
    )
  );

describe('RealtimeTab', () => {
  beforeEach(() => {
    getRunningSessions.mockClear();
  });

  it('should render loading', () => {
    renderComponent();

    expect(screen.getByTestId('realtime-tab-loading')).toBeInTheDocument();
  });

  it('should navigate to selection page if no sessions are running', async () => {
    renderComponent();

    await waitFor(() => {
      expect(screen.getByTestId('realtime-selection')).toBeInTheDocument();
    });
  });

  it('replaces itself in the history, so Back leaves real-time queries', async () => {
    renderComponent(['/previous', '/']);

    fireEvent.click(await screen.findByRole('button', { name: 'Back' }));

    expect(screen.getByTestId('previous-page')).toBeInTheDocument();
  });

  it('should navigate to sessions page if sessions are running', async () => {
    getRunningSessions.mockResolvedValue([TEST_REAL_TIME_SESSION]);

    renderComponent();

    await waitFor(() => {
      expect(screen.getByTestId('realtime-sessions')).toBeInTheDocument();
    });
  });
});
