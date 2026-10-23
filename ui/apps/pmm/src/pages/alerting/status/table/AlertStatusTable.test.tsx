import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { buildTableUrlParamKeys } from '@percona/peak-ui';
import { useLocation } from 'react-router-dom';
import { format } from 'date-fns/format';
import { LocalizationProvider } from '@mui/x-date-pickers';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFnsV3';
import { TestWrapper } from 'utils/testWrapper';
import { AlertRow } from '../AlertsPage.types';
import AlertStatusTable from './AlertStatusTable';

const createAlert = (overrides: Partial<AlertRow>): AlertRow => ({
  type: 'alert',
  id: 'alert-1',
  alertName: 'Alert one',
  ruleName: 'Rule one',
  state: 'Alerting',
  nodeId: 'node-1',
  serviceName: 'service-1',
  summary: 'Alert summary',
  labels: {},
  annotations: {},
  expression: 'up == 0',
  age: '1m',
  activeAt: '2026-04-15T12:00:00.000Z',
  silenced: false,
  silencedBy: [],
  silencedAge: '-',
  rawAlert: { labels: {}, annotations: {} },
  ...overrides,
});

const TEST_ROWS = [
  createAlert({
    id: 'alert-1',
    alertName: 'MySQL down',
    nodeId: 'node-1',
    serviceName: 'mysql-1',
    activeAt: '2026-04-15T10:00:00.000Z',
  }),
  createAlert({
    id: 'alert-2',
    alertName: 'High CPU',
    nodeId: 'node-2',
    serviceName: 'mongo-1',
    activeAt: '2026-04-15T14:00:00.000Z',
  }),
];

const LocationSearch = () => (
  <output data-testid="location-search">{useLocation().search}</output>
);

const renderTable = (url = '/alerting/status', rows = TEST_ROWS) =>
  render(
    <TestWrapper routerProps={{ initialEntries: [url] }}>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <AlertStatusTable
          rows={rows}
          onOpenDetail={vi.fn()}
          onNavigableRowsChange={vi.fn()}
        />
        <LocationSearch />
      </LocalizationProvider>
    </TestWrapper>
  );

const filterParam = (columnId: string, value: string) =>
  `${buildTableUrlParamKeys().filterPrefix}${columnId}=${encodeURIComponent(value)}`;

const hasRowWith = (text: string) =>
  screen.getAllByRole('row').some((row) => row.textContent?.includes(text));

const showFilters = () =>
  fireEvent.click(screen.getByRole('button', { name: /show\/hide filters/i }));

describe('AlertStatusTable column filters', () => {
  it('shows datetime pickers for Triggered at and no filter for State', () => {
    renderTable();
    showFilters();

    expect(screen.queryByLabelText(/filter by state/i)).toBeNull();
    expect(screen.getByLabelText(/filter by name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/filter by node/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/filter by service/i)).toBeInTheDocument();
    // The datetime-range variant renders Min/Max date-time pickers.
    expect(screen.getByPlaceholderText('Min')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Max')).toBeInTheDocument();
  });

  it('filters rows by the Triggered at range', async () => {
    renderTable();
    showFilters();

    expect(screen.getByText('MySQL down')).toBeInTheDocument();
    expect(screen.getByText('High CPU')).toBeInTheDocument();

    // Noon UTC in the local timezone: after 'MySQL down' (10:00Z),
    // before 'High CPU' (14:00Z).
    fireEvent.change(screen.getByPlaceholderText('Min'), {
      target: {
        value: format(
          new Date('2026-04-15T12:00:00.000Z'),
          'MM/dd/yyyy hh:mm a'
        ),
      },
    });

    await waitFor(() => expect(screen.queryByText('MySQL down')).toBeNull());
    expect(screen.getByText('High CPU')).toBeInTheDocument();
  });

  it('keeps matching alerts inside node groups when filtering', async () => {
    renderTable();
    fireEvent.click(screen.getByRole('switch', { name: /group by node/i }));
    showFilters();

    fireEvent.change(screen.getByLabelText(/filter by name/i), {
      target: { value: 'MySQL' },
    });

    await waitFor(() => expect(screen.queryByText('High CPU')).toBeNull());
    // The node-1 group row and its matching alert remain. Filter match
    // highlighting splits the alert name across elements, so match on the
    // row's text content.
    expect(screen.getAllByText('node-1').length).toBeGreaterThan(0);
    expect(
      screen
        .getAllByRole('row')
        .some((row) => row.textContent?.includes('MySQL down'))
    ).toBe(true);
    expect(screen.queryByText('node-2')).toBeNull();
  });
});

describe('AlertStatusTable URL filters', () => {
  it('opens prefiltered to the service named in ?f.serviceName=', () => {
    renderTable(`/alerting/status?${filterParam('serviceName', 'mongo-1')}`);

    expect(hasRowWith('High CPU')).toBe(true);
    expect(hasRowWith('MySQL down')).toBe(false);
    expect(screen.getByLabelText(/filter by service/i)).toHaveValue('mongo-1');
  });

  it('writes a column filter to the URL, so a reload keeps it', async () => {
    const { unmount } = renderTable();
    showFilters();

    fireEvent.change(screen.getByLabelText(/filter by service/i), {
      target: { value: 'mongo-1' },
    });

    await waitFor(() =>
      expect(screen.getByTestId('location-search')).toHaveTextContent(
        filterParam('serviceName', 'mongo-1')
      )
    );
    const search = screen.getByTestId('location-search').textContent;
    unmount();

    renderTable(`/alerting/status${search}`);

    expect(hasRowWith('MySQL down')).toBe(false);
    expect(screen.getByLabelText(/filter by service/i)).toHaveValue('mongo-1');
  });

  it('restores a Triggered at range from the URL', () => {
    renderTable(
      `/alerting/status?${filterParam(
        'activeAt',
        JSON.stringify(['2026-04-15T12:00:00.000Z', ''])
      )}`
    );

    expect(hasRowWith('High CPU')).toBe(true);
    expect(hasRowWith('MySQL down')).toBe(false);
  });

  it('keeps unrelated params and still shows 25 rows per page', () => {
    const rows = Array.from({ length: 30 }, (_, index) =>
      createAlert({ id: `alert-${index}`, alertName: `Alert ${index}` })
    );
    renderTable('/alerting/status?from=now-1h', rows);

    expect(screen.getByTestId('location-search')).toHaveTextContent(
      '?from=now-1h'
    );
    expect(screen.getAllByText(/^Alert \d+$/)).toHaveLength(25);
  });
});
