import { TestWrapper } from 'utils/testWrapper';
import { MainWithNav } from './MainWithNav';
import { render, screen } from '@testing-library/react';

const setup = ({
  isLoading = false,
  kioskModeActive = false,
}: {
  isLoading?: boolean;
  kioskModeActive?: boolean;
}) =>
  render(
    <TestWrapper
      authContext={{ isLoading }}
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
    setup({ isLoading: false, kioskModeActive: false });

    expect(screen.getByTestId('pmm-sidebar')).toBeInTheDocument();
  });

  it('does not show sidebar when kiosk mode is active', () => {
    setup({ isLoading: false, kioskModeActive: true });

    expect(screen.queryByTestId('pmm-sidebar')).toBeNull();
  });
});
