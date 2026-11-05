import { fireEvent, render, screen, within } from '@testing-library/react';
import { discoverRDS } from 'api/services';
import { DiscoverRDSEngine } from 'types/services.types';
import { Route, Routes, useLocation } from 'react-router-dom';
import { wrapWithQueryProvider, wrapWithSettings } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { TEST_USER_VIEWER } from 'utils/testStubs';
import { SettingsContext } from 'contexts/settings/settings.context';
import { AddServicePage } from './AddServicePage';
import { mockNodes } from './service-form/ServiceForm.testUtils';

vi.mock('api/nodes', () => ({ listNodes: vi.fn() }));
vi.mock('api/services', () => ({
  addService: vi.fn(),
  addAzureService: vi.fn(),
  discoverRDS: vi.fn().mockResolvedValue({ rdsInstances: [] }),
  discoverAzure: vi.fn(),
}));

const LocationProbe = () => (
  <span data-testid="location-probe">{useLocation().pathname}</span>
);

const renderPage = (
  path: string,
  {
    azurediscoverEnabled = false,
    isLoading = false,
    user = undefined as typeof TEST_USER_VIEWER | undefined,
  } = {}
) =>
  render(
    <TestWrapper
      routerProps={{ initialEntries: [path] }}
      {...(user && { userContext: { isLoading: false, user } })}
    >
      {wrapWithQueryProvider(
        wrapWithSettings(
          <>
            <LocationProbe />
            <Routes>
              <Route path="/add-instance/:type?" element={<AddServicePage />} />
            </Routes>
          </>,
          { isLoading, settings: { azurediscoverEnabled } }
        )
      )}
    </TestWrapper>
  );

const cardIds = () =>
  screen
    .getAllByTestId(/-instance$/)
    .map((card) => card.getAttribute('data-testid'));

describe('AddServicePage', () => {
  beforeEach(() => {
    mockNodes();
  });

  it('lists the service types in the original order', () => {
    renderPage('/add-instance');

    expect(
      screen.getByRole('heading', { name: 'Select service type' })
    ).toBeInTheDocument();
    expect(cardIds()).toEqual([
      'rds-instance',
      'mysql-instance',
      'mongodb-instance',
      'postgresql-instance',
      'proxysql-instance',
      'haproxy-instance',
      'valkey-instance',
      'external-instance',
    ]);
  });

  it('offers Azure only when Azure discovery is enabled', () => {
    renderPage('/add-instance', { azurediscoverEnabled: true });

    expect(cardIds().at(-1)).toBe('azure-instance');
  });

  it('describes each service type', () => {
    renderPage('/add-instance');

    expect(screen.getByTestId('mysql-instance')).toHaveTextContent(
      'MySQL, Percona Server for MySQL or MariaDB'
    );
    expect(screen.queryByTestId('add-service-back')).not.toBeInTheDocument();
  });

  it('goes back to the discovered instances from an instance form', async () => {
    vi.mocked(discoverRDS).mockResolvedValueOnce({
      rdsInstances: [
        {
          region: 'eu-west-1',
          az: 'eu-west-1a',
          instanceId: 'rds-mysql',
          nodeModel: 'db.t3.micro',
          address: 'rds-mysql.rds.amazonaws.com',
          port: 3306,
          engine: DiscoverRDSEngine.mysql,
          engineVersion: '8.0.35',
        },
      ],
    });
    renderPage('/add-instance/rds');

    fireEvent.click(await screen.findByTestId('start-monitoring-rds-mysql'));
    const back = screen.getByTestId('add-service-back');
    expect(back).toHaveTextContent('Discovered instances');

    fireEvent.click(back);

    expect(
      await screen.findByTestId('start-monitoring-rds-mysql')
    ).toBeInTheDocument();
    expect(screen.getByTestId('add-service-back')).toHaveTextContent(
      'Select service type'
    );
  });

  it('links each card to its form', () => {
    renderPage('/add-instance');

    expect(
      within(screen.getByTestId('mysql-instance')).getByRole('link')
    ).toHaveAttribute('href', '/add-instance/mysql');
  });

  it('cancels back to the inventory from the selection', () => {
    renderPage('/add-instance');

    expect(screen.getByTestId('add-service-cancel')).toHaveAttribute(
      'href',
      '/graph/inventory/services'
    );
  });

  it.each([
    ['mysql', 'Configuring MySQL service'],
    ['mongodb', 'Configuring MongoDB service'],
    ['postgresql', 'Configuring PostgreSQL service'],
    ['proxysql', 'Configuring ProxySQL service'],
    ['haproxy', 'Configuring HAProxy service'],
    ['valkey', 'Configuring Valkey service'],
    ['external', 'Add external service'],
  ])('opens the /add-instance/%s form directly', (type, title) => {
    renderPage(`/add-instance/${type}`);

    expect(screen.getByRole('heading', { name: title })).toBeInTheDocument();
    expect(screen.getByTestId('add-service-form')).toBeInTheDocument();
    expect(screen.queryByTestId('mysql-instance')).not.toBeInTheDocument();
    expect(screen.getByTestId('add-service-cancel')).toHaveAttribute(
      'href',
      '/add-instance'
    );
    expect(screen.getByTestId('add-service-back')).toHaveAttribute(
      'href',
      '/add-instance'
    );
  });

  it('redirects an unknown service type to the selection', () => {
    renderPage('/add-instance/mariadb');

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      /^\/add-instance$/
    );
  });

  it('redirects Azure to the selection when Azure discovery is disabled', () => {
    renderPage('/add-instance/azure');

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      /^\/add-instance$/
    );
  });

  it('keeps the Azure link until the settings arrive', () => {
    render(
      <TestWrapper routerProps={{ initialEntries: ['/add-instance/azure'] }}>
        {wrapWithQueryProvider(
          <SettingsContext.Provider
            value={{ isLoading: false, settings: null }}
          >
            <LocationProbe />
            <Routes>
              <Route path="/add-instance/:type?" element={<AddServicePage />} />
            </Routes>
          </SettingsContext.Provider>
        )}
      </TestWrapper>
    );

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/add-instance/azure'
    );
  });

  it('keeps the Azure link while the settings are loading', () => {
    renderPage('/add-instance/azure', { isLoading: true });

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/add-instance/azure'
    );
  });

  it.each([
    ['rds', 'Amazon RDS', {}],
    [
      'azure',
      'Microsoft Azure MySQL or PostgreSQL',
      { azurediscoverEnabled: true },
    ],
  ])('titles the /add-instance/%s discovery page', (type, title, options) => {
    renderPage(`/add-instance/${type}`, options);

    expect(screen.getByRole('heading', { name: title })).toBeInTheDocument();
  });

  it('opens Azure when Azure discovery is enabled', () => {
    renderPage('/add-instance/azure', { azurediscoverEnabled: true });

    expect(
      screen.getByRole('heading', {
        name: 'Microsoft Azure MySQL or PostgreSQL',
      })
    ).toBeInTheDocument();
  });

  it('is only available to admins', () => {
    renderPage('/add-instance', { user: TEST_USER_VIEWER });

    expect(screen.getByTestId('unauthorized')).toHaveTextContent(
      'Insufficient access permissions.'
    );
    expect(screen.queryByTestId('mysql-instance')).not.toBeInTheDocument();
  });
});
