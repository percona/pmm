import {
  fireEvent,
  render as renderWithoutRouter,
  screen,
  waitFor,
} from '@testing-library/react';
import { ReactElement } from 'react';
import { MemoryRouter } from 'react-router-dom';
import NavigationHeading from './NavigationHeading';

const toggleFn = vitest.fn();

const render = (ui: ReactElement) =>
  renderWithoutRouter(<MemoryRouter>{ui}</MemoryRouter>);

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

  it('names its expand and collapse buttons', () => {
    const { rerender } = render(
      <NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />
    );

    expect(
      screen.getByRole('button', { name: 'Collapse sidebar' })
    ).toBeInTheDocument();

    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );

    expect(
      screen.getByRole('button', { name: 'Expand sidebar' })
    ).toBeInTheDocument();
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

  it('disables the expand button while it slides away', () => {
    const { rerender } = render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );

    expect(screen.getByTestId('sidebar-open-button')).toBeDisabled();
  });

  it('moves focus to the other toggle after a toggle had it', async () => {
    const { rerender } = render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    screen.getByTestId('sidebar-open-button').focus();
    fireEvent.click(screen.getByTestId('sidebar-open-button'));
    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );

    expect(screen.getByTestId('sidebar-close-button')).toHaveFocus();

    fireEvent.click(screen.getByTestId('sidebar-close-button'));
    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );

    await waitFor(() =>
      expect(screen.getByTestId('sidebar-open-button')).toHaveFocus()
    );
  });

  it('leaves focus alone after a mouse click', () => {
    const { rerender } = render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    screen.getByTestId('sidebar-open-button').focus();
    fireEvent.click(screen.getByTestId('sidebar-open-button'), { detail: 1 });
    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );

    expect(screen.getByTestId('sidebar-close-button')).not.toHaveFocus();
  });

  it('leaves focus alone when the toggle did not have it', () => {
    const { rerender } = render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    fireEvent.click(screen.getByTestId('sidebar-open-button'));
    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );

    expect(screen.getByTestId('sidebar-close-button')).not.toHaveFocus();
  });

  it.each([true, false])(
    "links the logo to the user's home dashboard (open: %s)",
    (sidebarOpen) => {
      render(
        <NavigationHeading
          sidebarOpen={sidebarOpen}
          onToggleSidebar={toggleFn}
        />
      );

      expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute(
        'href',
        '/graph/'
      );
    }
  );

  it('names the logo in a tooltip only while collapsed', async () => {
    const { rerender } = render(
      <NavigationHeading sidebarOpen={false} onToggleSidebar={toggleFn} />
    );

    fireEvent.mouseOver(screen.getByRole('link', { name: 'Home' }));

    expect(await screen.findByRole('tooltip')).toHaveTextContent('Home');

    rerender(
      <MemoryRouter>
        <NavigationHeading sidebarOpen onToggleSidebar={toggleFn} />
      </MemoryRouter>
    );
    fireEvent.mouseOver(screen.getByRole('link', { name: 'Home' }));

    await waitFor(() =>
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    );
  });

  it('reports a click on the logo', () => {
    const onNavigate = vi.fn();
    render(
      <NavigationHeading
        sidebarOpen
        onToggleSidebar={toggleFn}
        onNavigate={onNavigate}
      />
    );

    fireEvent.click(screen.getByRole('link', { name: 'Home' }));

    expect(onNavigate).toHaveBeenCalled();
  });
});
