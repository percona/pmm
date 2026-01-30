import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { wrapWithQueryProvider } from 'utils/testUtils';
import RealtimeOverview from './RealtimeOverview';
import { TEST_MONGO_DB_QUERY_DATA } from 'utils/testStubs';
import { MemoryRouter, Route, Routes } from 'react-router-dom';

const { searchQueries } = vi.hoisted(() => ({
  searchQueries: vi.fn().mockResolvedValue({
    queries1: [],
  }),
}));

vi.mock('api/rta', () => ({
  searchQueries,
}));

const renderComponent = ({
  initialEntry = '/rta/overview?serviceIds=123',
}: {
  initialEntry?: string;
} = {}) =>
  render(
    wrapWithQueryProvider(
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route path="/rta/overview" element={<RealtimeOverview />} />
          <Route
            path="/rta/sessions"
            element={<div data-testid="realtime-sessions">Sessions</div>}
          />
        </Routes>
      </MemoryRouter>
    )
  );

describe('RealtimeOverview', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    searchQueries.mockResolvedValue({
      queries: [TEST_MONGO_DB_QUERY_DATA],
    });
  });

  it('should render', () => {
    renderComponent();
  });

  it('should render queries', async () => {
    renderComponent();

    expect(searchQueries).toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.getByText(TEST_MONGO_DB_QUERY_DATA.serviceName)
      ).toBeInTheDocument()
    );
  });

  it("shouldn't call api if no serviceIds are provided", () => {
    renderComponent({ initialEntry: '/rta/overview' });

    expect(searchQueries).not.toHaveBeenCalled();
  });

  it('should navigate to sessions page when all sessions button is clicked', () => {
    renderComponent();

    fireEvent.click(screen.getByTestId('overview-table-all-sessions-button'));

    expect(screen.getByTestId('realtime-sessions')).toBeInTheDocument();
  });

  it('details pane is not visible by default', () => {
    renderComponent();

    expect(searchQueries).toHaveBeenCalled();

    const detailsPane = screen.queryByTestId('query-details-pane');

    expect(detailsPane).toBeInTheDocument();
    expect(detailsPane).toHaveAttribute('aria-hidden', 'true');
  });

  it('should render details pane when a query is selected', async () => {
    renderComponent();

    expect(searchQueries).toHaveBeenCalled();

    const serviceName = await screen.findByText(
      TEST_MONGO_DB_QUERY_DATA.serviceName
    );
    fireEvent.click(serviceName);

    expect(screen.getByTestId('query-details-pane')).toBeInTheDocument();
  });

  it('should open details pane through row action', async () => {
    renderComponent();

    const rowAction = await screen.findByTestId('open-query-details');
    fireEvent.click(rowAction);

    expect(screen.getByTestId('query-details-pane')).toBeInTheDocument();
  });
});
