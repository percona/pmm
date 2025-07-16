import { screen, render } from '@testing-library/react';
import { wrapWithRouter, wrapWithUserProvider } from 'utils/testUtils';
import { TEST_USER_EDITOR, TEST_USER_VIEWER } from 'utils/testStubs';
import { User } from 'types/user.types';
import WelcomeCard from './WelcomeCard';

const renderWelcomeCard = (user?: User) =>
  render(wrapWithRouter(wrapWithUserProvider(<WelcomeCard />, { user })));

describe('WelcomeCard', () => {
  it('shows tour and add service buttons for admin', () => {
    renderWelcomeCard();

    expect(screen.queryByTestId('welcome-card-start-tour')).toBeInTheDocument();
    expect(
      screen.queryByTestId('welcome-card-add-service')
    ).toBeInTheDocument();
  });

  it('shows just tour button for editor', () => {
    renderWelcomeCard(TEST_USER_EDITOR);

    expect(screen.queryByTestId('welcome-card-start-tour')).toBeInTheDocument();
    expect(
      screen.queryByTestId('welcome-card-add-service')
    ).not.toBeInTheDocument();
  });

  it('shows just tour button for viewer', () => {
    renderWelcomeCard(TEST_USER_VIEWER);

    expect(screen.queryByTestId('welcome-card-start-tour')).toBeInTheDocument();
    expect(
      screen.queryByTestId('welcome-card-add-service')
    ).not.toBeInTheDocument();
  });
});
