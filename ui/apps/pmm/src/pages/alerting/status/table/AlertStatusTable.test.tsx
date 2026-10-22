import { fireEvent, render, screen, waitFor } from '@testing-library/react';
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
    activeAt: '2026-04-15T10:00:00.000Z',
  }),
  createAlert({
    id: 'alert-2',
    alertName: 'High CPU',
    nodeId: 'node-2',
    activeAt: '2026-04-15T14:00:00.000Z',
  }),
];

const renderTable = () =>
  render(
    <TestWrapper>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <AlertStatusTable
          rows={TEST_ROWS}
          onOpenDetail={vi.fn()}
          onNavigableRowsChange={vi.fn()}
        />
      </LocalizationProvider>
    </TestWrapper>
  );

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
