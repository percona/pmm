import { screen, render, waitFor } from '@testing-library/react';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  TEST_SERVICES,
  TEST_SERVICES_WITH_ONE_MYSQL,
} from 'utils/testStubs';
import { User } from 'types/user.types';
import WelcomeCard from './WelcomeCard';

const mocks = vi.hoisted(() => ({
  updateUserInfo: vi.fn(),
  listServices: vi.fn(() => Promise.resolve(TEST_SERVICES)),
}));

vi.mock('hooks/api/useUser', () => ({
  useUpdateUserInfo: () => ({
    mutate: mocks.updateUserInfo,
  }),
}));

vi.mock('api/services', () => ({
  listServices: () => mocks.listServices(),
}));

const renderWelcomeCard = (user?: User) =>
  render(
    wrapWithQueryProvider(
      wrapWithRouter(wrapWithUserProvider(<WelcomeCard />, { user }))
    )
  );

describe('WelcomeCard', () => {
  beforeEach(() => {
    mocks.updateUserInfo.mockClear();
    mocks.listServices.mockClear();
  });

  it('shows add service button for admin', async () => {
    renderWelcomeCard();

    expect(mocks.listServices).toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-add-service')
      ).toBeInTheDocument()
    );
  });

  it('hides add service button when services exist', async () => {
    mocks.listServices.mockReturnValueOnce(
      Promise.resolve(TEST_SERVICES_WITH_ONE_MYSQL)
    );

    renderWelcomeCard();

    expect(mocks.listServices).toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-add-service')
      ).not.toBeInTheDocument()
    );
  });
});
