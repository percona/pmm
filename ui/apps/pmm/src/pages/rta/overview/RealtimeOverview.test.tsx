import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { wrapWithQueryProvider } from 'utils/testUtils';
import RealtimeOverview from './RealtimeOverview';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_REAL_TIME_SESSION,
  TEST_REAL_TIME_SESSION_2,
} from 'utils/testStubs';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { Messages } from './RealtimeOverview.messages';

const { exportRtaQueriesToCsv } = vi.hoisted(() => ({
  exportRtaQueriesToCsv: vi.fn(),
}));

vi.mock('./export/exportRtaQueriesToCsv', () => ({
  exportRtaQueriesToCsv,
}));

const { searchQueries, getRunningSessions } = vi.hoisted(() => ({
  searchQueries: vi.fn().mockResolvedValue({
    queries1: [],
  }),
  getRunningSessions: vi.fn().mockResolvedValue([]),
}));

vi.mock('api/rta', () => ({
  searchQueries,
  getRunningSessions,
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
          <Route
            path="/rta/selection"
            element={<div data-testid="realtime-selection">Selection</div>}
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

    getRunningSessions.mockResolvedValue([
      TEST_REAL_TIME_SESSION,
      TEST_REAL_TIME_SESSION_2,
    ]);
  });

  it('should render', async () => {
    renderComponent();

    await waitFor(() => screen.getByTestId('realtime-overview-table'));
  });

  it('should render queries', async () => {
    renderComponent();

    expect(searchQueries).toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.getAllByText(TEST_MONGO_DB_QUERY_DATA.serviceName)[0]
      ).toBeInTheDocument()
    );
  });

  it("shouldn't call api if no serviceIds are provided", async () => {
    renderComponent({ initialEntry: '/rta/overview' });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(searchQueries).not.toHaveBeenCalled();
  });

  it('should navigate to sessions page when all sessions button is clicked', async () => {
    renderComponent();

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    fireEvent.click(screen.getByTestId('overview-table-all-sessions-button'));

    expect(screen.getByTestId('realtime-sessions')).toBeInTheDocument();
  });

  it('details pane is not visible by default', async () => {
    renderComponent();

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(searchQueries).toHaveBeenCalled();

    const detailsPane = screen.queryByTestId('query-details-pane');

    expect(detailsPane).toBeInTheDocument();
    expect(detailsPane).toHaveAttribute('aria-hidden', 'true');
  });

  it('should render details pane when a query is selected', async () => {
    renderComponent();

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(searchQueries).toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.getAllByText(TEST_MONGO_DB_QUERY_DATA.serviceName)[0]
      ).toBeInTheDocument()
    );

    const serviceName = await screen.getAllByText(
      TEST_MONGO_DB_QUERY_DATA.serviceName
    )[0];
    fireEvent.click(serviceName);

    expect(screen.getByTestId('query-details-pane')).toBeInTheDocument();
  });

  it('should be paused if no services are selected', async () => {
    renderComponent({ initialEntry: '/rta/overview' });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();
    expect(
      screen.getByTestId('overview-table-resume-button')
    ).toBeInTheDocument();
    expect(screen.getByText(Messages.resume)).toBeInTheDocument();
    expect(screen.getByTestId('overview-table-resume-button')).toBeDisabled();
  });

  it('should be resumed if services are selected', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();

    expect(
      screen.getByTestId('overview-table-pause-button')
    ).toBeInTheDocument();
    expect(screen.getByText(Messages.pause)).toBeInTheDocument();
  });

  it('should be paused if services are deselected', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();

    const clearButton = await screen.findByTitle('Clear');
    fireEvent.click(clearButton);

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();
    expect(
      screen.getByTestId('overview-table-resume-button')
    ).toBeInTheDocument();
  });

  it('should pause when the button is clicked', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();

    const pauseButton = screen.getByTestId('overview-table-pause-button');
    fireEvent.click(pauseButton);

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();

    expect(
      screen.getByTestId('overview-table-resume-button')
    ).toBeInTheDocument();
    expect(screen.getByText(Messages.resume)).toBeInTheDocument();
  });

  it('should resume when the button is clicked', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();

    // First pause
    const pauseButton = screen.getByTestId('overview-table-pause-button');
    fireEvent.click(pauseButton);

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();

    // Then resume
    const resumeButton = screen.getByTestId('overview-table-resume-button');
    fireEvent.click(resumeButton);

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();

    expect(
      screen.getByTestId('overview-table-pause-button')
    ).toBeInTheDocument();
    expect(screen.getByText(Messages.pause)).toBeInTheDocument();
  });

  it('should start fetching if services are selected (from empty)', async () => {
    renderComponent({
      initialEntry: '/rta/overview',
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();

    const openButton = await screen.findByTitle('Open');
    fireEvent.click(openButton);

    const serviceOptionId =
      'service-option-' + TEST_REAL_TIME_SESSION.serviceId;
    const serviceOption = await waitFor(() =>
      screen.findByTestId(serviceOptionId)
    );
    fireEvent.click(serviceOption);

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();
    expect(
      screen.getByTestId('overview-table-pause-button')
    ).toBeInTheDocument();
  });

  it('should stay paused when changing service selection if already paused (from nonempty)', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('auto-refresh-button')).not.toBeDisabled();

    const pauseButton = screen.getByTestId('overview-table-pause-button');
    fireEvent.click(pauseButton);

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();

    const openButton = await screen.findByTitle('Open');
    fireEvent.click(openButton);

    const serviceOptionId =
      'service-option-' + TEST_REAL_TIME_SESSION_2.serviceId;
    const serviceOption = await waitFor(() =>
      screen.findByTestId(serviceOptionId)
    );
    fireEvent.click(serviceOption);

    expect(screen.getByTestId('auto-refresh-button')).toBeDisabled();
    expect(
      screen.getByTestId('overview-table-resume-button')
    ).toBeInTheDocument();
  });

  it('doesnt show refresh button if no services are selected', async () => {
    renderComponent({
      initialEntry: '/rta/overview',
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(
      screen.queryByTestId('overview-table-refresh-button')
    ).not.toBeInTheDocument();
  });

  it("doesn't show refresh button when fetching", async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(
      screen.queryByTestId('overview-table-refresh-button')
    ).not.toBeInTheDocument();
  });

  it('shows refresh button if paused', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    const pauseButton = screen.getByTestId('overview-table-pause-button');
    fireEvent.click(pauseButton);

    expect(
      screen.getByTestId('overview-table-refresh-button')
    ).toBeInTheDocument();
  });

  it('refresh button fetches queries', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    const pauseButton = screen.getByTestId('overview-table-pause-button');
    fireEvent.click(pauseButton);

    const refreshButton = screen.getByTestId('overview-table-refresh-button');
    fireEvent.click(refreshButton);

    expect(searchQueries).toHaveBeenCalled();
  });

  it('redirects to selection page if no sessions are found', async () => {
    getRunningSessions.mockResolvedValue([]);

    renderComponent({
      initialEntry: '/rta/overview',
    });

    await waitFor(() =>
      expect(screen.getByTestId('realtime-selection')).toBeInTheDocument()
    );
  });

  it('disables export while live updates are running', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(screen.getByTestId('overview-table-export-button')).toBeDisabled();
  });

  it('enables export after pausing live updates', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    fireEvent.click(screen.getByTestId('overview-table-pause-button'));

    await waitFor(() =>
      expect(screen.getByTestId('overview-table-export-button')).not.toBeDisabled()
    );
  });

  it('exports visible rows when export is clicked', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    fireEvent.click(screen.getByTestId('overview-table-pause-button'));

    fireEvent.click(screen.getByTestId('overview-table-export-button'));

    expect(exportRtaQueriesToCsv).toHaveBeenCalledWith([
      expect.objectContaining({
        queryId: TEST_MONGO_DB_QUERY_DATA.queryId,
        serviceName: TEST_MONGO_DB_QUERY_DATA.serviceName,
        queryText: TEST_MONGO_DB_QUERY_DATA.queryText,
        queryExecutionDurationMs: 10,
      }),
    ]);
  });
});
