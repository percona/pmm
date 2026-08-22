import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { wrapWithQueryProvider } from 'utils/testUtils';
import RealtimeOverview from './RealtimeOverview';
import {
  TEST_MONGO_DB_QUERY_DATA,
  TEST_MYSQL_QUERY_DATA,
  TEST_RAW_MONGO_DB_QUERY_DATA,
  TEST_RAW_MYSQL_QUERY_DATA,
  TEST_REAL_TIME_SESSION,
  TEST_REAL_TIME_SESSION_2,
  TEST_REAL_TIME_SESSION_MYSQL,
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

// The overview derives the technology of the selection by matching the URL's
// serviceIds against the running sessions, so a test that cares about the
// technology has to line those up.
const renderMySqlSelection = () => {
  getRunningSessions.mockResolvedValue([TEST_REAL_TIME_SESSION_MYSQL]);
  // A MySQL selection must be fed MySQL queries, or the columns under test are
  // resolved from the MongoDB payload and prove nothing about mySqlPayload.
  searchQueries.mockResolvedValue({ queries: [TEST_RAW_MYSQL_QUERY_DATA] });

  return renderComponent({
    initialEntry: `/rta/overview?serviceIds=${TEST_REAL_TIME_SESSION_MYSQL.serviceId}`,
  });
};

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
      queries: [TEST_RAW_MONGO_DB_QUERY_DATA],
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

  it('should hide the database and user columns by default', async () => {
    renderMySqlSelection();

    await waitFor(() =>
      screen.getByTestId(`query-${TEST_MYSQL_QUERY_DATA.queryId}-host-cell`)
    );

    expect(
      screen.queryByTestId(
        `query-${TEST_MYSQL_QUERY_DATA.queryId}-database-cell`
      )
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTestId(`query-${TEST_MYSQL_QUERY_DATA.queryId}-user-cell`)
    ).not.toBeInTheDocument();
  });

  it('should render database and user columns from the MySQL payload once revealed', async () => {
    renderMySqlSelection();

    await waitFor(() =>
      screen.getByTestId(`query-${TEST_MYSQL_QUERY_DATA.queryId}-host-cell`)
    );

    fireEvent.click(screen.getByLabelText('Show/Hide columns'));
    fireEvent.click(await screen.findByLabelText('Database'));
    fireEvent.click(screen.getByLabelText('User'));

    await waitFor(() =>
      expect(
        screen.getByTestId(
          `query-${TEST_MYSQL_QUERY_DATA.queryId}-database-cell`
        )
      ).toHaveTextContent('mysql-database')
    );
    expect(
      screen.getByTestId(`query-${TEST_MYSQL_QUERY_DATA.queryId}-user-cell`)
    ).toHaveTextContent('mysql-user');
  });

  it('should render elapsed time with millisecond precision, and 0 as a duration', async () => {
    searchQueries.mockResolvedValue({
      queries: [
        {
          ...TEST_MONGO_DB_QUERY_DATA,
          queryExecutionDuration: '3ms',
          queryId: 'query-ms',
        },
        {
          ...TEST_MONGO_DB_QUERY_DATA,
          queryExecutionDuration: '0s',
          queryId: 'query-zero',
        },
        {
          ...TEST_MONGO_DB_QUERY_DATA,
          queryExecutionDuration: null,
          queryId: 'query-missing',
        },
      ],
    });

    renderComponent();

    await waitFor(() =>
      expect(
        screen.getByTestId('query-query-ms-elapsed-time-cell')
      ).toHaveTextContent('0.003s')
    );
    expect(
      screen.getByTestId('query-query-zero-elapsed-time-cell')
    ).toHaveTextContent('0.000s');
    expect(
      screen.getByTestId('query-query-missing-elapsed-time-cell')
    ).toHaveTextContent('Unavailable');
  });

  it('should hide the transaction control toggle for a MongoDB selection', async () => {
    renderComponent();

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(
      screen.queryByTestId('overview-table-hide-commit-toggle')
    ).not.toBeInTheDocument();
  });

  it('should show the transaction control toggle for a MySQL selection', async () => {
    renderMySqlSelection();

    await waitFor(() =>
      expect(
        screen.getByTestId('overview-table-hide-commit-toggle')
      ).toHaveTextContent('Hide transaction control')
    );
  });

  it('should not offer services of another technology while one is selected', async () => {
    getRunningSessions.mockResolvedValue([
      TEST_REAL_TIME_SESSION,
      TEST_REAL_TIME_SESSION_MYSQL,
    ]);

    renderComponent({
      initialEntry: `/rta/overview?serviceIds=${TEST_REAL_TIME_SESSION_MYSQL.serviceId}`,
    });

    fireEvent.click(await screen.findByTitle('Open'));

    expect(
      await screen.findByTestId(
        `service-option-${TEST_REAL_TIME_SESSION_MYSQL.serviceId}`
      )
    ).not.toHaveAttribute('aria-disabled', 'true');
    expect(
      screen.getByTestId(`service-option-${TEST_REAL_TIME_SESSION.serviceId}`)
    ).toHaveAttribute('aria-disabled', 'true');
  });

  it('should watch only the first technology when the URL names both', async () => {
    getRunningSessions.mockResolvedValue([
      TEST_REAL_TIME_SESSION,
      TEST_REAL_TIME_SESSION_MYSQL,
    ]);

    renderComponent({
      initialEntry: `/rta/overview?serviceIds=${TEST_REAL_TIME_SESSION_MYSQL.serviceId}&serviceIds=${TEST_REAL_TIME_SESSION.serviceId}`,
    });

    await waitFor(() =>
      expect(searchQueries).toHaveBeenLastCalledWith({
        serviceIds: [TEST_REAL_TIME_SESSION_MYSQL.serviceId],
      })
    );
  });

  it('should keep elapsed time pinned without offering pin controls', async () => {
    renderComponent();

    await waitFor(() =>
      screen.getByTestId(`query-${TEST_MONGO_DB_QUERY_DATA.queryId}-host-cell`)
    );

    expect(
      screen.getByTestId(
        `query-${TEST_MONGO_DB_QUERY_DATA.queryId}-elapsed-time-cell`
      )
    ).toHaveAttribute('data-pinned', 'true');

    fireEvent.click(screen.getByLabelText('Show/Hide columns'));

    await waitFor(() => expect(screen.getByRole('menu')).toBeInTheDocument());
    // The icon assertion does not depend on MRT's tooltip labelling, so it still
    // holds if those labels change.
    expect(screen.queryByTestId('PushPinIcon')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Pin to left')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Pin to right')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Unpin')).not.toBeInTheDocument();
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

  it('hides export while live updates are running', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    expect(
      screen.queryByTestId('overview-table-export-button')
    ).not.toBeInTheDocument();
  });

  it('enables export after pausing live updates', async () => {
    renderComponent({
      initialEntry:
        '/rta/overview?serviceIds=' + TEST_REAL_TIME_SESSION.serviceId,
    });

    await waitFor(() => screen.getByTestId('realtime-overview-table'));

    fireEvent.click(screen.getByTestId('overview-table-pause-button'));

    await waitFor(() =>
      expect(
        screen.getByTestId('overview-table-export-button')
      ).not.toBeDisabled()
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
