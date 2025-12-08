import { fireEvent, render, screen } from '@testing-library/react';
import NavigationHeading from './NavigationHeading';

const toggleFn = vitest.fn();

describe('NavigationHeading', () => {
  beforeEach(() => {
    toggleFn.mockClear();
  });

  it('if sidebars is open only close button is available', () => {
    render(<NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />);

    expect(screen.queryByTestId('sidebar-close-button')).toBeInTheDocument();
    expect(screen.queryByTestId('sidebar-open-button')).not.toBeInTheDocument();
  });

  it('if sidebars is closed only open button is available', () => {
    render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    expect(
      screen.queryByTestId('sidebar-close-button')
    ).not.toBeInTheDocument();
    expect(screen.queryByTestId('sidebar-open-button')).toBeInTheDocument();
  });

  it('can close sidebar', () => {
    render(<NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />);

    fireEvent.click(screen.getByTestId('sidebar-close-button'));

    expect(toggleFn).toHaveBeenCalled();
  });

  it('can open sidebar', () => {
    render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    fireEvent.click(screen.getByTestId('sidebar-open-button'));

    expect(toggleFn).toHaveBeenCalled();
  });
});
