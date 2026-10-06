import { render, screen, within } from '@testing-library/react';
import { Route, Routes, useLocation } from 'react-router-dom';
import { wrapWithSettings } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import { TEST_USER_VIEWER } from 'utils/testStubs';
import { AddServicePage } from './AddServicePage';

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
      {wrapWithSettings(
        <>
          <LocationProbe />
          <Routes>
            <Route path="/add-instance/:type?" element={<AddServicePage />} />
          </Routes>
        </>,
        { isLoading, settings: { azurediscoverEnabled } }
      )}
    </TestWrapper>
  );

const cardIds = () =>
  screen
    .getAllByTestId(/-instance$/)
    .map((card) => card.getAttribute('data-testid'));

describe('AddServicePage', () => {
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
    ['valkey', 'Configuring Valkey service'],
    ['external', 'Add external service'],
    ['rds', 'Amazon RDS'],
  ])('opens /add-instance/%s directly', (type, title) => {
    renderPage(`/add-instance/${type}`);

    expect(screen.getByRole('heading', { name: title })).toBeInTheDocument();
    expect(screen.queryByTestId('mysql-instance')).not.toBeInTheDocument();
    expect(screen.getByTestId('add-service-cancel')).toHaveAttribute(
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

  it('keeps the Azure link while the settings are loading', () => {
    renderPage('/add-instance/azure', { isLoading: true });

    expect(screen.getByTestId('location-probe')).toHaveTextContent(
      '/add-instance/azure'
    );
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
