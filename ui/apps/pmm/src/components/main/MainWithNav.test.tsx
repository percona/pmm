import { TestWrapper } from 'utils/testWrapper';
import { MainWithNav } from './MainWithNav';
import { render, screen } from '@testing-library/react';

const setup = ({
  isLoading = false,
  isLoggedIn = false,
  kioskModeActive = false,
}: {
  isLoading?: boolean;
  isLoggedIn?: boolean;
  kioskModeActive?: boolean;
}) =>
  render(
    <TestWrapper
      authContext={{ isLoading, isLoggedIn }}
      routerProps={{
        initialEntries: kioskModeActive ? ['?kiosk=true'] : ['/'],
      }}
    >
      <MainWithNav />
    </TestWrapper>
  );

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

  it('does not show sidebar when kiosk mode is active', () => {
    setup({ isLoading: false, isLoggedIn: true, kioskModeActive: true });

    expect(screen.queryByTestId('pmm-sidebar')).toBeNull();
  });
});
