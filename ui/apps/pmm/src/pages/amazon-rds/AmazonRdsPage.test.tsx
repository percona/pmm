import { fireEvent, render, screen, within } from '@testing-library/react';
import { useLocation } from 'react-router-dom';
import type { User } from 'types/user.types';
import { TEST_USER_ADMIN, TEST_USER_VIEWER } from 'utils/testStubs';
import { TestWrapper } from 'utils/testWrapper';
import { wrapWithSettings } from 'utils/testUtils';
import { ALERT_RULES, RDS_ROWS, SILENCED_ALERTS } from './AmazonRds.mocks';
import { AmazonRdsPage } from './AmazonRdsPage';

const { useRdsInstances, usePrometheusAlertRules, useAlertmanagerAlerts } =
  vi.hoisted(() => ({
    useRdsInstances: vi.fn(),
    usePrometheusAlertRules: vi.fn(),
    useAlertmanagerAlerts: vi.fn(),
  }));

vi.mock('hooks/api/useRds', () => ({
  useRdsInstances,
  RDS_REFETCH_INTERVAL_MS: 60_000,
}));
vi.mock('hooks/api/usePrometheusAlertRules', () => ({
  usePrometheusAlertRules,
}));
vi.mock('hooks/api/useAlertmanagerAlerts', () => ({ useAlertmanagerAlerts }));

const LocationSearch = () => (
  <div data-testid="location-search">{useLocation().search}</div>
);

const renderPage = ({
  search = '',
  user = TEST_USER_ADMIN,
}: { search?: string; user?: User } = {}) =>
  render(
    <TestWrapper
      userContext={{ isLoading: false, user }}
      routerProps={{ initialEntries: [`/amazon-rds${search}`] }}
    >
      {wrapWithSettings(
        <>
          <AmazonRdsPage />
          <LocationSearch />
        </>,
        { settings: { advisorEnabled: true } }
      )}
    </TestWrapper>
  );

const groupRows = () =>
  screen
    .getAllByRole('button', { name: /^Expand (?!all$)/ })
    .map((button) => button.closest('tr')?.textContent);

const shown = (text: string) => screen.getByText(text);

const groupRow = (name: string) =>
  screen
    .getByRole('button', { name: `Expand ${name}` })
    .closest('tr') as HTMLElement;

const locationSearch = () =>
  decodeURIComponent(screen.getByTestId('location-search').textContent ?? '');

// The groups table is a full Material React Table, slow to render under a parallel run.
describe('AmazonRdsPage', { timeout: 20_000 }, () => {
  beforeEach(() => {
    useRdsInstances.mockReturnValue({ data: RDS_ROWS, isError: false });
    usePrometheusAlertRules.mockReturnValue({ data: ALERT_RULES });
    useAlertmanagerAlerts.mockReturnValue({ data: SILENCED_ALERTS });
  });

  it('renders the title, header links and groups collapsed with their aggregates', () => {
    renderPage();

    expect(
      screen.getByRole('heading', { name: 'Amazon RDS' })
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Alerting' })).toHaveAttribute(
      'href',
      '/alerting/status'
    );
    expect(screen.getByRole('link', { name: 'Advisors' })).toHaveAttribute(
      'href',
      '/graph/advisors/insights'
    );
    expect(screen.getByText('database-3')).toBeInTheDocument();
    expect(screen.getAllByText('AWS info unavailable')).toHaveLength(2);
    expect(screen.getAllByText('Aurora cluster')).toHaveLength(3);
    expect(
      screen.getAllByRole('cell', { name: 'Aurora cluster' })
    ).toHaveLength(3);
    expect(screen.getByText('Replication group')).toBeInTheDocument();
    expect(screen.getAllByText('Instance')).toHaveLength(2);
    expect(screen.getAllByText('Unavailable')).toHaveLength(2);
    expect(screen.getByText('Aurora PostgreSQL')).toBeInTheDocument();
    expect(screen.getByText('MariaDB')).toBeInTheDocument();
    expect(screen.getAllByText('1 writer')).toHaveLength(7);
    expect(screen.getByText('unknown')).toBeInTheDocument();
    expect(screen.getByText('24.75 qps')).toBeInTheDocument();
    expect(screen.getByText('17.00 tps')).toBeInTheDocument();
    expect(screen.getByText('4.9 KiB used')).toBeInTheDocument();
    expect(screen.getByText('18 GiB free (min)')).toBeInTheDocument();
    expect(screen.getByText('50 GiB free')).toBeInTheDocument();
    expect(shown('8 of 8 groups')).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: 'database-3-instance-1' })
    ).not.toBeInTheDocument();
  });

  it('names the columns in order', () => {
    renderPage();

    const headers = screen
      .getAllByRole('columnheader')
      .map((header) => header.textContent ?? '');
    const positions = [
      'Region',
      'AWS account',
      'DB identifier',
      'Type',
      'Engine',
      'Alerts',
      'Instances',
      'Writers',
      'Throughput',
      'Storage',
    ].map((name) => headers.findIndex((header) => header.includes(name)));

    expect(positions).not.toContain(-1);
    expect(positions).toEqual([...positions].sort((a, b) => a - b));
    expect(new Set(positions).size).toBe(positions.length);
  });

  it('hides the Advisors link from viewers', () => {
    renderPage({ user: TEST_USER_VIEWER });

    expect(screen.getByRole('link', { name: 'Alerting' })).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: 'Advisors' })
    ).not.toBeInTheDocument();
    expect(screen.getByText('database-3')).toBeInTheDocument();
    expect(
      within(groupRow('database-3')).getByRole('link', {
        name: '2 firing alerts',
      })
    ).toBeInTheDocument();
  });

  it('shows the firing alerts of groups and instances', () => {
    renderPage();

    const database3 = within(groupRow('database-3'));

    expect(
      database3.getByRole('link', { name: '2 firing alerts' })
    ).toHaveAttribute('href', '/alerting/status');
    expect(database3.getByText('2 / 2')).toBeInTheDocument();
    expect(
      within(groupRow('database-2')).getByText('None')
    ).toBeInTheDocument();
    expect(
      database3.getByRole('link', { name: '2 firing alerts' })
    ).toHaveAttribute('target', '_blank');

    fireEvent.click(screen.getByRole('button', { name: 'Expand database-3' }));

    const instances = within(screen.getByRole('table', { name: 'database-3' }));

    expect(
      instances.getAllByRole('link', { name: '1 firing alert' })
    ).toHaveLength(2);
  });

  it('shows no alert value until the alerts load', () => {
    usePrometheusAlertRules.mockReturnValue({ data: undefined });
    renderPage();

    expect(screen.queryByText('None')).not.toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /firing alert/ })
    ).not.toBeInTheDocument();
  });

  it('expands a group into its instances, linking each to its dashboards in a new tab', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Expand database-3' }));

    const instances = screen.getByRole('table', { name: 'database-3' });

    expect(
      within(instances)
        .getAllByRole('row')
        .slice(1)
        .map((row) => row.textContent)
    ).toEqual([
      'database-3-instance-1MySQLOSwriterUp120.50 qps-',
      'database-3-instance-1-readerMySQLOSreaderUp14.25 qps-',
    ]);
    expect(
      within(instances).getByRole('link', {
        name: 'database-3-instance-1-reader',
      })
    ).toHaveAttribute(
      'href',
      '/graph/d/mysql-amazonaurora/mysql-amazon-aurora-details?var-service_name=database-3-instance-1-reader'
    );
    expect(
      within(instances).getByRole('link', {
        name: 'MySQL Instance Summary for database-3-instance-1',
      })
    ).toHaveAttribute(
      'href',
      '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=database-3-instance-1'
    );
    expect(
      within(instances).getByRole('link', {
        name: 'Node Summary for database-3-instance-1',
      })
    ).toHaveAttribute(
      'href',
      '/graph/d/node-instance-summary/node-summary?var-node_name=database-3-instance-1'
    );
    within(instances)
      .getAllByRole('link')
      .forEach((link) => {
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveAttribute('rel', 'noopener noreferrer');
      });
    expect(
      screen.getByRole('button', { name: 'Collapse database-3' })
    ).toHaveAttribute('aria-expanded', 'true');
  });

  it('expands every group at once', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: 'Expand all' }));

    expect(
      screen.getAllByRole('link', { name: /Node Summary for/ })
    ).toHaveLength(12);
    expect(
      screen.getByRole('link', { name: 'database-4-instance-1' })
    ).toHaveAttribute(
      'href',
      '/graph/d/postgresql-instance-summary/postgresql-instance-summary?var-service_name=database-4-instance-1'
    );
    expect(
      screen.getByRole('link', { name: 'rds-mysql-1-replica' })
    ).toHaveAttribute(
      'href',
      '/graph/d/mysql-instance-summary/mysql-instance-summary?var-service_name=rds-mysql-1-replica'
    );
    expect(
      screen.getByText('rds-mysql-1-replica-west, us-west-2')
    ).toBeInTheDocument();
    expect(screen.getByText('Down')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Collapse all' })
    ).toBeInTheDocument();
  });

  it('sorts groups by a column, missing values last', () => {
    renderPage();

    fireEvent.click(screen.getByRole('button', { name: /Sort by Throughput/ }));

    const descending = groupRows();

    expect(descending[0]).toContain('rds-mysql-1');
    expect(descending[7]).toContain('AWS info unavailable');

    fireEvent.click(
      screen.getByRole('button', { name: /Sorted by Throughput/ })
    );

    const ascending = groupRows();

    expect(ascending[0]).toContain('AWS info unavailable');
    expect(ascending[0]).toContain('PostgreSQL');
    expect(ascending[7]).toContain('AWS info unavailable');
  });

  it('searches after typing stops, expands groups matched by service and writes the URL', async () => {
    renderPage();

    fireEvent.change(
      screen.getByRole('textbox', {
        name: 'Search DB identifiers, types, instances, accounts',
      }),
      { target: { value: 'reader' } }
    );

    expect(shown('8 of 8 groups')).toBeInTheDocument();
    expect(
      await screen.findByText('2 of 8 groups', {}, { timeout: 5000 })
    ).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: 'database-3-instance-1-reader' })
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: 'database-3-instance-1' })
    ).not.toBeInTheDocument();
    expect(locationSearch()).toBe('?rds-q=reader');
  });

  it('restores the filters from the URL', () => {
    renderPage({
      search: '?rds-engine=postgres&rds-engine=aurora-postgresql',
    });

    expect(shown('3 of 8 groups')).toBeInTheDocument();
    expect(screen.getByText('database-4')).toBeInTheDocument();
    expect(screen.getByText('rds-pg-1')).toBeInTheDocument();
    expect(screen.queryByText('database-3')).not.toBeInTheDocument();
  });

  it.each([
    ['?rds-type=aurora-cluster', '3 of 8 groups'],
    ['?rds-account=222222222222', '3 of 8 groups'],
    ['?rds-region=us-east-2&rds-id=rds-pg-1', '1 of 8 groups'],
    ['?rds-q=database-3', '1 of 8 groups'],
  ])('filters by %s', (search, count) => {
    renderPage({ search });

    expect(shown(count)).toBeInTheDocument();
  });

  it('writes a selected filter to the URL, dropping identifiers it excludes', () => {
    renderPage({ search: '?rds-id=database-3&rds-id=rds-pg-1' });

    expect(shown('2 of 8 groups')).toBeInTheDocument();

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Region' }));
    fireEvent.click(screen.getByRole('option', { name: /us-east-2/ }));

    expect(locationSearch()).toBe('?rds-region=us-east-2&rds-id=rds-pg-1');
    expect(shown('1 of 8 groups')).toBeInTheDocument();
  });

  it('shows a no-match state that clears the filters', () => {
    renderPage({ search: '?rds-q=no-such-group&rds-engine=mysql' });

    expect(screen.getByText('No groups match the filters')).toBeInTheDocument();
    expect(shown('0 of 8 groups')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Clear filters' }));

    expect(shown('8 of 8 groups')).toBeInTheDocument();
    expect(locationSearch()).toBe('');
  });

  it('shows the loading, error and empty states', () => {
    useRdsInstances.mockReturnValue({ data: undefined, isError: false });
    const { unmount } = renderPage();

    expect(
      screen.getByLabelText('Loading Amazon RDS instances')
    ).toBeInTheDocument();
    unmount();

    useRdsInstances.mockReturnValue({ data: undefined, isError: true });
    const errored = renderPage();

    expect(
      screen.getByText('Could not load the Amazon RDS instances.')
    ).toBeInTheDocument();
    errored.unmount();

    useRdsInstances.mockReturnValue({ data: [], isError: false });
    renderPage();

    expect(
      screen.getByText('No Amazon RDS instances found')
    ).toBeInTheDocument();
  });

  it('keeps showing the data when a refresh fails', () => {
    useRdsInstances.mockReturnValue({ data: RDS_ROWS, isError: true });
    renderPage();

    expect(screen.getByText('database-3')).toBeInTheDocument();
  });
});
