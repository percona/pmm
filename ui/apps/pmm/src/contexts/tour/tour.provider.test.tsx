import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { TEST_USER_ADMIN } from 'utils/testStubs';
import { createAnonymousUser } from 'contexts/user/user.utils';
import { addAlerting } from 'contexts/navigation/navigation.utils';
import { User } from 'types/user.types';
import { TourProvider } from './tour.provider';

const mocks = vi.hoisted(() => ({
  user: undefined as User | undefined,
  updateUserInfo: vi.fn(),
}));

vi.mock('contexts/user', () => ({
  useUser: () => ({ user: mocks.user }),
}));

vi.mock('contexts/settings', () => ({
  useSettings: () => ({ settings: {} }),
}));

vi.mock('contexts/navigation', () => ({
  useNavigation: () => ({
    navTree: [addAlerting(true, true, mocks.user)],
    navOpen: true,
    setNavOpen: vi.fn(),
  }),
}));

vi.mock('hooks/api/useUser', () => ({
  useUpdateUserInfo: () => ({ mutateAsync: mocks.updateUserInfo }),
}));

vi.mock('utils/dom.utils', () => ({
  waitForVisible: () => Promise.resolve(true),
}));

const renderAtAlerting = (user: User) => {
  mocks.user = user;

  return render(
    <MemoryRouter initialEntries={['/alerting/status']}>
      <TourProvider />
    </MemoryRouter>
  );
};

describe('TourProvider', () => {
  beforeEach(() => {
    mocks.updateUserInfo.mockReset();
  });

  it('starts the alerting tour on an alerting page', async () => {
    renderAtAlerting(TEST_USER_ADMIN);

    expect(await screen.findByTestId('tour-step-title')).toHaveTextContent(
      'Alert rules'
    );
  });

  it('does not start the alerting tour for anonymous users', async () => {
    renderAtAlerting(createAnonymousUser());

    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(screen.queryByTestId('tour-step-title')).not.toBeInTheDocument();
  });

  it('stays closed while the saved completion has not reached user info yet', async () => {
    mocks.updateUserInfo.mockResolvedValue({});
    renderAtAlerting(TEST_USER_ADMIN);

    fireEvent.click(await screen.findByTestId('tour-close-button'));

    await waitFor(() =>
      expect(screen.queryByTestId('tour-step-title')).not.toBeInTheDocument()
    );
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(screen.queryByTestId('tour-step-title')).not.toBeInTheDocument();
  });

  it('closes the tour when saving completion fails', async () => {
    mocks.updateUserInfo.mockRejectedValue(new Error('401'));
    renderAtAlerting(TEST_USER_ADMIN);

    fireEvent.click(await screen.findByTestId('tour-close-button'));

    await waitFor(() =>
      expect(screen.queryByTestId('tour-step-title')).not.toBeInTheDocument()
    );
    expect(mocks.updateUserInfo).toHaveBeenCalledWith({
      alertingTourCompleted: true,
    });
  });
});
