import { TestWrapper } from 'utils/testWrapper';
import { MainWithNav } from './MainWithNav';
import { render, screen } from '@testing-library/react';
import { wrapWithGrafana, wrapWithQueryProvider } from 'utils/testUtils';
import { TEST_USER_ADMIN, TEST_USER_ANONYMOUS } from 'utils/testStubs';
import { User } from 'types/user.types';

const setup = ({
  isLoading = false,
  isLoggedIn = false,
  isAnonymous = false,
  kioskModeActive = false,
  search = '',
}: {
  isLoading?: boolean;
  isLoggedIn?: boolean;
  isAnonymous?: boolean;
  kioskModeActive?: boolean;
  /** URL search string (e.g. "render=1" for Grafana renderer). Prepended with "?" when setting location.search */
  search?: string;
}) => {
  const originalLocation = window.location;
  const searchString = search ? `?${search}` : '';
  Object.defineProperty(window, 'location', {
    value: {
      ...originalLocation,
      search: searchString,
    },
    writable: true,
  });

  let user: User | undefined;
  if (isAnonymous) {
    user = TEST_USER_ANONYMOUS;
  } else if (isLoggedIn) {
    user = TEST_USER_ADMIN;
  }

  return render(
    <TestWrapper
      authContext={{ isLoading, isLoggedIn }}
      userContext={{ isLoading: false, user }}
    >
      {wrapWithQueryProvider(
        wrapWithGrafana(<MainWithNav />, { isFullScreen: kioskModeActive })
      )}
    </TestWrapper>
  );
};

describe('MainWithNav', () => {
  it('shows loading', () => {
    setup({ isLoading: true });

    expect(screen.queryByTestId('pmm-loading-indicator')).not.toBeNull();
  });

  it("doesn't show loading", () => {
    setup({ isLoading: false });

    expect(screen.queryByTestId('pmm-loading-indicator')).toBeNull();
  });

  it('shows sidebar when kiosk mode is not active', () => {
    setup({ isLoading: false, isLoggedIn: true, kioskModeActive: false });

    expect(screen.getByTestId('pmm-sidebar')).toBeInTheDocument();
  });

  it('does not show sidebar when not logged in', () => {
    setup({ isLoading: false, isLoggedIn: false, kioskModeActive: false });

    expect(screen.queryByTestId('pmm-sidebar')).toBeNull();
  });

  it('does not show sidebar when kiosk mode is active', () => {
    setup({ isLoading: false, isLoggedIn: true, kioskModeActive: true });

    expect(screen.queryByTestId('pmm-sidebar')).toBeNull();
  });

  it('hides sidebar so the renderer gets a minimal layout', () => {
    setup({
      isLoading: false,
      isLoggedIn: true,
      kioskModeActive: false,
      search: 'render=1',
    });

    expect(screen.queryByTestId('pmm-sidebar')).toBeNull();
  });

  it('shows sidebar when not in renderer mode', () => {
    setup({ isLoading: false, isLoggedIn: true, kioskModeActive: false });

    expect(screen.getByTestId('pmm-sidebar')).toBeInTheDocument();
  });

  it('shows sidebar when user is anonymous', () => {
    setup({ isLoading: false, isLoggedIn: false, isAnonymous: true });

    expect(screen.getByTestId('pmm-sidebar')).toBeInTheDocument();
  });
});
