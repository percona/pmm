import { screen, render, fireEvent, waitFor } from '@testing-library/react';
import {
  wrapWithQueryProvider,
  wrapWithRouter,
  wrapWithUserProvider,
} from 'utils/testUtils';
import {
  TEST_SERVICES,
  TEST_SERVICES_WITH_ONE_MYSQL,
  TEST_USER_ADMIN,
  TEST_USER_EDITOR,
  TEST_USER_VIEWER,
} from 'utils/testStubs';
import { User } from 'types/user.types';
import WelcomeCard from './WelcomeCard';

const mocks = vi.hoisted(() => ({
  startTourMock: vi.fn(),
  updateUserInfo: vi.fn(),
  listServices: vi.fn(() => Promise.resolve(TEST_SERVICES)),
}));

vi.mock('contexts/tour', () => ({
  useTour: () => ({
    startTour: mocks.startTourMock,
  }),
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
    mocks.startTourMock.mockClear();
    mocks.updateUserInfo.mockClear();
    mocks.listServices.mockClear();
  });

  it('shows tour and add service buttons for admin', async () => {
    renderWelcomeCard();

    expect(mocks.listServices).toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-start-tour')
      ).toBeInTheDocument()
    );

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
        screen.queryByTestId('welcome-card-start-tour')
      ).toBeInTheDocument()
    );

    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-add-service')
      ).not.toBeInTheDocument()
    );
  });

  it('shows just tour button for editor', async () => {
    renderWelcomeCard(TEST_USER_EDITOR);

    expect(mocks.listServices).not.toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-start-tour')
      ).toBeInTheDocument()
    );
    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-add-service')
      ).not.toBeInTheDocument()
    );
  });

  it('shows just tour button for viewer', async () => {
    renderWelcomeCard(TEST_USER_VIEWER);

    expect(mocks.listServices).not.toHaveBeenCalled();

    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-start-tour')
      ).toBeInTheDocument()
    );
    await waitFor(() =>
      expect(
        screen.queryByTestId('welcome-card-add-service')
      ).not.toBeInTheDocument()
    );
  });

  it('starts product tour', () => {
    renderWelcomeCard();

    fireEvent.click(screen.getByTestId('welcome-card-start-tour'));

    expect(mocks.startTourMock).toHaveBeenCalledWith('product');
  });

  it('dismisses welcome card', () => {
    renderWelcomeCard();

    fireEvent.click(screen.getByTestId('welcome-card-dismiss'));

    expect(mocks.updateUserInfo).toHaveBeenCalledWith({
      productTourCompleted: true,
    });
  });

  it('does not show welcome card when tour already completed', async () => {
    renderWelcomeCard({
      ...TEST_USER_ADMIN,
      info: {
        ...TEST_USER_ADMIN.info,
        productTourCompleted: true,
      },
    });

    await waitFor(() =>
      expect(screen.queryByTestId('welcome-card')).not.toBeInTheDocument()
    );
  });
});
