import { TestWrapper } from 'utils/testWrapper';
import { MainWithNav } from './MainWithNav';
import { render, screen } from '@testing-library/react';
import { wrapWithGrafana, wrapWithQueryProvider } from 'utils/testUtils';

const setup = ({
  isLoading = false,
  isLoggedIn = false,
  kioskModeActive = false,
  search = '',
}: {
  isLoading?: boolean;
  isLoggedIn?: boolean;
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

  return render(
    <TestWrapper authContext={{ isLoading, isLoggedIn }}>
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
});
