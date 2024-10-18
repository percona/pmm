import { wrapWithUpdatesProvider } from 'utils/testUtils';
import { HomeLink } from './HomeLink';
import { fireEvent, render, screen } from '@testing-library/react';
import { UpdateStatus } from 'types/updates.types';
import { PMM_HOME_URL } from 'constants';
import { MemoryRouter } from 'react-router-dom';

describe('HomeLink', () => {
  it('navigates to PMM Home if client update is not pending', () => {
    render(
      wrapWithUpdatesProvider(
        <MemoryRouter initialEntries={[{ pathname: '/updates/clients' }]}>
          <HomeLink data-testid="home-link" />
        </MemoryRouter>,
        {
          status: UpdateStatus.UpToDate,
        }
      )
    );

    expect(screen.getByTestId('home-link')).toHaveAttribute(
      'href',
      PMM_HOME_URL
    );
  });

  it('shows modal if client update is pending', () => {
    render(
      wrapWithUpdatesProvider(
        <MemoryRouter initialEntries={[{ pathname: '/updates/clients' }]}>
          <HomeLink data-testid="home-link" />
        </MemoryRouter>,
        {
          status: UpdateStatus.UpdateClients,
          areClientsUpToDate: false,
        }
      )
    );
    const homeLink = screen.getByTestId('home-link');

    expect(homeLink).not.toHaveAttribute('href', PMM_HOME_URL);

    fireEvent.click(homeLink);

    expect(screen.getByTestId('modal-clients-update-pending')).toBeDefined();
  });

  it("doesn't propmt on pages other than client updates", () => {
    render(
      wrapWithUpdatesProvider(
        <MemoryRouter initialEntries={[{ pathname: '/updates' }]}>
          <HomeLink data-testid="home-link" />
        </MemoryRouter>,
        {
          status: UpdateStatus.UpdateClients,
          areClientsUpToDate: false,
        }
      )
    );

    expect(screen.getByTestId('home-link')).toHaveAttribute(
      'href',
      PMM_HOME_URL
    );
  });
});
