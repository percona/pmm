import { afterEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { NavigationContext } from 'contexts/navigation/navigation.context';
import { wrapWithQueryProvider } from 'utils/testUtils';
import { TestWrapper } from 'utils/testWrapper';
import type { NavItem } from 'types/navigation.types';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Sidebar } from './Sidebar';

const StatefulNavigation = ({
  initialOpen,
  navTree,
}: {
  initialOpen: boolean;
  navTree: NavItem[];
}) => {
  const [navOpen, setNavOpen] = useState(initialOpen);

  return (
    <NavigationContext.Provider value={{ navTree, navOpen, setNavOpen }}>
      <Sidebar />
      {/* Where Grafana takes the logo's /graph/ link. */}
      <Link to="/graph/d/pmm-home/home-dashboard">Home dashboard</Link>
      <Link to="/graph/d/mysql-user-details">User details</Link>
      <Link to="/inventory/nodes">Nodes</Link>
    </NavigationContext.Provider>
  );
};

const renderSidebar = (
  navOpen: boolean,
  navTree: NavItem[] = [],
  initialEntries = ['/']
) =>
  render(
    <TestWrapper routerProps={{ initialEntries }}>
      {wrapWithQueryProvider(
        <StatefulNavigation initialOpen={navOpen} navTree={navTree} />
      )}
    </TestWrapper>
  );

const inSidebar = (selector: string) =>
  screen.getByTestId('pmm-sidebar').querySelector(selector) as HTMLElement;

describe('Sidebar', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('closes after a click on a narrow layout', () => {
    vi.stubGlobal('matchMedia', (query: string) => ({
      matches: query.includes('max-width'),
      media: query,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
    renderSidebar(true, [{ id: 'overview', text: 'Overview', url: '/x' }]);

    fireEvent.click(screen.getByRole('link', { name: 'Overview' }));

    expect(
      screen.getByRole('button', { name: 'Expand sidebar' })
    ).toBeInTheDocument();
  });

  it('lays the list out at the open width only while the drawer moves', () => {
    renderSidebar(true, [{ id: 'overview', text: 'Overview', url: '/x' }]);

    fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }));

    expect(getComputedStyle(inSidebar('ul')).width).toBe('256px');

    fireEvent.transitionEnd(inSidebar('.MuiDrawer-paper'), {
      propertyName: 'width',
    });

    expect(getComputedStyle(inSidebar('ul')).width).toBe('100%');
  });

  it.each([
    ['the home dashboard', 'Home dashboard'],
    ['a dashboard with no entry', 'User details'],
  ])('clears the highlight on %s', (_, link) => {
    renderSidebar(
      true,
      [{ id: 'mysql', text: 'MySQL', url: '/graph/d/mysql' }],
      ['/graph/d/mysql']
    );
    const entry = screen.getByRole('link', { name: 'MySQL' });

    expect(entry).toHaveClass('Mui-selected');

    fireEvent.click(screen.getByRole('link', { name: link }));

    expect(entry).not.toHaveClass('Mui-selected');
  });

  it('keeps the highlight on a page with no entry outside dashboards', () => {
    renderSidebar(
      true,
      [{ id: 'mysql', text: 'MySQL', url: '/graph/d/mysql' }],
      ['/graph/d/mysql']
    );

    fireEvent.click(screen.getByRole('link', { name: 'Nodes' }));

    expect(screen.getByRole('link', { name: 'MySQL' })).toHaveClass(
      'Mui-selected'
    );
  });
});
